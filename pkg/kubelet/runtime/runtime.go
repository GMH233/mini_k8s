package runtime

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	v1 "minikubernetes/pkg/api/v1"
	"time"
)

// 底层pod表示
type Pod struct {
	// pod的UID
	ID        v1.UID
	Name      string
	Namespace string

	// 纳秒
	CreatedAt uint64

	Containers []*Container
}

type Container struct {
	// docker为container生成的id
	ID string
	// 和v1.Container.Name相同
	Name string

	Image string

	State ContainerState
}

// State represents the state of a container
type ContainerState string

const (
	ContainerStateCreated ContainerState = "created"

	ContainerStateRunning ContainerState = "running"

	ContainerStateExited ContainerState = "exited"

	ContainerStateUnknown ContainerState = "unknown"
)

// 底层的pod状态表示
type PodStatus struct {
	// pod的UID
	ID        v1.UID
	Name      string
	Namespace string
	// CNI赋予的ip
	IPs []string
	// 所有容器状态
	ContainerStatuses []*ContainerStatus
	// 生成该条status记录的时间
	TimeStamp time.Time
}

type ContainerStatus struct {
	// docker为container生成的id
	ID         string
	Name       string
	State      ContainerState
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int
	Image      string
	Resources  *ContainerResources
}

type ContainerResources struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

func GetContainerBridgeIP(containerName string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	defer cli.Close()
	f := filters.NewArgs()
	f.Add("name", containerName)
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{
		Filters: f,
	})
	if err != nil {
		return "", err
	}
	if len(containers) == 0 {
		return "", fmt.Errorf("container %s not found", containerName)
	}
	c := containers[0]
	if ep, ok := c.NetworkSettings.Networks["bridge"]; !ok {
		return "", fmt.Errorf("container %s does not have bridge network", containerName)
	} else {
		return ep.IPAddress, nil
	}
}
