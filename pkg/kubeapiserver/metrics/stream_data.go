package metrics

import (
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"sort"
	"sync"
	"time"
)

type MetricsDatabase interface {
	// MetricsDatabase接口
	// 保存metrics
	SavePodMetrics(podMetrics []*v1.PodRawMetrics) error
	// 获取某个Pod在metrics的timeStamp前n秒内的记录
	GetPodMetrics(podId v1.UID, timeStamp time.Time, n int) ([]*v1.PodRawMetrics, error)

	// 锁的实现应该隐藏在内部
}

const (
	DefaultWindowSize int = 60
)

type metricsDb struct {
	// 保存metrics的map
	metrics map[v1.UID]v1.PodRawMetrics
	// 读写锁
	rwLock sync.RWMutex
}

func NewMetricsDb() (*metricsDb, error) {
	return (&metricsDb{
		metrics: make(map[v1.UID]v1.PodRawMetrics),
	}), nil
}

// // 初始化map
// func (db *MetricsDb) Init() {
// 	db.metrics =make(map[v1.UID]v1.PodRawMetrics,0)
// }

func (db *metricsDb) SavePodMetrics(podMetrics []*v1.PodRawMetrics) error {
	// 先上写锁
	db.rwLock.Lock()
	defer db.rwLock.Unlock()
	// 根据ContainerInfo中的键值对，保存metrics

	// 对于podMetrics中的每一个podId:
	for _, podItem := range podMetrics {
		// 如果podId不存在，则直接插入
		if _, ok := db.metrics[podItem.UID]; !ok {
			fmt.Printf("podId %v not found, insert directly\n", podItem.UID)
			fmt.Printf("podItem: %v\n", podItem)
			db.metrics[podItem.UID] = *podItem
			fmt.Printf("len of metrics: %v\n", len(db.metrics))
		} else {
			// 如果podId存在，则对ContainerInfo进行处理
			for cName, cInfo := range podItem.ContainerInfo {
				// 信任containerInfo中对齐，直接append
				db.metrics[podItem.UID].ContainerInfo[cName] = append(db.metrics[podItem.UID].ContainerInfo[cName], cInfo...)
			}
		}
		// 检查过长
		for cName := range podItem.ContainerInfo {
			// 如果超过窗口大小，删除最早的
			curLen := len(db.metrics[podItem.UID].ContainerInfo[cName])

			if curLen > DefaultWindowSize {
				fmt.Printf("podId %v container %v has too many records, delete the earliest\n", podItem.UID, cName)
				db.metrics[podItem.UID].ContainerInfo[cName] = db.metrics[podItem.UID].ContainerInfo[cName][curLen-DefaultWindowSize:]
			}
		}
	}

	// 先假设不会出错
	return nil
}

func (db *metricsDb) GetPodMetrics(podId v1.UID, timeStamp time.Time, n int) ([]*v1.PodRawMetrics, error) {
	// 上读锁
	db.rwLock.RLock()
	defer db.rwLock.RUnlock()

	fmt.Printf("len of metrics: %v\n", len(db.metrics))
	for k, v := range db.metrics {
		fmt.Printf("key: %v, value: %v\n", k, v)
	}
	fmt.Printf("podId: %v\n", podId)

	if podMetrics, ok := db.metrics[podId]; ok {

		// 由时间戳来判断是否在n秒内 metrics是按照时间戳递增的
		if len(podMetrics.ContainerInfo) == 0 {
			return nil, fmt.Errorf("containerInfo is empty")
		}
		fmt.Printf("参数获取到的时间戳:%v\n", timeStamp)
		earliestValidTS := timeStamp.Add(-time.Duration(n) * time.Second)
		fmt.Printf("需要查询到的最早的时间戳:%v\n", earliestValidTS)
		for cName, containerInfo := range podMetrics.ContainerInfo {
			// 找到比earliestValidTS大的最小的index
			l := sort.Search(len(containerInfo), func(i int) bool {
				return containerInfo[i].TimeStamp.After(earliestValidTS)
			})
			r := sort.Search(len(containerInfo), func(i int) bool {
				return containerInfo[i].TimeStamp.After(timeStamp)
			})

			podMetrics.ContainerInfo[cName] = containerInfo[l:r]
		}

		return []*v1.PodRawMetrics{&podMetrics}, nil

	} else {
		return nil, fmt.Errorf("podId not found")
	}
}
