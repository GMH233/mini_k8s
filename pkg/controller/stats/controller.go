package stats

import (
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"os"
	"os/signal"
	"strconv"
	"time"
)

var magicCAdvisorPort = 8090

var promeLabels = map[string]string{
	"monitor": "prometheus",
}

type StatsController interface {
	RunStatsC() error
}

type statsController struct {
	client kubeclient.Client
}

func NewStatsManager(apiServerIP string) StatsController {
	manager := &statsController{}
	manager.client = kubeclient.NewClient(apiServerIP)
	return manager
}

func (sc *statsController) RunStatsC() error {

	err := sc.syncStats()

	if err != nil {
		return err
	}
	return nil
}

func (sc *statsController) syncStats() error {
	syncInterval := 5

	log.Printf("[Stats] start sync stats")
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		syncTicker := time.NewTicker(time.Duration(syncInterval) * time.Second)
		defer syncTicker.Stop()
		for {
			select {
			case <-syncTicker.C:
				err := sc.syncHandler()
				if err != nil {
					log.Printf("[Stats] sync stats failed: %v", err)
					continue
				}
			}
		}
	}()
	<-signalCh
	log.Printf("[Stats] stop sync stats")
	return nil
}

func (sc *statsController) syncHandler() error {
	log.Printf("[Stats] sync stats")
	nodes, err := sc.client.GetAllNodes()
	if err != nil {
		return err
	}
	err = os.MkdirAll("./test/nodes", os.ModePerm)
	if err != nil {
		return err
	}
	nodeMap := make(map[string]bool)

	for _, node := range nodes {
		// 记录生成过的node文件名
		nodeMap[node.Name+".yml"] = true
		err := genNodeFile(node.Name, node.Status.Address)
		if err != nil {
			return err
		}
	}
	// 删除nodeMap中不存在的文件
	files, err := os.ReadDir("./test/nodes")
	if err != nil {
		return err
	}
	for _, file := range files {
		if _, ok := nodeMap[file.Name()]; !ok {
			log.Println("delete file: ", file.Name())
			err = os.Remove(fmt.Sprintf("./test/nodes/%s", file.Name()))
			if err != nil {
				return err
			}
		}
	}
	for k := range nodeMap {
		delete(nodeMap, k)
	}

	// 建立pods文件夹
	err = os.MkdirAll("./test/pods", os.ModePerm)
	if err != nil {
		return err
	}
	podsMap := make(map[string]bool)

	// 过滤出所有的pod
	pods, err := sc.client.GetAllPods()
	if err != nil {
		return err
	}
	podsMatched := matchPodsByLabels(pods, promeLabels)

	for _, pod := range podsMatched {
		if portStr := pod.Labels["monitorPort"]; portStr != "" {
			podsMap[string(pod.UID)+".yml"] = true
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return err
			}

			err = genPodFile(string(pod.UID), pod.Status.PodIP, port)
			if err != nil {
				return err
			}
		}
	}

	// // 找出pod所在的node
	// for _, pod := range podsMatched {
	// 	for _, node := range nodes {
	// 		podsOnNode, err := sc.client.GetPodsByNodeName(node.Name)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		if len(podsOnNode) == 0 {
	// 			continue
	// 		}
	// 		for _, podOnNode := range podsOnNode {
	// 			if podOnNode.Name == pod.Name {
	// 				if portStr := podOnNode.Labels["monitorPort"]; portStr != "" {
	// 					log.Printf("Add file:%s", string(pod.UID)+".yml")
	// 					podsMap[string(pod.UID)+".yml"] = true
	// 					port, err := strconv.Atoi(portStr)
	// 					if err != nil {
	// 						return err
	// 					}

	// 					err = genPodFile(string(pod.UID), podOnNode.Status.PodIP, port)
	// 					if err != nil {
	// 						return err
	// 					}

	// 				}
	// 			}
	// 		}
	// 	}
	// }
	// 删除podsMap中不存在的文件
	files, err = os.ReadDir("./test/pods")
	if err != nil {
		return err
	}
	for _, file := range files {
		if _, ok := podsMap[file.Name()]; !ok {
			log.Println("delete file: ", file.Name())
			err = os.Remove(fmt.Sprintf("./test/pods/%s", file.Name()))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func genNodeFile(nodeName, nodeIP string) error {
	// 在/test下生成相关的yml文件
	// 文件名为nodeName.yml
	// 文件内容为nodeIP
	lineStr := "- targets:\n" + "  - " + nodeIP + fmt.Sprint(":", magicCAdvisorPort)

	fileName := fmt.Sprintf("./test/nodes/%s.yml", nodeName)

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte(lineStr))
	if err != nil {
		return err
	}
	return nil

}

func genPodFile(podID, podIP string, port int) error {
	// 生成pod的yml文件同理
	lineStr := "- targets:\n" + "  - " + podIP + fmt.Sprint(":", port)

	fileName := fmt.Sprintf("./test/pods/%s.yml", podID)
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write([]byte(lineStr))
	if err != nil {
		return err
	}
	return nil
}

func matchPodsByLabels(pods []*v1.Pod, labels map[string]string) []*v1.Pod {
	// match pods by labels
	var podBelonged []*v1.Pod
	for _, pod := range pods {
		if pod == nil || pod.Labels == nil {
			continue
		}
		belonged := true
		for k, v := range labels {
			if pod.Labels[k] != v {
				belonged = false
				break
			}
		}
		if belonged {
			podBelonged = append(podBelonged, pod)
		}
	}
	return podBelonged
}
