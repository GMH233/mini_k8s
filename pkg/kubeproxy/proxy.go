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
		log.Printf("Service %s added.", update.Service.Name)
	}
}

func (p *Proxy) HandleServiceDeletions(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service deletions...")
	for _, update := range updates {
		log.Printf("Service %s deleted.", update.Service.Name)
	}
}

func (p *Proxy) HandleServiceEndpointsUpdate(updates []*types.ServiceUpdateSingle) {
	log.Println("Handling service endpoints update...")
	for _, update := range updates {
		log.Printf("Service %s endpoints updated.", update.Service.Name)
	}
}

func (p *Proxy) DoCleanUp() {
	err := p.ipvs.Clear()
	if err != nil {
		log.Printf("Failed to clear ipvs: %v", err)
	}
	log.Printf("Proxy cleanup done.")
}
