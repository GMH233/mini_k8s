package podautoscaler

import (
	// "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
)

type HorizontalController struct {
	kube_cli kubeclient.Client
}

// 策略函数
func (hc *HorizontalController) Strategy() {

}

// 向apiserver发送请求改变pod数量

// 从apiServer中获取统计的信息
func GetPodMetrics() {

}

// 从apiServer中获取所有HorizontalPodAutoscaler的信息
func GetAutoscalers() {

}
func GetRelatedRepSets() {

}
