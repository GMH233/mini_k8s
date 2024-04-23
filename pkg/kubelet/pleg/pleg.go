package pleg

import (
	"context"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/runtime"
	"sync"
	"time"
)

type PodLifeCycleEventType string

const (
	// ContainerStarted any -> running
	ContainerStarted PodLifeCycleEventType = "ContainerStarted"
	// ContainerDied any -> exited
	ContainerDied PodLifeCycleEventType = "ContainerDied"
	// ContainerRemoved exited -> non-existent
	ContainerRemoved PodLifeCycleEventType = "ContainerRemoved"
	// ContainerChanged any -> unknown
	ContainerChanged PodLifeCycleEventType = "ContainerChanged"
)

const (
	RelistPeriod time.Duration = 5 * time.Second
)

type PodLifecycleEvent struct {
	// Pod ID
	PodId v1.UID
	// 事件类型
	Type PodLifeCycleEventType
}

type PodLifecycleEventGenerator interface {
	Start()
	Watch() chan *PodLifecycleEvent
	Stop()
}

type pleg struct {
	runtimeManager runtime.RuntimeManager
	eventCh        chan *PodLifecycleEvent
	podRecords     map[v1.UID]*podRecord
	cache          runtime.Cache

	// 用于同步停止
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	cacheLock sync.Mutex
}

type podRecord struct {
	old     *runtime.Pod
	current *runtime.Pod
}

