package runtime

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	v1 "minikubernetes/pkg/api/v1"
)

type RuntimeManager interface {
	AddPod(pod *v1.Pod) error
}

type runtimeManager struct {
}

func NewRuntimeManager() RuntimeManager {
	manager := &runtimeManager{}
	return manager
}

func (rm *runtimeManager) AddPod(pod *v1.Pod) error {

	PauseId, err := rm.CreatePauseContainer()
	if err != nil {
		panic(err)
	}

	containerList := pod.Spec.Containers
	for _, container := range containerList {
		_, err := rm.createContainer(&container, PauseId)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (rm *runtimeManager) CreatePauseContainer() (string, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	PauseContainerImage := "registry.aliyuncs.com/google_containers/pause:3.6"
	exi, err := rm.checkImages(PauseContainerImage)
	if err != nil {
		panic(err)
	}
	if !exi {
		fmt.Println("yes")
		reader, err := cli.ImagePull(ctx, PauseContainerImage, image.PullOptions{})
		if err != nil {
			panic(err)
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	} else {
		fmt.Println("already exist")
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: PauseContainerImage,
		Tty:   false,
		//ExposedPorts: exposedPortSet,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	return resp.ID, nil
}

func (rm *runtimeManager) createContainer(ct *v1.Container, PauseId string) (string, error) {
	repotag := ct.Image
	cmd := ct.Command
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	exi, err := rm.checkImages(repotag)
	if err != nil {
		panic(err)
	}
	if !exi {
		fmt.Println("yes")
		reader, err := cli.ImagePull(ctx, "docker.io/library/"+repotag, image.PullOptions{})
		if err != nil {
			panic(err)
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	} else {
		fmt.Println("already exist")
	}

	ports := ct.Ports
	exposedPortSet := nat.PortSet{}
	portKeys := make(map[string]struct{})

	for _, num := range ports {
		fmt.Println(num.ContainerPort)

		key := fmt.Sprint(num.ContainerPort) + "/tcp"
		fmt.Println(key)
		portKeys[key] = struct{}{}
	}
	for key := range portKeys {
		exposedPortSet[nat.Port(key)] = struct{}{}
	}

	pauseRef := "container:" + PauseId

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: repotag,
		Cmd:   cmd,
		Tty:   true,
		//ExposedPorts: exposedPortSet,
	}, &container.HostConfig{
		NetworkMode: container.NetworkMode(pauseRef),
		// Binds:        option.Binds,
		// PortBindings: option.PortBindings,
		//IpcMode: container.IpcMode(pauseRef),
		PidMode: container.PidMode(pauseRef),
		// VolumesFrom:  option.VolumesFrom,
		// Links:        option.Links,
		// Resources: container.Resources{
		// 	Memory:   option.MemoryLimit,
		// 	NanoCPUs: option.CPUResourceLimit,
		// },
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	// statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// case <-statusCh:
	// }

	// out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	// if err != nil {
	// 	panic(err)
	// }

	// stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return resp.ID, nil
}

func (rm *runtimeManager) checkImages(repotag string) (bool, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		fmt.Println("fail to get images", err)
		panic(err)
	}
	fmt.Println("Docker Images:")
	for _, image := range images {
		//fmt.Printf("ID: %s\n", image.ID)
		//fmt.Printf("RepoTags: %v\n", image.RepoTags)
		if image.RepoTags[0] == repotag {
			return true, nil
		}
		//fmt.Printf("Size: %d\n", image.Size)
		//fmt.Println("------------------------")
	}
	return false, nil
}
