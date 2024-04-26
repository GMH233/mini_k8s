package kubelet

import (
	"context"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/client"
	"minikubernetes/pkg/kubelet/pleg"
	kubepod "minikubernetes/pkg/kubelet/pod"
	"minikubernetes/pkg/kubelet/runtime"
	"minikubernetes/pkg/kubelet/types"
	"minikubernetes/pkg/kubelet/utils"
	"sync"
)

type Kubelet struct {
	nodeName       string
	podManger      kubepod.Manager
	podWorkers     PodWorkers
	pleg           pleg.PodLifecycleEventGenerator
	kubeClient     client.KubeletClient
	runtimeManager runtime.RuntimeManager
	cache          runtime.Cache
}

func NewMainKubelet(nodeName string, kubeClient client.KubeletClient) (*Kubelet, error) {
	kl := &Kubelet{}

	kl.nodeName = nodeName
	kl.podManger = kubepod.NewPodManager()
	kl.kubeClient = kubeClient
	kl.runtimeManager = runtime.NewRuntimeManager()
	kl.cache = runtime.NewCache()
	kl.pleg = pleg.NewPLEG(kl.runtimeManager, kl.cache)
	kl.podWorkers = NewPodWorkers(kl, kl.cache)

	log.Println("Kubelet initialized.")
	return kl, nil
}

func (kl *Kubelet) Run(ctx context.Context, wg *sync.WaitGroup, updates <-chan types.PodUpdate) {
	log.Println("Kubelet running...")
	// TODO 启动各种组件
	kl.pleg.Start()
	// kl.statusManager.Start()
	log.Println("Managers started.")
	kl.syncLoop(ctx, wg, updates)
}

func (kl *Kubelet) syncLoop(ctx context.Context, wg *sync.WaitGroup, updates <-chan types.PodUpdate) {
	defer wg.Done()
	log.Println("Sync loop started.")
	plegCh := kl.pleg.Watch()
	for {
		if !kl.syncLoopIteration(ctx, updates, plegCh) {
			break
		}
	}
	log.Println("Sync loop ended.")
	kl.DoCleanUp()
}

func (kl *Kubelet) syncLoopIteration(ctx context.Context, configCh <-chan types.PodUpdate, plegCh <-chan *pleg.PodLifecycleEvent) bool {
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
	case e := <-plegCh:
		log.Printf("PLEG event: pod %v --- %v.\n", e.PodId, e.Type)
		pod, ok := kl.podManger.GetPodByUid(e.PodId)
		if !ok {
			log.Printf("Pod %v not found.\n", e.PodId)
			return true
		}
		kl.HandlePodLifecycleEvent([]*v1.Pod{pod})
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
		kl.podWorkers.UpdatePod(pod, types.SyncPodCreate)
	}
}

func (kl *Kubelet) HandlePodDeletions(pods []*v1.Pod) {
	log.Println("Handling pod deletions...")
	for i, pod := range pods {
		log.Printf("deleted pod %v: %v.\n", i, pod.Name)
	}
}

func (kl *Kubelet) HandlePodLifecycleEvent(pods []*v1.Pod) {
	log.Println("Handling pod lifecycle events...")
	for i, pod := range pods {
		log.Printf("pod %v: %v.\n", i, pod.Name)
		kl.podWorkers.UpdatePod(pod, types.SyncPodSync)
	}
}

func (kl *Kubelet) DoCleanUp() {
	log.Println("Kubelet cleanup started.")
	// TODO 停止各种组件
	kl.pleg.Stop()
	log.Println("Kubelet cleanup ended.")
}

func (kl *Kubelet) SyncPod(pod *v1.Pod, syncPodType types.SyncPodType, podStatus *runtime.PodStatus) {
	switch syncPodType {
	case types.SyncPodCreate:
		log.Printf("Creating pod %v using container manager.\n", pod.Name)
		err := kl.runtimeManager.AddPod(pod)
		if err != nil {
			return
		}
		log.Printf("Pod %v created.\n", pod.Name)
	case types.SyncPodSync:
		log.Printf("Syncing pod %v\n", pod.Name)
		if podStatus == nil {
			log.Printf("Pod %v status is nil.\n", pod.Name)
			return
		}
		apiStatus := kl.computeApiStatus(podStatus)
		if apiStatus == nil {
			log.Printf("Pod %v status is unknown.\n", pod.Name)
			return
		}
		// kl.kubeClient.UpdatePodStatus(pod, apiStatus)
		log.Printf("Pod %v synced. Phase: %v\n", pod.Name, apiStatus.Phase)
	default:
		log.Printf("SyncPodType %v is not implemented.\n", syncPodType)
	}
}

func (kl *Kubelet) computeApiStatus(podStatus *runtime.PodStatus) *v1.PodStatus {
	running := 0
	exited := 0
	succeeded := 0
	failed := 0

	for _, containerStatus := range podStatus.ContainerStatuses {
		if containerStatus.State == runtime.ContainerStateRunning {
			running++
		} else if containerStatus.State == runtime.ContainerStateExited {
			exited++
			if containerStatus.ExitCode == 0 {
				succeeded++
			} else {
				failed++
			}
		}
	}
	if running != 0 {
		return &v1.PodStatus{
			Phase: v1.PodRunning,
		}
	} else if exited == len(podStatus.ContainerStatuses) {
		if failed > 0 {
			return &v1.PodStatus{
				Phase: v1.PodFailed,
			}
		}
		return &v1.PodStatus{
			Phase: v1.PodSucceeded,
		}
	} else {
		log.Printf("Pod %v status unknown!\n", podStatus.Name)
		return nil
	}
}
