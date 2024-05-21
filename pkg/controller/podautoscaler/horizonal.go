package podautoscaler

import (
	// "minikubernetes/pkg/api/v1"

	"fmt"
	"log"
	"math"
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

	var interval int = 10
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

					// replicaSet目前的副本数
					var curRpsNum int32 = rep.Spec.Replicas

					// 出现过的最大值
					var maxRpsNumAprd int32 = 0

					// 默认选择各种策略下出现的最大值
					selectPolicy := "Max"

					for metricPos, it := range hpa.Spec.Metrics {

						switch {
						case it.Name == v1.ResourceCPU:
							repNum := genRepNumFromCPU(hpa, metricPos, podsMatch, curRpsNum, hc.kube_cli)
							if repNum != -1 {
								maxRpsNumAprd = max(maxRpsNumAprd, repNum)
							}

						case it.Name == v1.ResourceMemory:
							repNum := genRepNumFromMemory(hpa, metricPos, podsMatch, curRpsNum, hc.kube_cli)
							if repNum != -1 {
								maxRpsNumAprd = max(maxRpsNumAprd, repNum)
							}
						default:
							log.Printf("[HPA] Metrics type not supported!\n")
						}
					}

					if selectPolicy == "Max" {
						if maxRpsNumAprd != curRpsNum && maxRpsNumAprd != 0 {
							// 通知apiserver改变ReplicaSet的副本数
							err := hc.changeRpsPodNum(rep.Name, rep.Namespace, maxRpsNumAprd)
							if err != nil {
								log.Printf("[HPA] Change ReplicaSet Pod Num failed, error: %v\n", err)
							}
						} else {
							log.Printf("[HPA] No need to change\n")
						}
					}
				}
			}
		}

	}()
	// ctrl+c
	<-signalCh
	log.Printf("HPA Controller exit\n")
	// log.Printf("HPA Controller exit abnormaly")
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

// func policyByRatio(hpa *v1.HorizontalPodAutoscaler,ratio float32) int32{

//		return 0
//	}
//
// TODOS 由ratio计算副本数的函数抽象成一个
func genRepNumFromCPU(hpa *v1.HorizontalPodAutoscaler, metricTypePos int, podsMatch []*v1.Pod, curRpsNum int32, kube_cli kubeclient.Client) int32 {

	log.Printf("[HPA] genRepNumFromCPU\n")
	//需要取得hpa中相关的策略字段，以获取相关的统计窗口大小
	upBehavior := hpa.Spec.Behavior.ScaleUp
	downBehavior := hpa.Spec.Behavior.ScaleDown
	// 这里假设apiserver在创建的时候就处理了默认策略

	upPeriod := upBehavior.PeriodSeconds
	downPeriod := downBehavior.PeriodSeconds

	log.Printf("[HPA] UpPeriod: %v, DownPeriod: %v\n", upPeriod, downPeriod)

	// spec中metrics的模板，包含资源类型，目标值等
	metricsTplt := hpa.Spec.Metrics[metricTypePos]
	if metricsTplt.Target.Type != v1.UtilizationMetricType {
		log.Printf("[HPA] Warning, use utilization for cpu\n")
		return -1
	}

	// 尝试扩容
	log.Printf("[HPA] Try to upscale by cpu\n")
	upAvg := genAvgCpuUsage(upPeriod, podsMatch, kube_cli)

	var ratio float32
	ratio = float32(upAvg / (metricsTplt.Target.AverageUtilization / 100))

	if ratio > 1.1 {
		var newRepNum int32 = 0
		// 可以扩容
		log.Printf("[HPA] UpScale: %v\n", hpa)

		tryAdd := (int32)(math.Ceil((float64)((ratio - 1) * float32(curRpsNum))))
		if hpa.Spec.Behavior.ScaleUp.Type == v1.PercentScalingPolicy {
			tryAddByPercent := (int32)(math.Ceil((float64)(hpa.Spec.Behavior.ScaleUp.Value * curRpsNum / 100)))
			tryAdd = min(tryAdd, tryAddByPercent)
		} else if hpa.Spec.Behavior.ScaleUp.Type == v1.PodsScalingPolicy {
			tryAdd = min(tryAdd, hpa.Spec.Behavior.ScaleUp.Value)
		} else {
			tryAdd = 1
		}
		log.Printf("[HPA] try Add: %v\n", tryAdd)
		newRepNum = curRpsNum + tryAdd
		if newRepNum > hpa.Spec.MaxReplicas {
			newRepNum = hpa.Spec.MaxReplicas
		}
		return newRepNum
	}

	// 尝试缩容
	log.Printf("[HPA] Try to downscale by cpu\n")
	downAvg := genAvgCpuUsage(downPeriod, podsMatch, kube_cli)

	ratio = float32(downAvg / (metricsTplt.Target.AverageUtilization / 100))

	if ratio < 0.9 {
		var newRepNum int32 = 0
		// 可以缩容
		log.Printf("[HPA] DownScale: %v\n", hpa)

		trySub := (int32)(math.Ceil((float64)((1 - ratio) * float32(curRpsNum))))
		if hpa.Spec.Behavior.ScaleDown.Type == v1.PercentScalingPolicy {
			trySubByPercent := (int32)(math.Ceil((float64)(hpa.Spec.Behavior.ScaleDown.Value * curRpsNum / 100)))
			trySub = min(trySub, trySubByPercent)
		} else if hpa.Spec.Behavior.ScaleDown.Type == v1.PodsScalingPolicy {
			trySub = min(trySub, hpa.Spec.Behavior.ScaleDown.Value)
		} else {
			trySub = 1
		}
		log.Printf("[HPA] try Sub: %v\n", trySub)
		newRepNum = curRpsNum - trySub
		if newRepNum < hpa.Spec.MinReplicas {
			newRepNum = hpa.Spec.MinReplicas
		}
		return newRepNum
	}

	// 保持不变
	log.Printf("[HPA] In tolerance, keep the same: %v\n", hpa)
	return curRpsNum
}

