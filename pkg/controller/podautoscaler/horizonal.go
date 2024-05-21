package podautoscaler

import (
	// "minikubernetes/pkg/api/v1"

	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"os"
	"os/signal"
	"time"
)

type HorizonalController interface {
	Run()
}
type horizonalController struct {
	kube_cli kubeclient.Client
}

func NewHorizonalController(apiServerIP string) HorizonalController {
	return &horizonalController{
		kube_cli: kubeclient.NewClient(apiServerIP),
	}
}
func (hc *horizonalController) Run() {

	var interval int = 5
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	log.Printf("Start HPA Controller\n")
	// 现在只处理一个scaler的情况
	// 如果之后有多个scaler，分别起协程定时处理
	// 这样还得有一个类似pleg的东西，定时检查是否有新的scaler
	go func() {
		syncTicker := time.NewTicker(time.Duration(interval) * time.Second)
		defer syncTicker.Stop()
		defer log.Printf("Stop HPA Controller\n")
		for {
			select {
			case <-syncTicker.C:

				// allPods []*v1.Pod
				// allReplicaSets []*v1.ReplicaSet
				// allHPAScalers []*v1.HorizontalPodAutoscaler

				// 1. 获取所有的HorizontalPodAutoscaler
				allHPAScalers, err := hc.kube_cli.GetAllHPAScalers()
				log.Printf("[HPA] Get all HorizontalPodAutoscaler: %v", allHPAScalers)
				if err != nil {
					log.Printf("Get all HorizontalPodAutoscaler failed, error: %v", err)
					// return err
				}
				if len(allHPAScalers) == 0 {
					log.Printf("No HorizontalPodAutoscaler found\n")
					continue
				}

				// 2. 获取所有的Pod
				allPods, err := hc.kube_cli.GetAllPods()
				log.Printf("[Pod] Get all Pods: %v", allPods)
				if err != nil {
					log.Printf("Get all Pods failed, error: %v", err)
					// return err
				}

				for _, hpa := range allHPAScalers {
					log.Printf("[HPA] Conducting HPA: %v\n", hpa)
					reps, err := hc.kube_cli.GetAllReplicaSets()
					if err != nil {
						log.Printf("Get all ReplicaSets failed, error: %v\n", err)
						// return err
					}
					// 根据hpa中spec的scaleTargetRef找到对应的ReplicaSet(目前只能有一个)
					rep, err := oneMatchRps(hpa, reps)
					if err != nil {
						log.Printf("Get all ReplicaSets failed, error: %v\n", err)
						// return err
					}
					if rep == nil {
						log.Printf("ReplicaSet not found\n")
						continue
					}
					log.Printf("[HPA] ReplicaSet Matched:   %v\n", rep)
					// 根据ReplicaSet的labels筛选所有的Pod
					podsMatch, err := oneMatchRpsLabels(rep, allPods)

					log.Printf("[HPA] Pods Matched: %v\n", podsMatch)

					//需要取得hpa中相关的策略字段，以获取相关的统计窗口大小
					upBehavior := hpa.Spec.Behavior.ScaleUp
					downBehavior := hpa.Spec.Behavior.ScaleDown
					// 这里假设apiserver在创建的时候就处理了默认策略

					upPeriod := upBehavior.PeriodSeconds
					downPeriod := downBehavior.PeriodSeconds

					log.Printf("[HPA] UpPeriod: %v, DownPeriod: %v\n", upPeriod, downPeriod)
					// 当前时间戳
					now := time.Now()

					// 根据pod的Id获取所有的Metrics
					for _, pod := range podsMatch {
						// A. 处理扩容的请求
						oneQuery := v1.MetricsQuery{
							UID:       pod.UID,
							TimeStamp: now,
							Window:    upPeriod,
						}

						oneMetrics, err := hc.kube_cli.GetPodMetrics(oneQuery)
						if err != nil {
							log.Printf("Get Pod Metrics failed, error: %v\n", err)
							// return err
						}

						if oneMetrics == nil {
							log.Printf("Pod Metrics not found\n")
							continue
						}

						// log.Printf("[HPA] Pod Metrics: %v\n", oneMetrics)
						// 根据HorizontalPodAutoscaler的策略进行扩缩容

						var sum, avg float32
						for _, oneCtnr := range oneMetrics.ContainerInfo {
							for _, i := range oneCtnr {
								sum += i.CPUUsage
							}
							if len(oneCtnr) != 0 {
								avg = sum / float32(len(oneCtnr))
							}
						}
						// avg = sum / float32(len(oneMetrics.ContainerInfo))
						log.Printf("[HPA] Pod Metrics avg cpu / all containers: %v\n", avg)
					}
				}

				// 6. 根据HorizontalPodAutoscaler的策略进行扩缩容
				// 7. 向apiserver发送请求改变ReplicaSet的副本数
			}
		}

	}()
	// 获取到ctrl+c信号
	<-signalCh
	log.Printf("HPA Controller exit\n")
	// log.Printf("HPA Controller exit abnormaly")
}

// 策略函数
func (hc *horizonalController) Strategy() {

}

// 向apiserver发送请求改变ReplicaSet的副本数
func (hc *horizonalController) changeRpsPodNum(name string, namespace string, repNum int) error {
	return nil
}

// 从apiServer里面获取相关的ReplicaSet
func oneMatchRps(hpa *v1.HorizontalPodAutoscaler, reps []*v1.ReplicaSet) (*v1.ReplicaSet, error) {
	refRpsName := hpa.Spec.ScaleTargetRef.Name
	// 默认在和hpa同一个namespace下去找hpa
	refRpsNamespace := hpa.Namespace

	for _, rep := range reps {
		if rep.Name == refRpsName && rep.Namespace == refRpsNamespace {
			return rep, nil
		}
	}
	return nil, fmt.Errorf("ReplicaSet not found")

}

// 根据ReplicaSet的labels筛选所有的Pod
func oneMatchRpsLabels(rep *v1.ReplicaSet, allPods []*v1.Pod) ([]*v1.Pod, error) {
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
