package controller

import (
	"log"
	"minikubernetes/pkg/controller/podautoscaler"
	"minikubernetes/pkg/controller/pv"
	"minikubernetes/pkg/controller/replicaset"
)

type ControllerManager interface {
	Run() error
}

type controllerManager struct {
	rsController  replicaset.ReplicaSetController
	hpaController podautoscaler.HorizonalController
	pvController  pv.Controller
}

func NewControllerManager(apiServerIP string) ControllerManager {
	manager := &controllerManager{}
	manager.rsController = replicaset.NewReplicasetManager(apiServerIP)
	manager.hpaController = podautoscaler.NewHorizonalController(apiServerIP)
	pvController, err := pv.NewPVController(apiServerIP)
	if err != nil {
		log.Printf("create pv controller failed, err: %v", err)
	}
	manager.pvController = pvController
	return manager
}

func (cm *controllerManager) Run() error {
	// 同时跑起两个controller
	if cm.pvController != nil {
		go cm.pvController.Run()
	}
	err := cm.rsController.RunRSC()
	if err != nil {
		return err
	}
	err = cm.hpaController.Run()
	if err != nil {
		return err
	}
	return nil
}
