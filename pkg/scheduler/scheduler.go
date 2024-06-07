package scheduler

import (
	"log"
	"math/rand"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"time"
)

const (
	Round_Policy        = "Round_Policy"
	Random_Policy       = "Random_Policy"
	NodeAffinity_Policy = "NodeAffinity_Policy"
)

type Scheduler interface {
	Run()
}

type scheduler struct {
	client          kubeclient.Client
	roundRobinCount int
	policy          string
}

func NewScheduler(apiServerIP string, policy string) Scheduler {
	manager := &scheduler{}
	manager.client = kubeclient.NewClient(apiServerIP)
	manager.roundRobinCount = 0
	manager.policy = policy
	return manager
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
		policy := sc.policy

		nodes, err := sc.informNodes()
		if err != nil {
			return err
		}
		reqs, limit, err := sc.getResources(pod.Spec.Containers)
		if err != nil {
			return err
		}
		//filteredNodes, err := sc.filterNodes(reqs, limit, nodes)
		//if err != nil {
		//	return err
		//}
		switch policy {
		case Round_Policy:
			SelectedNodes, err := sc.nodesInRoundPolicy(reqs, limit, nodes)

			err = sc.addPodToNode(SelectedNodes, pod)
			if err != nil {
				return err
			}
			break
		case Random_Policy:
			SelectedNodes, err := sc.nodesInRandomPolicy(reqs, limit, nodes)
			if err != nil {
				return err
			}
			err = sc.addPodToNode(SelectedNodes, pod)
			if err != nil {
				return err
			}
			break
		case NodeAffinity_Policy:
			SelectedNodes, err := sc.nodesInNodeAffinityPolicy(reqs, limit, nodes, pod)
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

func (sc *scheduler) nodesInNodeAffinityPolicy(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*v1.Node, pod *v1.Pod) (*v1.Node, error) {
	if pod.Labels == nil {
		return sc.nodesInRandomPolicy(rqs, lim, nodes)
	}
	var selectedNode *v1.Node
	for _, node := range nodes {
		if node.Labels == nil {
			continue
		}
		allLabelsMatch := true
		for key, value := range pod.Labels {
			if v, ok := node.Labels[key]; !ok || v != value {
				allLabelsMatch = false
				break
			}
		}
		if allLabelsMatch {
			selectedNode = node
			break
		}
	}
	if selectedNode == nil {
		return sc.nodesInRandomPolicy(rqs, lim, nodes)
	}
	return selectedNode, nil
}

func (sc *scheduler) informPods() ([]*v1.Pod, error) {
	pods, err := sc.client.GetAllUnscheduledPods()
	if err != nil {
		return nil, err
	}
	//var res []*v1.Pod
	//for _, pod := range pods {
	//	if pod.Status.Phase == v1.PodPending {
	//		res = append(res, pod)
	//	}
	//}
	return pods, nil
}

func (sc *scheduler) informNodes() ([]*v1.Node, error) {

	nodes, err := sc.client.GetAllNodes()
	if err != nil {
		return nil, err
	}
	return nodes, nil
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

func (sc *scheduler) filterNodes(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*v1.Node) ([]*v1.Node, error) {
	var res []*v1.Node
	for _, node := range nodes {
		//TODO：进行过滤
		res = append(res, node)
	}
	return res, nil
}

func (sc *scheduler) nodesInRoundPolicy(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*v1.Node) (*v1.Node, error) {
	if nodes == nil {
		return nil, nil
	}
	lens := len(nodes)
	if lens == 0 {
		return nil, nil
	}
	num := sc.roundRobinCount % lens
	sc.roundRobinCount++
	return nodes[num], nil
}

func (sc *scheduler) nodesInRandomPolicy(rqs []v1.ResourceList, lim []v1.ResourceList, nodes []*v1.Node) (*v1.Node, error) {
	if nodes == nil {
		return nil, nil
	}
	lens := len(nodes)
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(lens)
	return nodes[num], nil
}

func (sc *scheduler) addPodToNode(node *v1.Node, pod *v1.Pod) error {
	if node == nil {
		return nil
	}
	err := sc.client.AddPodToNode(*pod, *node)
	if err != nil {
		return err
	}
	return nil
}