func NewPLEG(runtimeManager runtime.RuntimeManager, cache runtime.Cache) PodLifecycleEventGenerator {
	ctx, cancel := context.WithCancel(context.Background())
	return &pleg{
		runtimeManager: runtimeManager,
		eventCh:        make(chan *PodLifecycleEvent, 100),
		podRecords:     make(map[v1.UID]*podRecord),
		cache:          cache,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (p *pleg) Start() {
	log.Printf("Starting PLEG...")
	p.wg.Add(1)
	go func() {
		timer := time.NewTimer(RelistPeriod)
		isStopped := false
		for !isStopped {
			select {
			case <-timer.C:
				p.Relist()
			case <-p.ctx.Done():
				isStopped = true
			}
		}
		log.Printf("PLEG stopped.")
		p.wg.Done()
	}()
}

func (p *pleg) Watch() chan *PodLifecycleEvent {
	return p.eventCh
}

func (p *pleg) Relist() {
	log.Printf("PLEG: Relisting...")
	// TODO: Relist
	ts := time.Now()

	// 获取runtime pods
	// pods, err := p.runtimeManager.GetPods()
	// if err != nil {
	// 	log.Printf("PLEG: Failed to get pods: %v", err)
	// 	return
	// }
	var pods []*runtime.Pod
	p.setCurrentPods(pods)

	eventMap := make(map[v1.UID][]*PodLifecycleEvent)
	for id := range p.podRecords {
		oldPod := p.getOldPodById(id)
		currentPod := p.getCurrentPodById(id)
		events := computeEvents(oldPod, currentPod)
		// 若一个pod没有任何事件，此次relist无需处理
		if len(events) != 0 {
			eventMap[id] = events
		}
	}

	for id, events := range eventMap {
		// 更新全局缓存
		currentPod := p.getCurrentPodById(id)
		if err, _ := p.updateCache(currentPod, id); err != nil {
			log.Printf("PLEG: Failed to update cache for pod %v: %v", id, err)
			// TODO reinspect in next relist
			continue
		}
		// 把current移入old，准备下一次relist
		p.shiftCurrentToOldById(id)
		// 发送事件
		for i := range events {
			if events[i].Type == ContainerChanged {
				continue
			}
			p.eventCh <- events[i]
		}
	}
	p.cache.UpdateTime(ts)
	log.Printf("PLEG: Relisting done.")
}

func (p *pleg) Stop() {
	log.Printf("Stopping PLEG...")
	p.cancel()
	p.wg.Wait()
}

func (p *pleg) getOldPodById(id v1.UID) *runtime.Pod {
	if record, ok := p.podRecords[id]; ok {
		return record.old
	}
	return nil
}

func (p *pleg) getCurrentPodById(id v1.UID) *runtime.Pod {
	if record, ok := p.podRecords[id]; ok {
		return record.current
	}
	return nil
}

func (p *pleg) setCurrentPods(pods []*runtime.Pod) {
	for id := range p.podRecords {
		p.podRecords[id].current = nil
	}
	for _, pod := range pods {
		id := pod.ID
		record, ok := p.podRecords[id]
		if !ok {
			record = &podRecord{}
			p.podRecords[id] = record
		}
		record.current = pod
	}
}

func computeEvents(oldPod, currentPod *runtime.Pod) []*PodLifecycleEvent {
	var id v1.UID
	if oldPod != nil {
		id = oldPod.ID
	} else if currentPod != nil {
		id = currentPod.ID
	}

	// 获取新旧pod所有容器
	cidMap := make(map[string]struct{})
	if oldPod != nil {
		for _, c := range oldPod.Containers {
			cidMap[c.ID] = struct{}{}
		}
	}
	if currentPod != nil {
		for _, c := range currentPod.Containers {
			cidMap[c.ID] = struct{}{}
		}
	}

	events := make([]*PodLifecycleEvent, 0)
	for cid := range cidMap {
		oldState := getPlegContainerState(oldPod, &cid)
		currentState := getPlegContainerState(currentPod, &cid)
		if oldState == currentState {
			continue
		}
		switch currentState {
		case plegContainerRunning:
			events = append(events, &PodLifecycleEvent{PodId: id, Type: ContainerStarted})
		case plegContainerExited:
			events = append(events, &PodLifecycleEvent{PodId: id, Type: ContainerDied})
		case plegContainerUnknown:
			events = append(events, &PodLifecycleEvent{PodId: id, Type: ContainerChanged})
		case plegContainerNonExistent:
			if oldState == plegContainerExited {
				events = append(events, &PodLifecycleEvent{PodId: id, Type: ContainerRemoved})
			} else {
				events = append(events,
					&PodLifecycleEvent{PodId: id, Type: ContainerDied},
					&PodLifecycleEvent{PodId: id, Type: ContainerRemoved})
			}
		}
	}

	return events
}

type plegContainerState string

const (
	plegContainerRunning     plegContainerState = "running"
	plegContainerExited      plegContainerState = "exited"
	plegContainerUnknown     plegContainerState = "unknown"
	plegContainerNonExistent plegContainerState = "non-existent"
)

func getPlegContainerState(pod *runtime.Pod, cid *string) plegContainerState {
	if pod == nil {
		return plegContainerNonExistent
	}
	for _, c := range pod.Containers {
		if c.ID == *cid {
			switch c.State {
			case runtime.ContainerStateCreated:
				return plegContainerUnknown
			case runtime.ContainerStateRunning:
				return plegContainerRunning
			case runtime.ContainerStateExited:
				return plegContainerExited
			case runtime.ContainerStateUnknown:
				return plegContainerUnknown
			}
		}
	}
	return plegContainerNonExistent
}

func (p *pleg) updateCache(currentPod *runtime.Pod, pid v1.UID) (error, bool) {
	// Relist与UpdateCache接口存在并发调用
	p.cacheLock.Lock()
	defer p.cacheLock.Unlock()

	// pid对应的pod存在事件，同时current为nil，直接删除
	if currentPod == nil {
		p.cache.Delete(pid)
		return nil, true
	}
	ts := time.Now()
	// status, err := p.runtimeManager.GetPodStatus(currentPod.ID, currentPod.Name, currentPod.Namespace)
	var status *runtime.PodStatus
	var err error
	// if err == nil {
	// 源码这里为status保留了旧ip地址，暂时不知道为什么，先不管
	// status.IPs = p.getPodIPs(pid, status)
	// }
	return err, p.cache.Set(currentPod.ID, status, err, ts)
}

func (p *pleg) shiftCurrentToOldById(id v1.UID) {
	record, ok := p.podRecords[id]
	if !ok {
		return
	}
	// 若current为nil，shift后就是两个nil，直接删除
	if record.current == nil {
		delete(p.podRecords, id)
		return
	}
	record.old = record.current
	record.current = nil
}
