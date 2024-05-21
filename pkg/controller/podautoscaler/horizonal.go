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
					if err != nil {
						log.Printf("Get all Pods failed, error: %v\n", err)
						// return err
					}
					if len(podsMatch) == 0 {
						log.Printf("No matched pods!\n")
						continue
					}

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

					allPodCpuAvg := float32(0)
					// 根据pod的Id获取所有的Metrics
					for _, pod := range podsMatch {
						// 1. 处理扩容的请求
						oneQuery := v1.MetricsQuery{
							UID:       pod.UID,
							TimeStamp: now,
							Window:    upPeriod,
						}

						// 一个pod的统计参数
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

						// A. 统计CPU
						var podCpuSum float32 = 0
						// 单个container的cpu使用率
						var sum, avg float32

						// 对于每个container 统计其cpu使用率
						// 目前是计算时间窗口内的平均使用率
						for _, oneCtnr := range oneMetrics.ContainerInfo {
							sum = 0
							for _, i := range oneCtnr {
								sum += i.CPUUsage
							}
							if len(oneCtnr) != 0 {
								avg = sum / float32(len(oneCtnr))
								log.Printf("[HPA] Pod Metrics avg cpu / one container: %v\n", avg)
							}
							podCpuSum += avg
						}
						podCpuAvg := podCpuSum / float32(len(oneMetrics.ContainerInfo))

						// 计算Pod的平均cpu使用率
						// 默认是所有container的平均值
						log.Printf("[HPA] Pod Metrics avg cpu / all containers in one pod: %v\n", podCpuAvg)
						allPodCpuAvg += podCpuAvg
					}

					// 计算所有Pod的平均cpu使用率
					allPodCpuAvg = allPodCpuAvg / float32(len(podsMatch))
					log.Printf("[HPA] All Pods Metrics avg cpu / all pods: %v\n", allPodCpuAvg)

					// 准备根据CPU扩容
					// 其实应该先找metrics,看看有没有cpu这一项
					// 在metrics里面找到cpu这一项
					// 目前仅允许有一项

					for _, metricsTplt := range hpa.Spec.Metrics {

						if metricsTplt.Name == v1.ResourceCPU {
							if metricsTplt.Target.Type == v1.UtilizationMetricType {
								if allPodCpuAvg > float32(metricsTplt.Target.UpperThreshold/100) {
									// 可以扩容
									log.Printf("[HPA] UpScale: %v\n", hpa)
									newRepNum := rep.Spec.Replicas + 1
									if newRepNum > hpa.Spec.MaxReplicas {
										newRepNum = hpa.Spec.MaxReplicas
									}
									// 向apiserver发送请求改变ReplicaSet的副本数
									err := hc.changeRpsPodNum(rep.Name, rep.Namespace, newRepNum)
									if err != nil {
									}
								} else if allPodCpuAvg < float32(metricsTplt.Target.LowerThreshold/100) {
									// 可以缩容
									log.Printf("[HPA] DownScale: %v\n", hpa)
									newRepNum := rep.Spec.Replicas - 1
									if newRepNum < hpa.Spec.MinReplicas {
										newRepNum = hpa.Spec.MinReplicas
									}
									// 向apiserver发送请求改变ReplicaSet的副本数
									err := hc.changeRpsPodNum(rep.Name, rep.Namespace, newRepNum)
									if err != nil {
									}

								}
							}
						}
					}

					// TODO B. 统计Memory
				}
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
func (hc *horizonalController) changeRpsPodNum(name string, namespace string, repNum int32) error {
	fmt.Printf("changeRpsPodNum: %v, %v, %v\n", name, namespace, repNum)
	err := hc.kube_cli.UpdateReplicaSet(name, namespace, repNum)
	if err != nil {
		return err
	}

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
