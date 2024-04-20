package pleg

import (
	"context"
	"log"
	"sync"
	"time"
)

type PodLifeCycleEventType string

const (
	// 容器开始运行
	ContainerStarted PodLifeCycleEventType = "ContainerStarted"
	// 容器退出
	ContainerDied PodLifeCycleEventType = "ContainerDied"
)

const (
	RelistPeriod time.Duration = 5 * time.Second
)

type PodLifecycleEvent struct {
	// Pod ID
	PodId string
	// 事件类型
	Type PodLifeCycleEventType
}

type PodLifecycleEventGenerator interface {
	Start()
	Watch() chan *PodLifecycleEvent
	Stop()
}

type pleg struct {
	eventCh chan *PodLifecycleEvent

	// 用于同步停止
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewPLEG() PodLifecycleEventGenerator {
	ctx, cancel := context.WithCancel(context.Background())
	return &pleg{
		eventCh: make(chan *PodLifecycleEvent, 100),
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (p *pleg) Start() {
	log.Printf("Starting PLEG...")
	p.wg.Add(1)
	go func() {
		timer := time.NewTimer(RelistPeriod)
		isStopped := false
		for !isStopped {
			select {
			case <-timer.C:
				p.Relist()
			case <-p.ctx.Done():
				isStopped = true
			}
		}
		log.Printf("PLEG stopped.")
		p.wg.Done()
	}()
}

func (p *pleg) Watch() chan *PodLifecycleEvent {
	return p.eventCh
}

func (p *pleg) Relist() {
	log.Printf("Relisting...")
	// TODO: Relist
	log.Printf("Relisting done.")
}

func (p *pleg) Stop() {
	log.Printf("Stopping PLEG...")
	p.cancel()
	p.wg.Wait()
}
