package kubelet

import (
	"context"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	kubepod "minikubernetes/pkg/kubelet/pod"
	"minikubernetes/pkg/kubelet/types"
	"minikubernetes/pkg/kubelet/utils"
	"sync"
)

type Kubelet struct {
	podManger  kubepod.Manager
	podWorkers PodWorkers
}

func NewMainKubelet(apiServerIP string) (*Kubelet, error) {
	kl := &Kubelet{}
	kl.podManger = kubepod.NewPodManager()
	kl.podWorkers = NewPodWorkers(kl)
	log.Println("Kubelet initialized.")
	return kl, nil
}

func (kl *Kubelet) Run(ctx context.Context, wg *sync.WaitGroup, updates <-chan types.PodUpdate) {
	log.Println("Kubelet running...")
	// TODO 启动各种组件
	// kl.pleg.Start()
	// kl.statusManager.Start()
	log.Println("Managers started.")
	kl.syncLoop(ctx, wg, updates)
}

func (kl *Kubelet) syncLoop(ctx context.Context, wg *sync.WaitGroup, updates <-chan types.PodUpdate) {
	defer wg.Done()
	log.Println("Sync loop started.")
	for {
		if !kl.syncLoopIteration(ctx, updates) {
			break
		}
	}
	log.Println("Sync loop ended.")
	kl.DoCleanUp()
}

func (kl *Kubelet) syncLoopIteration(ctx context.Context, configCh <-chan types.PodUpdate) bool {
	// TODO 加入plegCh，syncCh等
	select {
	case update, ok := <-configCh:
		if !ok {
			// 意料之外的通道关闭
			log.Println("Config channel is closed")
			return false
		}
		switch update.Op {
		case types.ADD:
			kl.HandlePodAdditions(update.Pods)
		case types.DELETE:
			kl.HandlePodDeletions(update.Pods)
		default:
			log.Printf("Type %v is not implemented.\n", update.Op)
		}
	case <-ctx.Done():
		// 人为停止
		return false
	}
	return true
}

func (kl *Kubelet) HandlePodAdditions(pods []*v1.Pod) {
	log.Println("Handling pod additions...")
	utils.SortPodsByCreationTime(pods)
	for i, pod := range pods {
		log.Printf("new pod %v: %v.\n", i, pod.Name)
		kl.podManger.UpdatePod(pod)
		// TODO 检查pod是否可以被admit
		// TODO 调用podWorkers.UpdatePod
		kl.podWorkers.UpdatePod(pod, types.SyncPodCreate)
	}
}

func (kl *Kubelet) HandlePodDeletions(pods []*v1.Pod) {
	log.Println("Handling pod deletions...")
	for i, pod := range pods {
		log.Printf("deleted pod %v: %v.\n", i, pod.Name)
	}
}

func (kl *Kubelet) DoCleanUp() {
	log.Println("Kubelet cleanup started.")
	// TODO 停止各种组件
	log.Println("Kubelet cleanup ended.")
}

func (kl *Kubelet) SyncPod(pod *v1.Pod, syncPodType types.SyncPodType) {
	switch syncPodType {
	case types.SyncPodCreate:
		log.Printf("Creating pod %v using container manager.\n", pod.Name)
	default:
		log.Printf("SyncPodType %v is not implemented.\n", syncPodType)
	}
}
