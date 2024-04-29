package runtime

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types/mount"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"

	v1 "minikubernetes/pkg/api/v1"
	nw "minikubernetes/pkg/kubelet/network"
)

type RuntimeManager interface {
	AddPod(pod *v1.Pod) error
	GetAllPods() ([]*Pod, error)
	GetPodStatus(ID v1.UID, PodName string, PodSpace string) (*PodStatus, error)
	DeletePod(ID v1.UID) error
	RestartPod(pod *v1.Pod) error
}

type runtimeManager struct {
	lock  sync.Mutex
	IpMap map[v1.UID]string
}

func (rm *runtimeManager) GetAllPods() ([]*Pod, error) {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	containers, err := rm.getAllContainers()
	if err != nil {
		panic(err)
	}
	var ret []*Pod

	for _, container := range containers {
		flag := false
		tempContainer := new(Container)
		tempContainer.ID = container.ID
		tempContainer.Image = container.Image
		tempContainer.Name = container.Labels["Name"]
		rm.checkContainerState(container, tempContainer)

		podBelonged := container.Labels["PodName"]

		for _, podexist := range ret {
			if podexist.Name == podBelonged {
				//fmt.Println("yes " + podBelonged)
				podexist.Containers = append(podexist.Containers, tempContainer)
				flag = true
				break
			}
		}
		if !flag {
			//fmt.Println("no " + podBelonged)
			tempPod := new(Pod)
			tempPod.Name = podBelonged
			tempPod.Namespace = container.Labels["PodNamespace"]
			tempPod.Containers = append(tempPod.Containers, tempContainer)
			tempPod.ID = v1.UID(container.Labels["PodID"])
			ret = append(ret, tempPod)
		}

	}
	return ret, nil
}

func (rm *runtimeManager) GetPodStatus(ID v1.UID, PodName string, PodSpace string) (*PodStatus, error) {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	containers, err := rm.getPodContainers(PodName)
	if err != nil {
		panic(err)
	}
	return &PodStatus{
		ID:                ID,
		IPs:               []string{rm.IpMap[ID]},
		Name:              PodName,
		Namespace:         PodSpace,
		ContainerStatuses: containers,
		TimeStamp:         time.Now(),
	}, nil
}

func (rm *runtimeManager) getPodContainers(PodName string) ([]*ContainerStatus, error) {
	containers, err := rm.getAllContainers()
	if err != nil {
		panic(err)
	}
	var ret []*ContainerStatus
	for _, container := range containers {
		tempContainer := new(ContainerStatus)
		if container.Labels["PodName"] == PodName {
			tempContainer.ID = container.ID
			tempContainer.Image = container.Image
			tempContainer.Name = container.Labels["Name"]
			tempContainer.CreatedAt = time.Unix(container.Created, 0)
			rm.checkContainerStatus(container, tempContainer)

			ret = append(ret, tempContainer)
		}
	}
	return ret, nil
}

func NewRuntimeManager() RuntimeManager {
	manager := &runtimeManager{}
	manager.IpMap = make(map[v1.UID]string)
	return manager
}

func (rm *runtimeManager) AddPod(pod *v1.Pod) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	PauseId, err := rm.CreatePauseContainer(pod.UID, pod.Name, pod.Namespace)
	if err != nil {
		panic(err)
	}

	containerList := pod.Spec.Containers
	for _, container := range containerList {
		volumes, err := rm.createVolumeDir(pod)
		if err != nil {
			return err
		}
		_, err = rm.createContainer(&container, PauseId, pod.UID, pod.Name, pod.Namespace, volumes)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (rm *runtimeManager) CreatePauseContainer(PodID v1.UID, PodName string, PodNameSpace string) (string, error) {
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
		//fmt.Println("yes")
		reader, err := cli.ImagePull(ctx, PauseContainerImage, image.PullOptions{})
		if err != nil {
			panic(err)
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	} else {
		//fmt.Println("already exist")
	}
	label := make(map[string]string)
	label["PodID"] = string(PodID)
	label["PodName"] = PodName
	label["PauseType"] = "pause"
	label["Name"] = PodName + ":PauseContainer"
	label["PodNamespace"] = PodNameSpace
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:  PauseContainerImage,
		Tty:    false,
		Labels: label,
		//ExposedPorts: exposedPortSet,
	}, nil, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	ip, err := nw.Attach(resp.ID)
	if err != nil {
		panic(err)
	}
	rm.IpMap[PodID] = ip

	return resp.ID, nil
}

