package app

import (
	"context"
	"github.com/google/uuid"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet"
	"minikubernetes/pkg/kubelet/client"
	"minikubernetes/pkg/kubelet/types"
	"os"
	"os/signal"
	"sync"
	"time"
)

type KubeletServer struct {
	kubeClient      client.KubeletClient
	nodeName        string
	latestLocalPods []*v1.Pod
	updates         chan types.PodUpdate
}

func NewKubeletServer(apiServerIP string) (*KubeletServer, error) {
	ks := &KubeletServer{}
	ks.kubeClient = client.NewKubeletClient(apiServerIP)
	ks.latestLocalPods = make([]*v1.Pod, 0)
	ks.updates = make(chan types.PodUpdate)
	return ks, nil
}

func (kls *KubeletServer) Run() {
	// TODO 获取当前节点的nodeId
	kls.nodeName = "node-0"

	// context+wait group实现notify和join
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)

	// 捕获SIGINT
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	kls.RunKubelet(ctx, &wg)

	go kls.watchApiServer(ctx, &wg)

	// SIGINT到来，调用cancel()，并等待所有goroutine结束
	<-signalCh
	log.Println("Received interrupt signal, shutting down...")
	cancel()
	wg.Wait()
}

func (kls *KubeletServer) RunKubelet(ctx context.Context, wg *sync.WaitGroup) {
	kl, err := kls.createAndInitKubelet()
	if err != nil {
		return
	}
	kls.startKubelet(ctx, wg, kl)
}

func (kls *KubeletServer) createAndInitKubelet() (*kubelet.Kubelet, error) {
	kl, err := kubelet.NewMainKubelet(kls.nodeName, kls.kubeClient)
	if err != nil {
		return nil, err
	}
	return kl, nil
}

func (kls *KubeletServer) startKubelet(ctx context.Context, wg *sync.WaitGroup, kl *kubelet.Kubelet) {
	go kl.Run(ctx, wg, kls.updates)
}

func (kls *KubeletServer) watchApiServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	SyncPeriod := 5 * time.Second
	timer := time.NewTimer(SyncPeriod)
	for {
		select {
		case <-timer.C:
			kls.updateLocalPods()
			timer.Reset(SyncPeriod)
		case <-ctx.Done():
			log.Println("Shutting down api server watcher")
			return
		}
	}
}

func (kls *KubeletServer) updateLocalPods() {
	// TODO: Get pods for current node from api server
	// Mock
	// newLocalPods := getMockPods(kls.latestLocalPods)
	newLocalPods, err := kls.kubeClient.GetPodsByNodeName(kls.nodeName)
	if err != nil {
		log.Printf("Failed to get pods for node %s: %v", kls.nodeName, err)
		return
	}

	// For now, we only allow additions and deletions
	oldTable := make(map[v1.UID]*v1.Pod)
	newTable := make(map[v1.UID]*v1.Pod)
	for _, pod := range kls.latestLocalPods {
		oldTable[pod.ObjectMeta.UID] = pod
	}
	for _, pod := range newLocalPods {
		newTable[pod.ObjectMeta.UID] = pod
	}
	additions := make([]*v1.Pod, 0)
	deletions := make([]*v1.Pod, 0)
	for _, pod := range kls.latestLocalPods {
		if _, ok := newTable[pod.ObjectMeta.UID]; !ok {
			deletions = append(deletions, pod)
		}
	}
	for _, pod := range newLocalPods {
		if _, ok := oldTable[pod.ObjectMeta.UID]; !ok {
			additions = append(additions, pod)
		}
	}
	if len(additions) != 0 {
		kls.updates <- types.PodUpdate{
			Pods: additions,
			Op:   types.ADD,
		}
	}
	if len(deletions) != 0 {
		kls.updates <- types.PodUpdate{
			Pods: deletions,
			Op:   types.DELETE,
		}
	}
	kls.latestLocalPods = newLocalPods
}

// 以下为fake数据
var firstTime bool = true

func getMockPods(oldPods []*v1.Pod) []*v1.Pod {
	if !firstTime {
		return oldPods
	}
	firstTime = false
	conport := v1.ContainerPort{ContainerPort: 8080}
	conport2 := v1.ContainerPort{ContainerPort: 8090}
	contain := v1.Container{
		Image:   "alpine:latest",
		Command: []string{},
		Ports:   []v1.ContainerPort{conport},
	}
	contain2 := v1.Container{
		Image:   "alpine:latest",
		Command: []string{},
		Ports:   []v1.ContainerPort{conport2},
	}
	podSpec := v1.PodSpec{
		Containers: []v1.Container{contain, contain2},
	}
	pod := &v1.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		}, ObjectMeta: v1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "default",
			CreationTimestamp: time.Now(),
			UID:               v1.UID(uuid.New().String()),
		},
		Spec: podSpec,
	}
	return []*v1.Pod{pod}
}
