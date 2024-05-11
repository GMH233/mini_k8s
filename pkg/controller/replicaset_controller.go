package controller

import (
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"time"
)

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

	// pod.Spec.RestartPolicy = v1.RestartPolicyAlways
	//TODO：调用API
	//println("------------------------------")
	//println(pod.Name)
	//for _, v := range pod.Spec.Containers {
	//	println("[]" + v.Image)
	//}
	//for k, v := range pod.ObjectMeta.Labels {
	//	println(k + ":" + v)
	//}

	pod.Name = pod.Name + time.Now().Format("2006-01-02 15:04:05.000000")
	err := rc.client.AddPod(*pod)
	if err != nil {
		return
	}
	//fmt.Println("here add a pod")
	//fmt.Println(pod.ObjectMeta.Name)
	return
}

func (rc *replicaSetController) deletePod(name, namespace string) {
	//TODO:调用API
	err := rc.client.DeletePod(name, namespace)
	if err != nil {
		return
	}
	//fmt.Println("here delete a pod")
}

func (rc *replicaSetController) syncReplicaSet() error {

	//fmt.Println("start to create fake test example")
	//testcase := 3
	//repTest := &v1.ReplicaSet{
	//	Spec: v1.ReplicaSetSpec{
	//		Replicas: 2,
	//		Selector: &v1.LabelSelector{
	//			MatchLabels: map[string]string{
	//				"1": "2",
	//				"4": "5",
	//			},
	//		},
	//	},
	//	Template: v1.PodTemplateSpec{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "test temp",
	//		},
	//	},
	//	TypeMeta: v1.TypeMeta{APIVersion: "test api"},
	//}
	//allPodsTest := make([][]*v1.Pod, 4)
	//for i := range allPodsTest {
	//	// 为每个元素分配内存，使用 make 初始化切片
	//	allPodsTest[i] = make([]*v1.Pod, 4) // 假设每行长度为 4
	//}
	//allPodsTest[0][0] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "55",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[1][0] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[1][1] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "55",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[2][0] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[2][1] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[2][2] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[3][0] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[3][1] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "2",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}
	//allPodsTest[3][2] = &v1.Pod{
	//	ObjectMeta: v1.ObjectMeta{
	//		Labels: map[string]string{
	//			"1": "22",
	//			"4": "5",
	//		},
	//	},
	//	Status: v1.PodStatus{
	//		Phase: v1.PodRunning,
	//	},
	//}

	//_, err := rc.client.GetAllReplicaSets()
	//if err != nil {
	//	return err
	//}
	for {
		var reps []*v1.ReplicaSet
		var allPods []*v1.Pod
		//TODO:轮询查找pod和rs的数量

		//fmt.Println("start to test example ")
		//fmt.Println(testcase)
		//reps = append(reps, repTest)
		//allPods = allPodsTest[testcase]

		reps, err := rc.client.GetAllReplicaSets()
		if err != nil {
			return err
		}

		allPods, err = rc.client.GetAllPods()
		if err != nil {
			return err
		}

		for _, rep := range reps {
			//println(rep.Name)
			//println(rep.Spec.Replicas)
			//for k, v := range rep.Spec.Selector.MatchLabels {
			//	println(k + ":" + v)
			//}
			//println(rep.Spec.Template.Name)
			//for _, ct := range rep.Spec.Template.Spec.Containers {
			//	println(ct.Image)
			//}
			//for k, v := range rep.Spec.Template.Labels {
			//	println(k + ":" + v)
			//}
			//return nil

			allPodsMatch, err := rc.oneReplicaSetMatch(rep, allPods)
			if err != nil {
				return err
			}
			toStart, err := rc.oneReplicaSetCheck(allPodsMatch, rep.Spec.Replicas)
			if err != nil {
				return err
			}
			//fmt.Println(toStart)
			//fmt.Println("start loop")
			for i := 1; i <= toStart; i++ {
				//println(i)
				//println("the " + "to add pod")
				pod := &v1.Pod{
					TypeMeta:   rep.TypeMeta,
					ObjectMeta: rep.Spec.Template.ObjectMeta,
					Spec:       rep.Spec.Template.Spec,
				}
				rc.addPod(pod)
			}

		}

		//testcase--
		//if testcase < 0 {
		//	break
		//}
		time.Sleep(5000 * time.Millisecond)

	}
	//return nil

}

func (rc *replicaSetController) oneReplicaSetMatch(rep *v1.ReplicaSet, allPods []*v1.Pod) ([]*v1.Pod, error) {
	resLabelSelector := rep.Spec.Selector
	var podBelonged []*v1.Pod
	//fmt.Println(len(allPods))
	for _, pod := range allPods {
		//fmt.Println(i)
		if pod == nil || pod.Labels == nil {
			continue
		}
		//fmt.Println(pod.Labels)
		//fmt.Println("************************************")
		//fmt.Println(resLabelSelector.MatchLabels)
		//fmt.Println("-----------------------------------")
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
