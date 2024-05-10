package app

import (
	"context"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"minikubernetes/pkg/kubeproxy"
	"minikubernetes/pkg/kubeproxy/types"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"
)

type ProxyServer struct {
	client  kubeclient.Client
	updates chan *types.ServiceUpdate
	latest  map[v1.UID]*types.ServiceAndEndpoints
}

func NewProxyServer(apiServerIP string) (*ProxyServer, error) {
	ps := &ProxyServer{}
	ps.client = kubeclient.NewClient(apiServerIP)
	ps.updates = make(chan *types.ServiceUpdate, 1)
	ps.latest = make(map[v1.UID]*types.ServiceAndEndpoints)
	return ps, nil
}

func (ps *ProxyServer) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)

	signalCh := make(chan os.Signal, 10)
	signal.Notify(signalCh, os.Interrupt)

	ps.RunProxy(ctx, &wg)
	go ps.watchApiServer(ctx, &wg)

	<-signalCh
	log.Println("Received interrupt signal, shutting down...")
	cancel()
	wg.Wait()
}

func (ps *ProxyServer) RunProxy(ctx context.Context, wg *sync.WaitGroup) {
	proxy, err := kubeproxy.NewKubeProxy()
	if err != nil {
		log.Printf("Failed to create proxy: %v", err)
		return
	}
	go proxy.Run(ctx, wg, ps.updates)
}

func (ps *ProxyServer) watchApiServer(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	SyncPeriod := 4 * time.Second
	timer := time.NewTimer(SyncPeriod)
	for {
		select {
		case <-timer.C:
			ps.updateService()
			timer.Reset(SyncPeriod)
		case <-ctx.Done():
			log.Println("Shutting down api server watcher")
			return
		}
	}
}

func (ps *ProxyServer) updateService() {
	pods, err := ps.client.GetAllPods()
	if err != nil {
		log.Printf("Failed to get pods: %v", err)
		return
	}
	services, err := ps.client.GetAllServices()
	if err != nil {
		log.Printf("Failed to get services: %v", err)
		return
	}
	//pods := getMockPods()
	//services := getMockServices()

	newMap := make(map[v1.UID]*types.ServiceAndEndpoints)
	for _, service := range services {
		var endpoints []*types.Endpoint
		for _, pod := range pods {
			if !isSelectorMatched(pod, service) {
				continue
			}
			if pod.Status.Phase != v1.PodRunning {
				continue
			}
			if pod.Status.PodIP == "" {
				continue
			}

			var ports []types.EndpointPort
			for _, port := range service.Spec.Ports {
			SearchLoop:
				for _, container := range pod.Spec.Containers {
					for _, containerPort := range container.Ports {
						if containerPort.ContainerPort == port.TargetPort && containerPort.Protocol == port.Protocol {
							ports = append(ports, types.EndpointPort{
								Port:     port.TargetPort,
								Protocol: port.Protocol,
							})
							break SearchLoop
						}
					}
				}
			}

			endpoints = append(endpoints, &types.Endpoint{
				IP:    pod.Status.PodIP,
				Ports: ports,
			})
		}
		newMap[service.ObjectMeta.UID] = &types.ServiceAndEndpoints{
			Service:   service,
			Endpoints: endpoints,
		}
	}

	defer func() {
		ps.latest = newMap
	}()

	var additions []*types.ServiceUpdateSingle
	var deletions []*types.ServiceUpdateSingle
	var endpointUpdates []*types.ServiceUpdateSingle

	for uid, oldItem := range ps.latest {
		if _, ok := newMap[uid]; !ok {
			deletions = append(deletions, &types.ServiceUpdateSingle{
				Service: oldItem.Service,
			})
		}
	}
	for uid, newItem := range newMap {
		oldItem := ps.latest[uid]
		if diff, update, op := calculateDiff(oldItem, newItem); diff {
			if op == types.ServiceAdd {
				additions = append(additions, update)
			} else {
				endpointUpdates = append(endpointUpdates, update)
			}
		}
	}

	if len(additions) > 0 {
		ps.updates <- &types.ServiceUpdate{
			Updates: additions,
			Op:      types.ServiceAdd,
		}
	}
	if len(deletions) > 0 {
		ps.updates <- &types.ServiceUpdate{
			Updates: deletions,
			Op:      types.ServiceDelete,
		}
	}
	if len(endpointUpdates) > 0 {
		ps.updates <- &types.ServiceUpdate{
			Updates: endpointUpdates,
			Op:      types.ServiceEndpointsUpdate,
		}
	}
}

