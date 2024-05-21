package controller

import (
	"minikubernetes/pkg/controller/podautoscaler"
	"minikubernetes/pkg/controller/replicaset"
)

type ControllerManager interface {
	Run() error
}

type controllerManager struct {
	rsController  replicaset.ReplicaSetController
	hpaController podautoscaler.HorizonalController
}

func NewControllerManager(apiServerIP string) ControllerManager {
	manager := &controllerManager{}
	manager.rsController = replicaset.NewReplicasetManager(apiServerIP)
	manager.hpaController = podautoscaler.NewHorizonalController(apiServerIP)
	return manager
}

func (cm *controllerManager) Run() error {
	// 同时跑起两个controller
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
