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
		kl.HandlePodLifecycleEvent(pod, e)
	case <-ctx.Done():
		// 人为停止
		return false
	}
	return true
}

func (kl *Kubelet) HandlePodAdditions(pods []*v1.Pod) {
	log.Println("Handling pod additions...")
	utils.SortPodsByCreationTime(pods)
	for _, pod := range pods {
		// log.Printf("new pod %v: %v.\n", i, pod.Name)
		kl.podManger.UpdatePod(pod)
		// TODO 检查pod是否可以被admit
		kl.podWorkers.UpdatePod(pod, types.SyncPodCreate)
	}
}

func (kl *Kubelet) HandlePodDeletions(pods []*v1.Pod) {
	log.Println("Handling pod deletions...")
	for i, pod := range pods {
		log.Printf("deleted pod %v: %v.\n", i, pod.Name)
		kl.podManger.DeletePod(pod)
		kl.podWorkers.UpdatePod(pod, types.SyncPodKill)
	}
}

func (kl *Kubelet) HandlePodLifecycleEvent(pod *v1.Pod, event *pleg.PodLifecycleEvent) {
	log.Println("Handling pod lifecycle events...")
	//if event.Type == pleg.ContainerRemoved {
	//	if pod.Spec.RestartPolicy == v1.RestartPolicyAlways || pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure {
	//		kl.podWorkers.UpdatePod(pod, types.SyncPodRecreate)
	//		return
	//	}
	//}
	if event.Type == pleg.ContainerDied && pod.Spec.RestartPolicy == v1.RestartPolicyAlways {
		kl.podWorkers.UpdatePod(pod, types.SyncPodRecreate)
		return
	}
	// TODO 实现OnFailure策略
	kl.podWorkers.UpdatePod(pod, types.SyncPodSync)
}

func (kl *Kubelet) DoCleanUp() {
	log.Println("Kubelet cleanup started.")
	// TODO 停止各种组件
	kl.pleg.Stop()
	// unregister node
	err := kl.kubeClient.UnregisterNode(kl.nodeName)
	if err != nil {
		log.Printf("Failed to unregister node %v: %v\n", kl.nodeName, err)
	}
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
		apiStatus := kl.computeApiStatus(pod, podStatus)
		if apiStatus == nil {
			// log.Printf("Pod %v status is unknown.\n", pod.Name)
			return
		}
		err := kl.kubeClient.UpdatePodStatus(pod, apiStatus)
		log.Printf("Pod %v syncing to apiserver. Phase: %v\n", pod.Name, apiStatus.Phase)
		if err != nil {
			log.Printf("Failed to update pod %v status to api server: %v\n", pod.Name, err)
			return
		}
		log.Printf("Pod %v synced\n", pod.Name)
	case types.SyncPodKill:
		log.Printf("Killing pod %v\n", pod.Name)
		err := kl.runtimeManager.DeletePod(pod.UID)
		if err != nil {
			log.Printf("Failed to kill pod %v: %v\n", pod.Name, err)
			return
		}
		log.Printf("Pod %v killed.\n", pod.Name)
	case types.SyncPodRecreate:
		log.Printf("Recreating pod %v\n", pod.Name)
		err := kl.runtimeManager.RestartPod(pod)
		if err != nil {
			log.Printf("Failed to recreate pod %v: %v\n", pod.Name, err)
			return
		}
		log.Printf("Pod %v recreated.\n", pod.Name)
	default:
		log.Printf("SyncPodType %v is not implemented.\n", syncPodType)
	}
}

func (kl *Kubelet) computeApiStatus(pod *v1.Pod, podStatus *runtime.PodStatus) *v1.PodStatus {
	running := 0
	exited := 0
	succeeded := 0
	failed := 0

	if len(podStatus.ContainerStatuses) != len(pod.Spec.Containers) {
		log.Printf("Pod %v has untracked container!\n", podStatus.Name)
		return nil
	}

	podIP := ""
	if podStatus.IPs != nil && len(podStatus.IPs) > 0 {
		podIP = podStatus.IPs[0]
	}

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
			PodIP: podIP,
		}
	} else if exited == len(podStatus.ContainerStatuses) {
		if failed > 0 {
			return &v1.PodStatus{
				Phase: v1.PodFailed,
				PodIP: podIP,
			}
		}
		return &v1.PodStatus{
			Phase: v1.PodSucceeded,
			PodIP: podIP,
		}
	} else {
		log.Printf("Pod %v status unknown!\n", podStatus.Name)
		return nil
	}
}