func isSelectorMatched(pod *v1.Pod, svc *v1.Service) bool {
	for key, value := range svc.Spec.Selector {
		if label, ok := pod.Labels[key]; !ok || label != value {
			return false
		}
	}
	return true
}

func calculateDiff(oldItem, newItem *types.ServiceAndEndpoints) (bool, *types.ServiceUpdateSingle, types.ServiceOperation) {
	var op types.ServiceOperation
	var diff bool
	if oldItem == nil {
		op = types.ServiceAdd
		oldItem = &types.ServiceAndEndpoints{}
		diff = true
	} else {
		op = types.ServiceEndpointsUpdate
		diff = false
	}
	update := &types.ServiceUpdateSingle{
		Service: newItem.Service,
	}
	oldIPMap := make(map[string][]types.EndpointPort)
	newIPMap := make(map[string][]types.EndpointPort)
	for _, ep := range oldItem.Endpoints {
		oldIPMap[ep.IP] = ep.Ports
	}
	for _, ep := range newItem.Endpoints {
		newIPMap[ep.IP] = ep.Ports
	}
	for ip, oldPorts := range oldIPMap {
		if newPorts, ok := newIPMap[ip]; !ok || !isEndpointPortsEqual(oldPorts, newPorts) {
			diff = true
			update.EndpointDeletions = append(update.EndpointDeletions, &types.Endpoint{
				IP:    ip,
				Ports: oldPorts,
			})
			if ok {
				update.EndpointAdditions = append(update.EndpointAdditions, &types.Endpoint{
					IP:    ip,
					Ports: newPorts,
				})
			}
		}
	}
	for ip, newPorts := range newIPMap {
		if _, ok := oldIPMap[ip]; !ok {
			diff = true
			update.EndpointAdditions = append(update.EndpointAdditions, &types.Endpoint{
				IP:    ip,
				Ports: newPorts,
			})
		}
	}
	return diff, update, op
}

// 以下为fake数据
var cnt int = 0

func getMockPods() []*v1.Pod {
	cnt++
	if cnt == 1 {
		return []*v1.Pod{}
	}
	if cnt == 2 {
		return []*v1.Pod{
			{
				ObjectMeta: v1.ObjectMeta{
					UID:    "abcde",
					Labels: map[string]string{"app": "nginx"},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Ports: []v1.ContainerPort{
								{
									ContainerPort: 8080,
									Protocol:      v1.ProtocolTCP,
								},
								{
									ContainerPort: 9090,
									Protocol:      v1.ProtocolTCP,
								},
							},
						},
					},
				},
				Status: v1.PodStatus{
					Phase: v1.PodRunning,
					PodIP: "10.32.0.1",
				},
			},
		}
	}
	return []*v1.Pod{
		{
			ObjectMeta: v1.ObjectMeta{
				UID:    "abcde",
				Labels: map[string]string{"app": "nginx"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Ports: []v1.ContainerPort{
							{
								ContainerPort: 8080,
								Protocol:      v1.ProtocolTCP,
							},
							{
								ContainerPort: 9091,
								Protocol:      v1.ProtocolTCP,
							},
						},
					},
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
				PodIP: "10.32.0.1",
			},
		},
	}
}

func getMockServices() []*v1.Service {
	//if cnt > 2 {
	//	return []*v1.Service{}
	//}
	return []*v1.Service{
		{
			ObjectMeta: v1.ObjectMeta{
				Name: "nginx",
				UID:  "bdafsdfjlasdfas",
			},
			Spec: v1.ServiceSpec{
				Selector:  map[string]string{"app": "nginx"},
				ClusterIP: "100.1.1.0",
				Ports: []v1.ServicePort{
					{
						Port:       80,
						TargetPort: 8080,
						Protocol:   v1.ProtocolTCP,
					},
					{
						Port:       90,
						TargetPort: 9090,
						Protocol:   v1.ProtocolTCP,
					},
				},
			},
		},
	}
}

func isEndpointPortsEqual(a, b []types.EndpointPort) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Slice(a, func(i, j int) bool {
		return a[i].Port < a[j].Port
	})
	sort.Slice(b, func(i, j int) bool {
		return b[i].Port < b[j].Port
	})
	for i := range a {
		if a[i].Port != b[i].Port || a[i].Protocol != b[i].Protocol {
			return false
		}
	}
	return true
}
