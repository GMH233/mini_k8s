package controller

import v1 "minikubernetes/pkg/api/v1"

type ReplicaSetControllerInterface interface {
	RunRSC()
}

type ReplicaSetController struct {
	syncHandler func() error
}

func (rc *ReplicaSetController) RunRSC() error {

	err := rc.syncReplicaSet()

	if err != nil {
		return err
	}
	return nil
}

func (rc *ReplicaSetController) addPod(template v1.PodTemplateSpec, meta v1.TypeMeta) {
	pod := &v1.Pod{
		TypeMeta:   meta,
		ObjectMeta: template.ObjectMeta,
		Spec:       template.Spec,
	}
	pod.Spec.RestartPolicy = v1.RestartPolicyAlways
	//TODO：调用API
}

func (rc *ReplicaSetController) deletePod(podId v1.UID) {
	//TODO:调用API
}

func (rc *ReplicaSetController) syncReplicaSet() error {
	for {
		var reps []*v1.ReplicaSet
		var allPods []*v1.Pod
		//TODO:轮询查找pod和rs的数量

		for _, rep := range reps {
			allPodsMatch, err := rc.oneReplicaSetMatch(rep, allPods)
			if err != nil {
				return err
			}
			toStart, err := rc.oneReplicaSetCheck(allPodsMatch, *rep.Spec.Replicas)
			if err != nil {
				return err
			}
			for i := 1; i <= toStart; i++ {
				rc.addPod(rep.Template, rep.TypeMeta)
			}

		}
	}

}

func (rc *ReplicaSetController) oneReplicaSetMatch(rep *v1.ReplicaSet, allPods []*v1.Pod) ([]*v1.Pod, error) {
	resLabelSelector := rep.Spec.Selector
	var podBelonged []*v1.Pod
	for _, pod := range allPods {
		podLabels := pod.Labels
		belonged := true
		for k, v := range resLabelSelector.MatchLabels {
			if podLabels[k] != v {
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

func (rc *ReplicaSetController) oneReplicaSetCheck(allPodsMatch []*v1.Pod, wanted int32) (int, error) {
	wantedNum := int(wanted)
	replicasNum := 0
	stateMark := false
	for _, pod := range allPodsMatch {
		if replicasNum == wantedNum {
			stateMark = true
		}
		if stateMark {
			rc.deletePod(pod.UID)

		} else {
			switch pod.Status.Phase {
			case v1.PodRunning:
			case v1.PodPending:
				replicasNum++
				break
			case v1.PodFailed:
			case v1.PodSucceeded:
				replicasNum++
				break
			}
		}

	}
	if !stateMark {
		neededNum := wantedNum - replicasNum
		return neededNum, nil
	}
	return 0, nil
}
