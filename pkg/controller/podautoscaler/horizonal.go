package podautoscaler

type HorizontalController struct {
}

// 目前直接从collector中获取信息
// 之后可以从apiserver的metrics接口获取

// 策略函数
func (hc *HorizontalController) Strategy() {

}

// 向apiserver发送请求改变pod数量

// 从collector中获取信息
func GetPodMetrics() {

}