func (rm *runtimeManager) createContainer(ct *v1.Container, PauseId string, PodID v1.UID, PodName string, PodNameSpace string, volumes map[string]string) (string, error) {
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
		//fmt.Println("yes")
		reader, err := cli.ImagePull(ctx, "docker.io/library/"+repotag, image.PullOptions{})
		if err != nil {
			panic(err)
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	} else {
		//fmt.Println("already exist")
	}

	ports := ct.Ports
	exposedPortSet := nat.PortSet{}
	portKeys := make(map[string]struct{})

	for _, num := range ports {
		//fmt.Println(num.ContainerPort)

		key := fmt.Sprint(num.ContainerPort) + "/tcp"
		//fmt.Println(key)
		portKeys[key] = struct{}{}
	}
	for key := range portKeys {
		exposedPortSet[nat.Port(key)] = struct{}{}
	}

	pauseRef := "container:" + PauseId
	label := make(map[string]string)
	label["PodID"] = string(PodID)
	label["PodName"] = PodName
	label["Name"] = ct.Name
	label["PodNamespace"] = PodNameSpace

	var mounts []mount.Mount
	for _, volume := range ct.VolumeMounts {
		if dir, ok := volumes[volume.Name]; !ok {
			return "", fmt.Errorf("create container: volume %s not declared", volume.Name)
		} else {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: dir,
				Target: volume.MountPath,
			})
		}
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:  repotag,
		Cmd:    cmd,
		Tty:    true,
		Labels: label,
		//ExposedPorts: exposedPortSet,
	}, &container.HostConfig{
		NetworkMode: container.NetworkMode(pauseRef),
		// Binds:        option.Binds,
		// PortBindings: option.PortBindings,
		//IpcMode: container.IpcMode(pauseRef),
		PidMode: container.PidMode(pauseRef),
		Mounts:  mounts,
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

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
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
		//fmt.Println("fail to get images", err)
		panic(err)
	}
	//fmt.Println("Docker Images:")
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

// func (rm *runtimeManager) getAllRunningContainers() ([]types.Container, error) {
// 	ctx := context.Background()
// 	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer cli.Close()

// 	containers, err := cli.ContainerList(ctx, container.ListOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	return containers, nil

// }

func (rm *runtimeManager) getAllContainers() ([]types.Container, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}
	var ret []types.Container
	for _, container := range containers {
		//fmt.Println("container's state" + container.State)
		//fmt.Println("container's status" + container.Status)
		if _, ok := container.Labels["PauseType"]; !ok {
			ret = append(ret, container)
		}
	}

	return ret, nil
}

func (rm *runtimeManager) checkContainerState(ct types.Container, ct_todo *Container) {
	state := ct.State
	//status := ct.Status
	if state == "exited" {
		ct_todo.State = ContainerStateExited
		return
	}
	if state == "running" {
		ct_todo.State = ContainerStateRunning
		return
	}
	if state == "created" {
		ct_todo.State = ContainerStateCreated
		return
	}
	ct_todo.State = ContainerStateUnknown
}

func (rm *runtimeManager) checkContainerStatus(ct types.Container, ct_todo *ContainerStatus) {
	state := ct.State
	status := ct.Status
	if state == "exited" {
		ct_todo.State = ContainerStateExited

		left := strings.Index(status, "(")
		if left == -1 {
			return
		}
		right := strings.Index(status, ")")
		if right == -1 {
			return
		}
		strNumber := status[left+1 : right]
		number, _ := strconv.Atoi(strNumber)

		ct_todo.ExitCode = number
		return
	}
	if state == "running" {
		ct_todo.State = ContainerStateRunning
		return
	}
	if state == "created" {
		ct_todo.State = ContainerStateCreated
		return
	}
	ct_todo.State = ContainerStateUnknown
}

func (rm *runtimeManager) DeletePod(ID v1.UID) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	containers, err := rm.getAllContainersIncludingPause()
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		if container.Labels["PodID"] == string(ID) {
			err := rm.deleteContainer(container)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
}

func (rm *runtimeManager) getAllContainersIncludingPause() ([]types.Container, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		panic(err)
	}
	for _, container := range containers {
		if _, ok := container.Labels["PauseType"]; !ok {
			err := nw.Detach(container.ID)
			if err != nil {
				panic(err)
			}

		}
	}

	return containers, nil
}

func (rm *runtimeManager) deleteContainer(ct types.Container) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	noWaitTimeout := 0
	if ct.State == "running" {
		if err := cli.ContainerStop(ctx, ct.ID, container.StopOptions{Timeout: &noWaitTimeout}); err != nil {
			panic(err)
		}
	}
	if err := cli.ContainerRemove(ctx, ct.ID, container.RemoveOptions{}); err != nil {
		panic(err)
	}

	return nil
}

func (rm *runtimeManager) RestartPod(pod *v1.Pod) error {
	err := rm.DeletePod(pod.UID)
	if err != nil {
		return err
	}
	err = rm.AddPod(pod)
	if err != nil {
		return err
	}
	return nil
}

// volume在主机上的管理由kubelet负责
func (rm *runtimeManager) createVolumeDir(pod *v1.Pod) (map[string]string, error) {
	ret := make(map[string]string)
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir == nil && volume.HostPath == nil {
			return nil, fmt.Errorf("create volume dir: volume %s has no source", volume.Name)
		}
		if volume.HostPath != nil && volume.EmptyDir != nil {
			return nil, fmt.Errorf("create volume dir: volume %s cannot have both hostPath and emptyDir", volume.Name)
		}
		if volume.EmptyDir != nil {
			dir := fmt.Sprintf("/tmp/minikubernetes/volumes/%s/%s", string(pod.UID), volume.Name)
			// MkdirAll会创建所有父目录，若目标目录已存在也不会报错
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("create volume dir: failed to create volume dir %s: %v", dir, err)
			}
			ret[volume.Name] = dir
		} else {
			dir := volume.HostPath.Path
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("create volume dir: failed to create volume dir %s: %v", dir, err)
			}
			ret[volume.Name] = dir
		}
	}
	return ret, nil
}
