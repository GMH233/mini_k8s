package metrics // test

import (
	"bytes"
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"minikubernetes/pkg/kubelet/runtime"
	"os/exec"
	"strings"
	"sync"
	"time"

	CAdvClient "github.com/google/cadvisor/client/v2"
	Info "github.com/google/cadvisor/info/v2"
)

// func main() {
// 	mc, err := NewMetricsCollector()
// 	if err != nil {
// 		fmt.Printf("new metrics collector err: %v", err.Error())
// 	}
// 	mc.Run()
// }

var mdebug bool = true

type metricsCollector struct {
	// 从cadvisor中获取容器指标的操作者
	ip         string
	port       int
	cad_client *CAdvClient.Client

	// kubeclient,和apiserver的stats接口交互
	kube_cli kubeclient.Client
	// podStat得加锁
	podStatsLock sync.Mutex
	podStats     []*runtime.PodStatus
	conLastTime  map[string]time.Time
}

type MetricsCollector interface {
	// MetricsCollector接口
	SetPodInfo(podStats []*runtime.PodStatus)
	Run()
}

func NewMetricsCollector() MetricsCollector {
	// 创建一个新的MetricsCollector
	var newMetricsCollector MetricsCollector
	newMetricsCollector = &metricsCollector{
		ip:   "127.0.0.1",
		port: 8090,
	}
	return newMetricsCollector
}
func (mc *metricsCollector) tryStartCAdvisor() {
	cmd := exec.Command("docker", "run",
		"--volume=/:/rootfs:ro",
		"--volume=/var/run:/var/run:rw",
		"--volume=/sys:/sys:ro",
		"--volume=/var/lib/docker/:/var/lib/docker:ro",
		"--publish="+fmt.Sprint(mc.port)+":8080",
		"--detach=true",
		"--name=cadvisor",
		"google/cadvisor:latest")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("start cadvisor err: %v, stderr: %v", err.Error(), stderr.String())
	}
}

func (mc *metricsCollector) Run() {
	// 运行MetricsCollector
	mc.init()

	const interval int = 5

	// 定期检查cadvisor是否存活
	go func() {
		syncTicker := time.NewTicker(time.Duration(interval) * time.Second)
		defer syncTicker.Stop()
		for {
			select {
			case <-syncTicker.C:
				err := mc.CheckAlive()
				if err != nil {
					// 尝试启动cadvisor
					fmt.Printf(err.Error())
					mc.tryStartCAdvisor()
				} else {
					if mc.podStats != nil {

						// 获取容器指标
						curMetrics, err := mc.getMetrics()
						if err != nil {
							fmt.Printf("get metrics err: %v", err.Error())
						}
						log.Printf("metrics to be uploaded: %v", curMetrics)
						// 上传指标
						err = mc.uploadMetrics(curMetrics)
						if err != nil {
							fmt.Printf("upload metrics err: %v", err.Error())
						}
					}
				}
			}
		}
	}()

}

func (mc *metricsCollector) init() {
	err := mc.CheckAlive()
	if err != nil {
		// 尝试启动cadvisor
		mc.tryStartCAdvisor()
	}
	// 初始化MetricsCollector
	URLStr := fmt.Sprintf("http://%s:%d", mc.ip, mc.port)
	client, err := CAdvClient.NewClient(URLStr)
	mc.cad_client = client
	if err != nil {
		fmt.Printf("init cadvisor client err: %v", err.Error())
	}

	mc.kube_cli = kubeclient.NewClient("192.168.1.10")

	mc.conLastTime = make(map[string]time.Time)
	mc.podStats = nil

}

func (mc *metricsCollector) SetPodInfo(podStats []*runtime.PodStatus) {
	// 由kubelet设置pod信息
	// 注意是Pod 到 容器非人的字符串 的映射
	log.Printf("set pod info: %v", podStats)
	mc.podStatsLock.Lock()
	if mc.podStats == nil {
		mc.podStats = make([]*runtime.PodStatus, 0)
	}
	mc.podStats = podStats
	mc.podStatsLock.Unlock()
}

