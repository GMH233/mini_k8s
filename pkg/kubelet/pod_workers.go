package kubelet

import (
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/types"
	"sync"
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
	SyncPod(pod *v1.Pod, syncPodType types.SyncPodType)
}

type podWorkers struct {
	lock       sync.Mutex
	podSyncer  PodSyncer
	podUpdates map[v1.UID]chan UpdatePodOptions
}

func NewPodWorkers(podSyncer PodSyncer) PodWorkers {
	return &podWorkers{
		podSyncer:  podSyncer,
		podUpdates: make(map[v1.UID]chan UpdatePodOptions),
	}
}

func (pw *podWorkers) UpdatePod(pod *v1.Pod, syncPodType types.SyncPodType) {
	if _, ok := pw.podUpdates[pod.ObjectMeta.UID]; !ok {
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
	}
}

func (pw *podWorkers) workerLoop(updates <-chan UpdatePodOptions) {
	log.Println("Pod worker started.")
	for update := range updates {
		pw.podSyncer.SyncPod(update.Pod, update.SyncPodType)
	}
}
