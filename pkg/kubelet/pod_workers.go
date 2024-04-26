package kubelet

import (
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/runtime"
	"minikubernetes/pkg/kubelet/types"
	"sync"
	"time"
)

type PodWorkerState string

const (
	SyncPod        PodWorkerState = "SyncPod"
	TerminatingPod PodWorkerState = "TerminatingPod"
	TerminatedPod  PodWorkerState = "TerminatedPod"
)

type UpdatePodOptions struct {
	SyncPodType types.SyncPodType
	Pod         *v1.Pod
}

type PodWorkers interface {
	UpdatePod(pod *v1.Pod, syncPodType types.SyncPodType)
}

type PodSyncer interface {
	SyncPod(pod *v1.Pod, syncPodType types.SyncPodType, podStatus *runtime.PodStatus)
}

type podWorkers struct {
	lock       sync.Mutex
	podSyncer  PodSyncer
	podUpdates map[v1.UID]chan UpdatePodOptions
	cache      runtime.Cache
}

func NewPodWorkers(podSyncer PodSyncer, cache runtime.Cache) PodWorkers {
	return &podWorkers{
		podSyncer:  podSyncer,
		podUpdates: make(map[v1.UID]chan UpdatePodOptions),
		cache:      cache,
	}
}

func (pw *podWorkers) UpdatePod(pod *v1.Pod, syncPodType types.SyncPodType) {
	if updateCh, ok := pw.podUpdates[pod.ObjectMeta.UID]; !ok {
		if syncPodType != types.SyncPodCreate {
			log.Printf("Pod worker goroutine for pod %s does not exist.", pod.ObjectMeta.UID)
			return
		}
		updates := make(chan UpdatePodOptions, 1)
		pw.podUpdates[pod.ObjectMeta.UID] = updates
		go pw.workerLoop(updates)
		updates <- UpdatePodOptions{
			SyncPodType: syncPodType,
			Pod:         pod,
		}
	} else {
		if syncPodType == types.SyncPodCreate {
			log.Printf("Pod worker goroutine for pod %s already exists.", pod.ObjectMeta.UID)
			return
		}
		if syncPodType == types.SyncPodSync {
			updateCh <- UpdatePodOptions{
				SyncPodType: syncPodType,
				Pod:         pod,
			}
		}
	}
}

func (pw *podWorkers) workerLoop(updates <-chan UpdatePodOptions) {
	log.Println("Pod worker started.")
	var lastSyncTime time.Time
	for update := range updates {
		lastSyncTime = time.Now()
		if update.SyncPodType == types.SyncPodCreate {
			pw.podSyncer.SyncPod(update.Pod, update.SyncPodType, nil)
			continue
		}
		if update.SyncPodType == types.SyncPodSync {
			status, err := pw.cache.GetNewerThan(update.Pod.ObjectMeta.UID, lastSyncTime)
			if err != nil {
				log.Printf("Failed to get pod status for pod %s: %v", update.Pod.ObjectMeta.UID, err)
				continue
			}
			pw.podSyncer.SyncPod(update.Pod, update.SyncPodType, status)
			continue
		}
	}
}
