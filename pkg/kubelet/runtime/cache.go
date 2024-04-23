package runtime

import (
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
		return &PodStatus{ID: id}, nil
	}
	return data.status, data.err
}

func (c *cache) GetNewerThan(id v1.UID, minTime time.Time) (*PodStatus, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for {
		data, ok := c.pods[id]
		isGlobalNewer := c.timestamp != nil && c.timestamp.After(minTime)
		if !ok && isGlobalNewer {
			return &PodStatus{ID: id}, nil
		}
		if ok && data.modified.After(minTime) {
			return data.status, data.err
		}
		c.lock.RUnlock()
		time.Sleep(1 * time.Second)
		c.lock.RLock()
	}
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
