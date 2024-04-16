package kubelet

import (
	"context"
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/types"
	"sync"
)

type Kubelet struct {
}

func NewMainKubelet() (*Kubelet, error) {
	fmt.Println("Kubelet initialized.")
	return &Kubelet{}, nil
}

func (kl *Kubelet) Run(ctx context.Context, wg *sync.WaitGroup, updates <-chan types.PodUpdate) {
	fmt.Println("Kubelet running...")
	// TODO 启动各种组件
	// kl.pleg.Start()
	// kl.statusManager.Start()
	fmt.Println("Managers started.")
	kl.syncLoop(ctx, wg, updates)
}

func (kl *Kubelet) syncLoop(ctx context.Context, wg *sync.WaitGroup, updates <-chan types.PodUpdate) {
	defer wg.Done()
	fmt.Println("Sync loop started.")
	for {
		if !kl.syncLoopIteration(ctx, wg, updates) {
			break
		}
	}
	fmt.Println("Sync loop ended.")
	kl.DoCleanUp()
}

func (kl *Kubelet) syncLoopIteration(ctx context.Context, wg *sync.WaitGroup, configCh <-chan types.PodUpdate) bool {
	// TODO 加入plegCh，syncCh等
	select {
	case update, ok := <-configCh:
		if !ok {
			// 意料之外的通道关闭
			fmt.Println("Config channel is closed")
			return false
		}
		switch update.Op {
		case types.ADD:
			kl.HandlePodAdditions(update.Pods)
		case types.DELETE:
			kl.HandlePodDeletions(update.Pods)
		default:
			fmt.Printf("Type %v is not implemented.\n", update.Op)
		}
	case <-ctx.Done():
		// 人为停止
		return false
	}
	return true
}

func (kl *Kubelet) HandlePodAdditions(pods []*v1.Pod) {
	fmt.Println("Handling pod additions...")
	for i, pod := range pods {
		fmt.Printf("new pod %v: %v.\n", i, pod.Name)
	}
}

func (kl *Kubelet) HandlePodDeletions(pods []*v1.Pod) {
	fmt.Println("Handling pod deletions...")
	for i, pod := range pods {
		fmt.Printf("deleted pod %v: %v.\n", i, pod.Name)
	}
}

func (kl *Kubelet) DoCleanUp() {
	fmt.Println("Kubelet cleanup started.")
	// TODO 停止各种组件
	fmt.Println("Kubelet cleanup ended.")
}
