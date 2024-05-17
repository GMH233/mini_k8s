package scheduler

import (
	"log"
	"math/rand"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"time"
)

const (
	Round_Policy  = "Round_Policy"
	Random_Policy = "Random_Policy"
)

type Scheduler interface {
}

type Node struct {
}

type scheduler struct {
	client kubeclient.Client
}

func (sc *scheduler) Run() {
	for {
		err := sc.syncLoop()
		if err != nil {
			log.Println("sync loop err:", err)
			break
		}
		time.Sleep(1000 * time.Millisecond)
	}

}

func (sc *scheduler) syncLoop() error {
	pods, err := sc.informPods()
	if err != nil {
		return err
	}
	for _, pod := range pods {
		policy := "Round_Policy"

		nodes, err := sc.informNodes()
		if err != nil {
			return err
		}
		reqs, limit, err := sc.getResources(pod.Spec.Containers)
		if err != nil {
			return err
		}
		filteredNodes, err := sc.filterNodes(reqs, limit, nodes)
		if err != nil {
			return err
		}
		switch policy {
		case Round_Policy:
			SelectedNodes, err := sc.nodesInRoundPolicy(reqs, limit, filteredNodes)
			if err != nil {
				return err
			}
			err = sc.addPodToNode(SelectedNodes, pod)
			if err != nil {
				return err
			}
			break
		case Random_Policy:
			SelectedNodes, err := sc.nodesInRandomPolicy(reqs, limit, filteredNodes)
			if err != nil {
				return err
			}
			err = sc.addPodToNode(SelectedNodes, pod)
			if err != nil {
				return err
			}
			break
		}

	}
	return nil
}

func (sc *scheduler) informPods() ([]*v1.Pod, error) {
	pods, err := sc.client.GetAllPods()
	if err != nil {
		return nil, err
	}
	var res []*v1.Pod
	for _, pod := range pods {
		if pod.Status.Phase == v1.PodPending {
			res = append(res, pod)
		}
	}
	return res, nil
}

func (sc *scheduler) informNodes() ([]*Node, error) {
	return nil, nil
}

func (sc *scheduler) getResources(cts []v1.Container) ([]v1.ResourceList, []v1.ResourceList, error) {
	var reqs []v1.ResourceList
	var limit []v1.ResourceList
	for _, ct := range cts {
		reqs = append(reqs, ct.Resources.Requests)
		limit = append(limit, ct.Resources.Limits)
	}
	return reqs, limit, nil
}

func (sc *scheduler) filterNodes(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*Node) ([]*Node, error) {
	var res []*Node
	for _, node := range nodes {
		//TODO：进行过滤
		res = append(res, node)
	}
	return res, nil
}

func (sc *scheduler) nodesInRoundPolicy(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*Node) (*Node, error) {
	if nodes == nil {
		return nil, nil
	}

	return nodes[0], nil
}

func (sc *scheduler) nodesInRandomPolicy(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*Node) (*Node, error) {
	if nodes == nil {
		return nil, nil
	}
	lens := len(nodes)
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(lens)
	return nodes[num], nil
}

func (sc *scheduler) addPodToNode(node *Node, pod *v1.Pod) error {
	//TODO:调用API
	return nil
}
