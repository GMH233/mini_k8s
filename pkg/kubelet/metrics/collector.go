// package metrics
package main // test

import (
	"bytes"
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"os/exec"
	"strings"

	CAdvClient "github.com/google/cadvisor/client/v2"
	Info "github.com/google/cadvisor/info/v2"
)

func main() {
	mc, err := NewMetricsCollector()
	if err != nil {
		fmt.Printf("new metrics collector err: %v", err.Error())
	}
	mc.Run()
}

type metricsCollector struct {
	// 从cadvisor中获取容器指标的操作者
	ip         string
	port       int
	cad_client *CAdvClient.Client

	// kubeclient,和apiserver的stats接口交互

	pods *[]v1.Pod
}

type MetricsCollector interface {
	// MetricsCollector接口
	SetPodInfo()
	Run()
}

func NewMetricsCollector() (MetricsCollector, error) {
	// 创建一个新的MetricsCollector
	var newMetricsCollector MetricsCollector
	newMetricsCollector = &metricsCollector{
		ip:   "localhost",
		port: 8090,
	}
	return newMetricsCollector, nil
}

func (mc *metricsCollector) Run() {
	// 运行MetricsCollector
	mc.init()

	startUp := true
	// 定期检查cadvisor是否存活
	// TODO: routine check

	err := mc.CheckAlive()
	if err != nil {
		// 启动cadvisor
		if startUp {
			// cmd := exec.Command("docker", "run",
			// 	"--volume=/:/rootfs:ro",
			// 	"--volume=/var/run:/var/run:rw",
			// 	"--volume=/sys:/sys:ro",
			// 	"--volume=/var/lib/docker/:/var/lib/docker:ro",
			// 	"--publish="+fmt.Sprint(mc.port)+":8080",
			// 	"--detach=true",
			// 	"--name=cadvisor",
			// 	"google/cadvisor:latest")
			// var stdout, stderr bytes.Buffer
			// cmd.Stdout = &stdout
			// cmd.Stderr = &stderr
			// err = cmd.Run()
			// if err != nil {
			// 	fmt.Printf("start cadvisor err: %v, stderr: %v", err.Error(), stderr.String())
			// } else {
			// 	startUp = false
			// }
		}
	}

	// 获取容器指标
	mc.getMetrics()

	// TODO: 上传指标

}

func (mc *metricsCollector) init() {
	// 初始化MetricsCollector
	client, err := CAdvClient.NewClient("http://" + mc.ip + ":" + fmt.Sprint(mc.port))
	mc.cad_client = client
	if err != nil {
		fmt.Printf("init cadvisor client err: %v", err.Error())
	}

}

func (mc *metricsCollector) SetPodInfo() {
	// 由kubelet设置pod信息
	// 注意是Pod 到 容器非人的字符串 的映射

}

// containername/containerid,type, timeinterval
func (mc *metricsCollector) getMetrics() {
	// 指定字段从cadvisor中获取容器指标
	// Count是获取的
	const NANOCORES float64 = 1000000000
	const MEGABYTE float64 = 1024 * 1024
	var interval int = 10 + 1 // 第一个必然是nil

	request := Info.RequestOptions{
		IdType:    Info.TypeName,
		Count:     interval,
		Recursive: false,
		MaxAge:    nil,
	}

	// 容器非人的字符串
	var dockerId string = "/docker/8416a8e063e9a30961be4f88d5bfbad3b8b6911fc4a6c36ec7bd73729ecd5e38"
	sInfo, err := mc.cad_client.Stats(dockerId, &request)
	if err != nil {
		fmt.Printf("get container info err: %v", err.Error())
	}
	// fmt.Printf("container info: %v\n", sInfo)
	// sInfo的key是dockerId

	// 这样就可以获得cpu瞬时的占用率了,total_usage以nano core为单位.
	for _, item := range sInfo[dockerId].Stats {
		// 第一个item必然是nil
		if item.CpuInst != nil {
			total_usage := item.CpuInst.Usage.Total
			// vcpu个数
			cpu_usage := float64(total_usage) / NANOCORES
			fmt.Printf("container stats:%v\n", cpu_usage)

		}

	}

	// 还有是容器的内存占用率,以byte为单位
	for _, item := range sInfo[dockerId].Stats {
		if item.Memory != nil {
			memory_usage := item.Memory.Usage
			// 以mb计量
			mb_usage := float64(memory_usage) / MEGABYTE
			fmt.Printf("container memory stats:%v\n", mb_usage)
		}
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
	cmd = exec.Command("nc", "-vz", mc.ip, string(mc.port))
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ping cadvisor err: %v", err.Error())
	}
	if !strings.Contains(stdout.String(), "succeeded") {
		return fmt.Errorf("cannot connect to cadvisor")
	}
	return nil
}

func (mc *metricsCollector) uploadMetrics() {
	// 将容器指标上传到apiserver
}
