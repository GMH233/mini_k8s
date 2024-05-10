package kubeproxy

import (
	"context"
	"log"
	"minikubernetes/pkg/kubeproxy/route"
	"minikubernetes/pkg/kubeproxy/types"
	"sync"
)

type Proxy struct {
	ipvs route.IPVS
}

func NewKubeProxy() (*Proxy, error) {
	p := &Proxy{}
	ipvs, err := route.NewIPVS()
	if err != nil {
		return nil, err
	}
	p.ipvs = ipvs
	return p, nil
}

func (p *Proxy) Run(ctx context.Context, wg *sync.WaitGroup, updates <-chan *types.ServiceUpdate) {
	log.Printf("KubeProxy running...")
	err := p.ipvs.Init()
	if err != nil {
		log.Printf("Failed to init ipvs: %v", err)
		return
	}
	log.Printf("IPVS initialized.")
	p.syncLoop(ctx, wg, updates)
}

func (p *Proxy) syncLoop(ctx context.Context, wg *sync.WaitGroup, updates <-chan *types.ServiceUpdate) {
	defer wg.Done()
	log.Printf("Sync loop started.")
	for {
		if !p.syncLoopIteration(ctx, updates) {
			break
		}
	}
	log.Printf("Sync loop ended.")
	p.DoCleanUp()
}

func (p *Proxy) syncLoopIteration(ctx context.Context, updateCh <-chan *types.ServiceUpdate) bool {
	select {
	case update, ok := <-updateCh:
		if !ok {
			log.Printf("Service update channel closed.")
			return false
		}
		switch update.Op {
		case types.ServiceAdd:
			p.HandleServiceAdditions(update.Updates)
		case types.ServiceDelete:
			p.HandleServiceDeletions(update.Updates)
		case types.ServiceEndpointsUpdate:
			p.HandleServiceEndpointsUpdate(update.Updates)
		}
	case <-ctx.Done():
		return false
	}
	return true
}

func (p *Proxy) HandleServiceAdditions(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service additions...")
	for _, update := range updates {
		log.Printf("Adding service %s.", update.Service.Name)
		svc := update.Service
		vip := svc.Spec.ClusterIP
		reversePortMap := make(map[int32]int32)
		for _, svcPort := range svc.Spec.Ports {
			err := p.ipvs.AddVirtual(vip, uint16(svcPort.Port), svcPort.Protocol)
			if err != nil {
				log.Printf("Failed to add virtual server: %v", err)
				continue
			}
			reversePortMap[svcPort.TargetPort] = svcPort.Port
		}
		for _, endpoint := range update.EndpointAdditions {
			for _, endpointPort := range endpoint.Ports {
				if svcPort, ok := reversePortMap[endpointPort.Port]; ok {
					err := p.ipvs.AddRoute(vip, uint16(svcPort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to add route: %v", err)
					}
				}
			}
		}
		log.Printf("Service %s added.", svc.Name)
	}
}

func (p *Proxy) HandleServiceDeletions(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service deletions...")
	for _, update := range updates {
		log.Printf("Deleting service %s.", update.Service.Name)
		svc := update.Service
		vip := svc.Spec.ClusterIP
		for _, svcPort := range svc.Spec.Ports {
			err := p.ipvs.DeleteVirtual(vip, uint16(svcPort.Port), svcPort.Protocol)
			if err != nil {
				log.Printf("Failed to delete virtual server: %v", err)
			}
		}
		log.Printf("Service %s deleted.", svc.Name)
	}
}

func (p *Proxy) HandleServiceEndpointsUpdate(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service endpoints update...")
	for _, update := range updates {
		log.Printf("Updating service %s.", update.Service.Name)
		svc := update.Service
		vip := svc.Spec.ClusterIP
		reversePortMap := make(map[int32]int32)
		for _, svcPort := range svc.Spec.Ports {
			reversePortMap[svcPort.TargetPort] = svcPort.Port
		}
		for _, endpoint := range update.EndpointDeletions {
			for _, endpointPort := range endpoint.Ports {
				if svcPort, ok := reversePortMap[endpointPort.Port]; ok {
					err := p.ipvs.DeleteRoute(vip, uint16(svcPort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to delete route: %v", err)
					}
				}
			}
		}
		for _, endpoint := range update.EndpointAdditions {
			for _, endpointPort := range endpoint.Ports {
				if svcPort, ok := reversePortMap[endpointPort.Port]; ok {
					err := p.ipvs.AddRoute(vip, uint16(svcPort), endpoint.IP, uint16(endpointPort.Port), endpointPort.Protocol)
					if err != nil {
						log.Printf("Failed to add route: %v", err)
					}
				}
			}
		}
		log.Printf("Service %s updated.", svc.Name)
	}
}

func (p *Proxy) DoCleanUp() {
	err := p.ipvs.Clear()
	if err != nil {
		log.Printf("Failed to clear ipvs: %v", err)
	}
	log.Printf("Proxy cleanup done.")
}
