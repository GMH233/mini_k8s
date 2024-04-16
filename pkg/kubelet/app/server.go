package app

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet"
	"minikubernetes/pkg/kubelet/types"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"
)

type KubeletServer struct {
	apiServerIP     string
	nodeId          string
	latestLocalPods []*v1.Pod
	updates         chan types.PodUpdate
}

func NewKubeletServer(apiServerIP string) (*KubeletServer, error) {
	return &KubeletServer{
		apiServerIP:     apiServerIP,
		latestLocalPods: make([]*v1.Pod, 0),
		updates:         make(chan types.PodUpdate),
	}, nil
}

func (kls *KubeletServer) Run() {
	// TODO 获取当前节点的nodeId
	kls.nodeId = "node1"
	kls.RunKubelet()
}

func (kls *KubeletServer) RunKubelet() {
	// context+waitgroup实现notify和join
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(2)

	// 捕获SIGINT
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	kl, err := kls.createAndInitKubelet()
	if err != nil {
		return
	}
	kls.startKubelet(ctx, &wg, kl)
	go kls.watchApiServer(ctx, &wg)

	// SIGINT到来，调用cancel()，并等待所有goroutine结束
	<-signalCh
	fmt.Println("Received interrupt signal, shutting down...")
	cancel()
	wg.Wait()
}

func (kls *KubeletServer) createAndInitKubelet() (*kubelet.Kubelet, error) {
	kl, err := kubelet.NewMainKubelet()
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
			fmt.Println("Shutting down api server watcher")
			return
		}
	}
}

func (kls *KubeletServer) updateLocalPods() {
	// TODO: Get pods for current node from api server
	// Mock
	newLocalPods := getMockPods(kls.latestLocalPods)

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

	// sort the pods by creation time
	sort.Slice(kls.latestLocalPods, func(i, j int) bool {
		return kls.latestLocalPods[i].CreationTimestamp.Before(kls.latestLocalPods[j].CreationTimestamp)
	})
}

func getMockPods(oldPods []*v1.Pod) []*v1.Pod {
	newPods := append(oldPods, &v1.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "nginx",
			Namespace:         "default",
			CreationTimestamp: time.Now(),
			UID:               v1.UID(uuid.New().String()),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
	})
	newPods = append(newPods, &v1.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              "busybox",
			Namespace:         "default",
			CreationTimestamp: time.Now(),
			UID:               v1.UID(uuid.New().String()),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "busybox",
					Image: "busybox:latest",
				},
			},
		},
	})
	newPods = newPods[1:]
	return newPods
}