// containername/containerid,type, timeinterval
func (mc *metricsCollector) getMetrics() ([]*v1.PodRawMetrics, error) {
	// 指定字段从cadvisor中获取容器指标
	// Count是获取的
	const NANOCORES float64 = 1e9
	const MEGABYTE float64 = 1024 * 1024
	var interval int = 10 + 1 // 第一个必然是nil

	request := Info.RequestOptions{
		IdType:    Info.TypeName,
		Count:     interval,
		Recursive: false,
		MaxAge:    new(time.Duration),
	}
	*request.MaxAge = time.Duration(0)

	// 保存所有的pod指标
	var AllPodMetrics []*v1.PodRawMetrics

	mc.podStatsLock.Lock()
	// 对于每一个Pod:都得获取指标
	for _, podStat := range mc.podStats {
		var podMetrics v1.PodRawMetrics
		podId := podStat.ID
		podMetrics.UID = podId
		podMetrics.ContainerInfo = make(map[string][]v1.PodRawMetricsItem)

		// 对于每一个容器
		for _, container := range podStat.ContainerStatuses {
			// 只统计running的容器
			if container.State != runtime.ContainerStateRunning {
				continue
			}
			// 容器的非人的字符串
			var dockerId string = container.ID
			// 如果容器对应的map不存在，则创建
			if _, ok := podMetrics.ContainerInfo[dockerId]; !ok {
				podMetrics.ContainerInfo[dockerId] = make([]v1.PodRawMetricsItem, 0)
			}
			var PrefixDockerId string = fmt.Sprintf("/docker/%s", container.ID)
			MapPfxIdStats, err := mc.cad_client.Stats(PrefixDockerId, &request)

			if err != nil {
				fmt.Printf("get container info err: %v", err.Error())
			}
			// fmt.Printf("container info: %v\n", sInfo)
			// Map_PfxId_Stats的key是PrefixDockerId
			// 这样就可以获得cpu瞬时的占用率了,total_usage以nano core为单位.
			for _, item := range MapPfxIdStats[PrefixDockerId].Stats {
				var cMetricItem v1.PodRawMetricsItem
				// 第一个item必然是nil
				if item.CpuInst == nil || item.Memory == nil {
					continue
				}
				//如果比上次时间戳还小，就不用上传
				if lastTime, ok := mc.conLastTime[dockerId]; ok {
					if !item.Timestamp.After(lastTime) {
						fmt.Printf("时间戳不早于上次，不用上传\n")
						continue
					}
				}
				// 更新时间戳
				fmt.Printf("更新时间戳\n")
				mc.conLastTime[dockerId] = item.Timestamp

				cMetricItem.TimeStamp = item.Timestamp

				{
					total_usage := item.CpuInst.Usage.Total

					// 需要转化成利用率，有点反直觉，但是不用除以物理核心数
					cpu_usage := (float64(total_usage) / NANOCORES)
					cMetricItem.CPUUsage = cpu_usage
					// fmt.Printf("container stats:%v\n", cpu_usage)

				}

				{
					memory_usage := item.Memory.Usage
					// 以mb计量
					mb_usage := float64(memory_usage) / MEGABYTE
					cMetricItem.MemoryUsage = mb_usage
					// fmt.Printf("container memory stats:%v\n", mb_usage)
				}

				// TODO: 增加对于网络占用率的统计
				// 合理：容器的网络占用率[积累量]
				// for _, item := range sInfo[dockerId].Stats {
				// 	if item.Network != nil {
				// 		// 以byte为单位
				// 		rx_bytes := item.Network.Interfaces[0].RxBytes // 接受的byte数量
				// 		tx_bytes := item.Network.Interfaces[0].TxBytes // 传出去的byte数量
				// 		fmt.Printf("container network stats: rx:%v, tx:%v\n", rx_bytes, tx_bytes)
				// 	}
				// }

				// TODO: 增加对于容器的disk io占用率的统计
				// 待测量
				// for _, item := range sInfo[dockerId].Stats {
				// 	if item.DiskIo != nil {
				// 		// 以byte为单位
				// 		read_bytes := item.DiskIo.IoServiceBytes[0].Stats
				// 		write_bytes := item.DiskIo.IoServiceBytes[1].Stats
				// 		fmt.Printf("container disk io stats: read:%v, write:%v\n", read_bytes, write_bytes)
				// 	}
				// }
				podMetrics.ContainerInfo[dockerId] = append(podMetrics.ContainerInfo[dockerId], cMetricItem)
			}

		}
		// var dockerId string = "/docker/8416a8e063e9a30961be4f88d5bfbad3b8b6911fc4a6c36ec7bd73729ecd5e38"

		AllPodMetrics = append(AllPodMetrics, &podMetrics)
	}
	mc.podStatsLock.Unlock()
	return AllPodMetrics, nil
}

func (mc *metricsCollector) CheckAlive() error {
	// 检查cadvisor是否存活
	cmd := exec.Command("docker", "ps")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// cadvisor未启动
		return fmt.Errorf("cadvisor err: %v, stderr: %v", err.Error(), stderr.String())
	}

	// 检查是否有cadvisor 字段
	if !strings.Contains(stdout.String(), "cadvisor") {
		return fmt.Errorf("cadvisor not found in docker ps")
	}

	// 检查是否能够获取端口信息
	cmd = exec.Command("nc", "-vz", mc.ip, fmt.Sprintf("%d", mc.port))
	stdout.Reset()
	stderr.Reset()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ping cadvisor err: %v, %v , %v", err.Error(), stderr.String(), stdout.String())
	}
	return nil
}

func (mc *metricsCollector) uploadMetrics(metrics []*v1.PodRawMetrics) error {
	// 将容器指标上传到apiserver
	// 调用kubeclient中的上传函数
	err := mc.kube_cli.UploadPodMetrics(metrics)
	if err != nil {
		return err
	}
	return nil
}
