package runtime

import (
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"sync"
	"time"
)

type Cache interface {
	// 非阻塞获取
	Get(v1.UID) (*PodStatus, error)
	// 如果数据比缓存新，则更新缓存
	Set(v1.UID, *PodStatus, error, time.Time) (updated bool)
	// 阻塞获取
	GetNewerThan(v1.UID, time.Time) (*PodStatus, error)
	Delete(v1.UID)
	UpdateTime(time.Time)
}

type cacheData struct {
	// Pod的状态
	status *PodStatus
	// 获取Pod状态时遇到的错误
	err error
	// 最后一次更新时间
	modified time.Time
}

type cache struct {
	lock sync.RWMutex
	pods map[v1.UID]*cacheData
	// 所有缓存内容至少比timestamp新。
	timestamp *time.Time
}

func NewCache() Cache {
	return &cache{pods: map[v1.UID]*cacheData{}}
}

func (c *cache) Get(id v1.UID) (*PodStatus, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	data, ok := c.pods[id]
	if !ok {
		return nil, fmt.Errorf("cache: pod %v not found", id)
	}
	return data.status, data.err
}

func (c *cache) GetNewerThan(id v1.UID, minTime time.Time) (*PodStatus, error) {
	c.lock.RLock()
	maxRetry := 3
	for i := 0; i < maxRetry; i++ {
		data, ok := c.pods[id]
		isGlobalNewer := c.timestamp != nil && (c.timestamp.After(minTime) || c.timestamp.Equal(minTime))
		if !ok && isGlobalNewer {
			// should not be here
			c.lock.RUnlock()
			return nil, fmt.Errorf("cache: global timestamp is newer but pod %v not exist", id)
		}
		if ok && (data.modified.After(minTime) || data.modified.Equal(minTime)) {
			c.lock.RUnlock()
			return data.status, data.err
		}
		// 可以期待之后会有更新
		c.lock.RUnlock()
		if i == maxRetry-1 {
			break
		}
		time.Sleep(1 * time.Second)
		c.lock.RLock()
	}
	// BUGFIX: 一次Lifecycle可能包含多个容器事件，但一个pod只做一次update cache，
	// 第一个事件被worker处理后，worker时间戳新于cache时间戳，后续事件处理时到这里全部阻塞
	// 若cache后面不会被更新（例如pod long running），则会永久阻塞
	return nil, fmt.Errorf("cache: pod %v newer than %v not found after %vs", id, minTime, maxRetry-1)
}

func (c *cache) Set(id v1.UID, status *PodStatus, err error, timestamp time.Time) (updated bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if data, ok := c.pods[id]; ok && data.modified.After(timestamp) {
		return false
	}
	c.pods[id] = &cacheData{status: status, err: err, modified: timestamp}
	return true
}

func (c *cache) Delete(id v1.UID) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.pods, id)
}

func (c *cache) UpdateTime(timestamp time.Time) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.timestamp = &timestamp
}
