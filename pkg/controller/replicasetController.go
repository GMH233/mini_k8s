package controller

import (
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"time"
)

const FormatTime = "2006-01-02 15:04:05.000000"

type ReplicaSetController interface {
	RunRSC() error
}

type replicaSetController struct {
	client      kubeclient.Client
	syncHandler func() error
}

func NewReplicasetManager(apiServerIP string) ReplicaSetController {
	manager := &replicaSetController{}
	manager.client = kubeclient.NewClient(apiServerIP)
	return manager
}

func (rc *replicaSetController) RunRSC() error {

	err := rc.syncReplicaSet()

	if err != nil {
		return err
	}
	return nil
}

func (rc *replicaSetController) addPod(pod *v1.Pod) {

	pod.Name = pod.Name + time.Now().Format(FormatTime)
	pod.TypeMeta.Kind = "Pod"
	err := rc.client.AddPod(*pod)
	if err != nil {
		return
	}
	return
}

func (rc *replicaSetController) deletePod(name, namespace string) {
	err := rc.client.DeletePod(name, namespace)
	if err != nil {
		return
	}

}

func (rc *replicaSetController) syncReplicaSet() error {

	for {
		var reps []*v1.ReplicaSet
		var allPods []*v1.Pod

		reps, err := rc.client.GetAllReplicaSets()
		if err != nil {
			return err
		}

		allPods, err = rc.client.GetAllPods()
		if err != nil {
			return err
		}

		for _, rep := range reps {

			allPodsMatch, err := rc.oneReplicaSetMatch(rep, allPods)
			if err != nil {
				return err
			}
			toStart, err := rc.oneReplicaSetCheck(allPodsMatch, rep.Spec.Replicas)
			if err != nil {
				return err
			}
			for i := 1; i <= toStart; i++ {

				pod := &v1.Pod{
					TypeMeta:   rep.TypeMeta,
					ObjectMeta: rep.Spec.Template.ObjectMeta,
					Spec:       rep.Spec.Template.Spec,
				}
				rc.addPod(pod)
			}

		}

		time.Sleep(5000 * time.Millisecond)

	}

}

func (rc *replicaSetController) oneReplicaSetMatch(rep *v1.ReplicaSet, allPods []*v1.Pod) ([]*v1.Pod, error) {
	resLabelSelector := rep.Spec.Selector
	var podBelonged []*v1.Pod

	for _, pod := range allPods {
		if pod == nil || pod.Labels == nil {
			continue
		}

		belonged := true
		for k, v := range resLabelSelector.MatchLabels {
			if pod.Labels[k] != v {
				belonged = false
				break
			}
		}
		if belonged {
			podBelonged = append(podBelonged, pod)
		}

	}
	return podBelonged, nil

}

func (rc *replicaSetController) oneReplicaSetCheck(allPodsMatch []*v1.Pod, wanted int32) (int, error) {
	wantedNum := int(wanted)
	replicasNum := 0
	stateMark := false
	for _, pod := range allPodsMatch {
		if replicasNum == wantedNum {
			stateMark = true
		}
		if stateMark {
			rc.deletePod(pod.Name, pod.Namespace)

		} else {
			if pod.Status.Phase == v1.PodRunning {
				replicasNum++
			} else if pod.Status.Phase == v1.PodPending {
				replicasNum++
			} else if pod.Status.Phase == v1.PodFailed {

			} else if pod.Status.Phase == v1.PodSucceeded {
				replicasNum++
			}
		}

	}

	if !stateMark {
		neededNum := wantedNum - replicasNum
		return neededNum, nil
	}
	return 0, nil
}