func genAvgCpuUsage(windowSz int32, podsMatch []*v1.Pod, kube_cli kubeclient.Client) float32 {
	// 当前时间戳
	now := time.Now()

	allPodCpuAvg := float32(0)
	// 根据pod的Id获取所有的Metrics
	for _, pod := range podsMatch {
		// 1. 处理扩容的请求
		oneQuery := v1.MetricsQuery{
			UID:       pod.UID,
			TimeStamp: now,
			Window:    windowSz,
		}

		// 一个pod的统计参数
		oneMetrics, err := kube_cli.GetPodMetrics(oneQuery)
		if err != nil {
			log.Printf("Get Pod Metrics failed, error: %v\n", err)
			// return err
		}

		if oneMetrics == nil {
			log.Printf("Pod Metrics not found\n")
			continue
		}
		if len(oneMetrics.ContainerInfo) == 0 {
			log.Printf("Wait for ContainerInfo to become valid\n")
			continue
		}

		var podCpuSum float32 = 0
		// 单个container的cpu使用率
		var sum, avg float32

		// 对于每个container 统计其cpu使用率
		// 目前是计算时间窗口内的平均使用率
		for _, oneCtnr := range oneMetrics.ContainerInfo {
			sum = 0
			avg = 0
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
	return allPodCpuAvg
}

func genRepNumFromMemory(hpa *v1.HorizontalPodAutoscaler, metricPos int, podsMatch []*v1.Pod, curRpsNum int32, kube_cli kubeclient.Client) int32 {
	log.Printf("[HPA] genRepNumFromMemory\n")
	//需要取得hpa中相关的策略字段，以获取相关的统计窗口大小
	upBehavior := hpa.Spec.Behavior.ScaleUp
	downBehavior := hpa.Spec.Behavior.ScaleDown

	upPeriod := upBehavior.PeriodSeconds
	downPeriod := downBehavior.PeriodSeconds
	log.Printf("[HPA] UpPeriod: %v, DownPeriod: %v\n", upPeriod, downPeriod)

	// spec中metrics的模板，包含资源类型，目标值等
	metricsTplt := hpa.Spec.Metrics[metricPos]
	fmt.Print(string(metricsTplt.Name))
	if metricsTplt.Target.Type != v1.AverageValueMetricType {
		log.Printf("[HPA] Warning, use AverageValue for memory\n")
		return -1
	}

	// 尝试扩容
	log.Printf("[HPA] Try to upscale by memory\n")
	upAvg := genAvgMemoryUsage(upPeriod, podsMatch, kube_cli)

	var ratio float32
	ratio = float32(upAvg / metricsTplt.Target.AverageValue)

	// 因为内存的抖动可能会更大，所以这里将阈值设置得高一点
	if ratio > 1.5 {
		var newRepNum int32 = 0
		// 可以扩容
		log.Printf("[HPA] UpScale: %v\n", hpa)

		tryAdd := (int32)(math.Ceil((float64)((ratio - 1) * float32(curRpsNum))))
		if hpa.Spec.Behavior.ScaleUp.Type == v1.PercentScalingPolicy {
			tryAddByPercent := (int32)(math.Ceil((float64)(hpa.Spec.Behavior.ScaleUp.Value * curRpsNum / 100)))
			tryAdd = min(tryAdd, tryAddByPercent)
		} else if hpa.Spec.Behavior.ScaleUp.Type == v1.PodsScalingPolicy {
			tryAdd = min(tryAdd, hpa.Spec.Behavior.ScaleUp.Value)
		} else {
			tryAdd = 1
		}
		log.Printf("[HPA] try Add: %v\n", tryAdd)
		newRepNum = curRpsNum + tryAdd
		if newRepNum > hpa.Spec.MaxReplicas {
			newRepNum = hpa.Spec.MaxReplicas
		}
		return newRepNum
	}

	// 尝试缩容
	log.Printf("[HPA] Try to downscale by memory\n")
	downAvg := genAvgMemoryUsage(downPeriod, podsMatch, kube_cli)

	ratio = float32(downAvg / metricsTplt.Target.AverageValue)

	if ratio < 0.5 {
		var newRepNum int32 = 0
		// 可以缩容
		log.Printf("[HPA] DownScale: %v\n", hpa)

		trySub := (int32)(math.Ceil((float64)((1 - ratio) * float32(curRpsNum))))
		if hpa.Spec.Behavior.ScaleDown.Type == v1.PercentScalingPolicy {
			trySubByPercent := (int32)(math.Ceil((float64)(hpa.Spec.Behavior.ScaleDown.Value * curRpsNum / 100)))
			trySub = min(trySub, trySubByPercent)
		} else if hpa.Spec.Behavior.ScaleDown.Type == v1.PodsScalingPolicy {
			trySub = min(trySub, hpa.Spec.Behavior.ScaleDown.Value)
		} else {
			trySub = 1
		}
		log.Printf("[HPA] try Sub: %v\n", trySub)
		newRepNum = curRpsNum - trySub
		if newRepNum < hpa.Spec.MinReplicas {
			newRepNum = hpa.Spec.MinReplicas
		}

		return newRepNum
	}

	// 保持不变
	log.Printf("[HPA] In tolerance, keep the same: %v\n", hpa)
	return curRpsNum
}

// TODO 将获取平均值的函数抽象成一个
// 但是memory比较特殊 如果没有大的变化 新的
func genAvgMemoryUsage(windowSz int32, podsMatch []*v1.Pod, kube_cli kubeclient.Client) float32 {
	// 当前时间戳
	now := time.Now()

	allPodMemAvg := float32(0)
	// 根据pod的Id获取所有的Metrics
	for _, pod := range podsMatch {
		// 1. 处理扩容的请求
		oneQuery := v1.MetricsQuery{
			UID:       pod.UID,
			TimeStamp: now,
			Window:    windowSz,
		}

		// 一个pod的统计参数
		oneMetrics, err := kube_cli.GetPodMetrics(oneQuery)
		if err != nil {
			log.Printf("Get Pod Metrics failed, error: %v\n", err)
			// return err
		}

		if oneMetrics == nil {
			log.Printf("Pod Metrics not found\n")
			continue
		}
		if len(oneMetrics.ContainerInfo) == 0 {
			log.Printf("Wait for ContainerInfo to become valid\n")
			continue
		}

		var podMemSum float32 = 0
		// 单个container的内存使用率
		var sum, avg float32

		// 对于每个container 统计其内存使用率
		// 目前是计算时间窗口内的平均使用率
		for _, oneCtnr := range oneMetrics.ContainerInfo {
			sum = 0
			avg = 0
			for _, i := range oneCtnr {
				sum += i.MemoryUsage
			}
			if len(oneCtnr) != 0 {
				avg = sum / float32(len(oneCtnr))
				log.Printf("[HPA] Pod Metrics avg memory / one container: %v\n", avg)
			}
			podMemSum += avg
		}
		podMemAvg := podMemSum / float32(len(oneMetrics.ContainerInfo))

		// 计算Pod的平均内存使用率
		// 默认是所有container的平均值
		log.Printf("[HPA] Pod Metrics avg memory / all containers in one pod: %v\n", podMemAvg)
		allPodMemAvg += podMemAvg
	}

	// 计算所有Pod的平均内存使用率
	allPodMemAvg = allPodMemAvg / float32(len(podsMatch))
	log.Printf("[HPA] All Pods Metrics avg memory / all pods: %v\n", allPodMemAvg)
	return allPodMemAvg
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
