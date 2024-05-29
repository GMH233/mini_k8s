package runtime

import (
	"context"
	"fmt"
	"io"
	"minikubernetes/pkg/microservice/envoy"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/mount"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	v1 "minikubernetes/pkg/api/v1"
	nw "minikubernetes/pkg/kubelet/network"

	kubeletclient "minikubernetes/pkg/kubelet/client"
)

type RuntimeManager interface {
	AddPod(pod *v1.Pod) error
	GetAllPods() ([]*Pod, error)
	GetPodStatus(ID v1.UID, PodName string, PodSpace string) (*PodStatus, error)
	DeletePod(ID v1.UID) error
	RestartPod(pod *v1.Pod) error
	CleanVolumes(pod *v1.Pod) error
}

type runtimeManager struct {
	lock         sync.Mutex
	IpMap        map[v1.UID]string
	nameserverIP string
	client       kubeletclient.KubeletClient
}

func (rm *runtimeManager) GetAllPods() ([]*Pod, error) {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	containers, err := rm.getAllContainers()
	if err != nil {
		// panic(err)
		return nil, err
	}
	var ret []*Pod

	for _, container := range containers {
		flag := false
		tempContainer := new(Container)
		tempContainer.ID = container.ID
		tempContainer.Image = container.Image
		tempContainer.Name = container.Labels["Name"]
		rm.checkContainerState(container, tempContainer)

		podBelonged, ok := container.Labels["PodName"]
		if !ok {
			continue
		}

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
		//panic(err)
		return nil, err
	}
	podStatus := &PodStatus{
		ID:                ID,
		IPs:               nil,
		Name:              PodName,
		Namespace:         PodSpace,
		ContainerStatuses: containers,
		TimeStamp:         time.Now(),
	}
	ip, ok := rm.IpMap[ID]
	if ok {
		podStatus.IPs = append(podStatus.IPs, ip)
	}
	return podStatus, nil
}

func (rm *runtimeManager) getPodContainers(PodName string) ([]*ContainerStatus, error) {
	containers, err := rm.getAllContainers()
	if err != nil {
		//panic(err)
		return nil, err
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

func NewRuntimeManager(nameserverIP string, client kubeletclient.KubeletClient) RuntimeManager {
	manager := &runtimeManager{}
	manager.IpMap = make(map[v1.UID]string)
	manager.nameserverIP = nameserverIP
	manager.client = client
	return manager
}

func (rm *runtimeManager) addPodLocked(pod *v1.Pod) error {
	//PauseId, err := rm.CreatePauseContainer(pod.UID, pod.Name, pod.Namespace)
	PauseId, err := rm.CreatePauseContainer(pod)
	if err != nil {
		//panic(err)
		return err
	}

	for _, c := range pod.Spec.InitContainers {
		err = rm.createInitContainer(&c, PauseId)
		if err != nil {
			return err
		}
	}

	containerList := pod.Spec.Containers
	for _, container := range containerList {
		volumes, err := rm.createVolumeDir(pod)
		if err != nil {
			return err
		}
		_, err = rm.createContainer(&container, PauseId, pod.UID, pod.Name, pod.Namespace, volumes)
		if err != nil {
			//panic(err)
			return err
		}
	}

	return nil
}

func (rm *runtimeManager) AddPod(pod *v1.Pod) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	return rm.addPodLocked(pod)
}

// func (rm *runtimeManager) CreatePauseContainer(PodID v1.UID, PodName string, PodNameSpace string) (string, error) {
func (rm *runtimeManager) CreatePauseContainer(pod *v1.Pod) (string, error) {
	PodID := pod.UID
	PodName := pod.Name
	PodNameSpace := pod.Namespace
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		//panic(err)
		return "", err
	}
	defer cli.Close()
	PauseContainerImage := "registry.aliyuncs.com/google_containers/pause:3.6"
	exi, err := rm.checkImages(PauseContainerImage)
	if err != nil {
		//panic(err)
		return "", err
	}
	if !exi {
		//fmt.Println("yes")
		reader, err := cli.ImagePull(ctx, PauseContainerImage, image.PullOptions{})
		if err != nil {
			//panic(err)
			return "", err
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
		Image:        PauseContainerImage,
		Tty:          false,
		Labels:       label,
		ExposedPorts: rm.getExposedPorts(pod.Spec.Containers),
	}, &container.HostConfig{
		PortBindings: make(nat.PortMap),
		DNS:          []string{rm.nameserverIP},
	}, nil, nil, "")
	if err != nil {
		//panic(err)
		return "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		//panic(err)
		return "", err
	}

	ip, err := nw.Attach(resp.ID)
	if err != nil {
		//panic(err)
		return "", err
	}
	rm.IpMap[PodID] = ip

	return resp.ID, nil
}

func (rm *runtimeManager) getExposedPorts(containers []v1.Container) nat.PortSet {
	exposedPorts := nat.PortSet{}
	for _, c := range containers {
		ports := c.Ports
		portKeys := make(map[string]struct{})
		for _, port := range ports {
			var p string
			switch port.Protocol {
			case v1.ProtocolTCP:
				p = "/tcp"
			case v1.ProtocolUDP:
				p = "/udp"
			default:
				p = "/tcp"
			}
			key := fmt.Sprint(port.ContainerPort) + p
			portKeys[key] = struct{}{}
		}
		for key := range portKeys {
			exposedPorts[nat.Port(key)] = struct{}{}
		}
	}
	return exposedPorts
}

func (rm *runtimeManager) getPortBindings(containers []v1.Container) nat.PortMap {
	portBindings := make(nat.PortMap)
	for _, c := range containers {
		ports := c.Ports
		for _, num := range ports {
			key := nat.Port(fmt.Sprint(num.ContainerPort) + "/tcp")
			portBindings[key] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "",
				},
			}
		}
	}
	return portBindings
}

// 目前init container只是为了支持sidecar，简化了很多逻辑
func (rm *runtimeManager) createInitContainer(c *v1.Container, pauseID string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()
	exist, err := rm.checkImages(c.Image)
	if err != nil {
		return err
	}
	if !exist {
		readCloser, err := cli.ImagePull(context.Background(), c.Image, image.PullOptions{})
		if err != nil {
			return err
		}
		// 读取pull的输出
		_, _ = io.ReadAll(readCloser)
		_ = readCloser.Close()
	}
	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode("container:" + pauseID),
	}
	if c.SecurityContext != nil {
		if c.SecurityContext.Privileged != nil && *c.SecurityContext.Privileged {
			hostConfig.Privileged = *c.SecurityContext.Privileged
		}
	}
	// run container
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: c.Image,
		Tty:   true,
	}, hostConfig, nil, nil, "")
	if err != nil {
		return err
	}
	if err = cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	// wait for init container to finish
	statusCh, errCh := cli.ContainerWait(context.Background(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			return err
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("init container exited with status %d", status.StatusCode)
		}
	}
	// delete init container
	if err = cli.ContainerRemove(context.Background(), resp.ID, container.RemoveOptions{}); err != nil {
		return err
	}
	return nil
}

func (rm *runtimeManager) createContainer(ct *v1.Container, PauseId string, PodID v1.UID, PodName string, PodNameSpace string, volumes map[string]string) (string, error) {
	repotag := ct.Image
	cmd := ct.Command
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		//panic(err)
		return "", err
	}
	defer cli.Close()

	exi, err := rm.checkImages(repotag)
	if err != nil {
		//panic(err)
		return "", err
	}
	if !exi {
		//fmt.Println("yes")
		reader, err := cli.ImagePull(ctx, repotag, image.PullOptions{})
		if err != nil {
			//panic(err)
			return "", err
		}
		defer reader.Close()
		io.Copy(os.Stdout, reader)
	} else {
		//fmt.Println("already exist")
	}
	//
	//ports := ct.Ports
	//exposedPortSet := nat.PortSet{}
	//portKeys := make(map[string]struct{})
	//
	//for _, num := range ports {
	//	//fmt.Println(num.ContainerPort)
	//
	//	key := fmt.Sprint(num.ContainerPort) + "/tcp"
	//	//fmt.Println(key)
	//	portKeys[key] = struct{}{}
	//}
	//for key := range portKeys {
	//	exposedPortSet[nat.Port(key)] = struct{}{}
	//}

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

	config := &container.Config{
		Image:  repotag,
		Cmd:    cmd,
		Tty:    true,
		Labels: label,
	}

	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode(pauseRef),
		PidMode:     container.PidMode(pauseRef),
		Mounts:      mounts,
	}

	if ct.SecurityContext != nil {
		if ct.SecurityContext.Privileged != nil && *ct.SecurityContext.Privileged {
			hostConfig.Privileged = *ct.SecurityContext.Privileged
		} else if ct.SecurityContext.RunAsUser != nil {
			config.User = strconv.FormatInt(*ct.SecurityContext.RunAsUser, 10)
		}
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		//panic(err)
		return "", err
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		//panic(err)
		return "", err
	}

	// statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	// select {
	// case err := <-errCh:
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// case <-statusCh:
	// }

	//out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	//if err != nil {
	//	//panic(err)
	//	return "", err
	//}
	//
	//stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return resp.ID, nil
}

func (rm *runtimeManager) checkImages(repotag string) (bool, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		//panic(err)
		return false, err
	}
	defer cli.Close()
	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		//fmt.Println("fail to get images", err)
		//panic(err)
		return false, err
	}
	//fmt.Println("Docker Images:")
	for _, image := range images {
		//fmt.Printf("ID: %s\n", image.ID)
		//fmt.Printf("RepoTags: %v\n", image.RepoTags)
		//if len(image.RepoTags) == 0 {
		//	continue
		//}
		//if image.RepoTags[0] == repotag {
		//	return true, nil
		//}
		for _, rt := range image.RepoTags {
			if rt == repotag {
				return true, nil
			}
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
		//panic(err)
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		//panic(err)
		return nil, err
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

func (rm *runtimeManager) deletePodLocked(ID v1.UID) error {
	containers, err := rm.getAllContainersIncludingPause()
	if err != nil {
		return err
	}
	for _, container := range containers {
		if container.Labels["PodID"] == string(ID) {
			if _, ok := container.Labels["PauseType"]; ok {
				// log.Printf("Container ID: %s\n", container.ID)
				err = nw.Detach(container.ID)
				if err != nil {
					return err
				}
				delete(rm.IpMap, v1.UID(container.Labels["PodID"]))
			}
			err = rm.deleteContainer(container)
			if err != nil {
				return err
			}
		}
		//if _, ok := container.Labels["IsEnvoy"]; ok {
		//	if container.Labels["EnvoyPodID"] == string(ID) {
		//		err = rm.deleteContainer(container)
		//		if err != nil {
		//			return err
		//		}
		//	}
		//}
	}
	return nil
}

func (rm *runtimeManager) DeletePod(ID v1.UID) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	return rm.deletePodLocked(ID)
}

func (rm *runtimeManager) getAllContainersIncludingPause() ([]types.Container, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}
	//for _, container := range containers {
	//	if _, ok := container.Labels["PauseType"]; ok {
	//		log.Printf("Container ID: %s\n", container.ID)
	//		err := nw.Detach(container.ID)
	//		if err != nil {
	//			panic(err)
	//		}
	//		delete(rm.IpMap, v1.UID(container.Labels["PodID"]))
	//	}
	//}

	return containers, nil
}

func (rm *runtimeManager) deleteContainer(ct types.Container) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		//panic(err)
		return err
	}
	defer cli.Close()
	noWaitTimeout := 0
	if ct.State == "running" {
		if err := cli.ContainerStop(ctx, ct.ID, container.StopOptions{Timeout: &noWaitTimeout}); err != nil {
			//panic(err)
			return err
		}
	}
	if err := cli.ContainerRemove(ctx, ct.ID, container.RemoveOptions{}); err != nil {
		//panic(err)
		return err
	}

	return nil
}

func (rm *runtimeManager) RestartPod(pod *v1.Pod) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	//err := rm.deletePodLocked(pod.UID)
	//if err != nil {
	//	return err
	//}
	//err = rm.addPodLocked(pod)
	//if err != nil {
	//	return err
	//}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()
	containers, err := rm.getAllContainers()
	if err != nil {
		return err
	}
	noWaitTimeout := 0
	for _, ct := range containers {
		if ct.Labels["PodID"] == string(pod.UID) {
			err = cli.ContainerRestart(context.Background(), ct.ID, container.StopOptions{Timeout: &noWaitTimeout})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// volume在主机上的管理由kubelet负责
func (rm *runtimeManager) createVolumeDir(pod *v1.Pod) (map[string]string, error) {
	ret := make(map[string]string)
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir == nil && volume.HostPath == nil && volume.PersistentVolumeClaim == nil {
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
		} else if volume.HostPath != nil {
			dir := volume.HostPath.Path
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("create volume dir: failed to create volume dir %s: %v", dir, err)
			}
			ret[volume.Name] = dir
		} else {
			pv, err := rm.client.GetPVByPVCName(volume.PersistentVolumeClaim.ClaimName, pod.Namespace)
			if err != nil {
				return nil, fmt.Errorf("create volume dir: failed to get pv by pvc name %s: %v", volume.PersistentVolumeClaim.ClaimName, err)
			}
			dir := fmt.Sprintf("/tmp/minikubernetes/volumes/%s/%s", string(pod.UID), volume.Name)
			err = os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("create volume dir: failed to create volume dir %s: %v", dir, err)
			}
			// mount -t nfs -o vers=3
			output, err := exec.Command("mount", "-t", "nfs", "-o", "vers=3", pv.Status.Server+":"+pv.Status.Path, dir).CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("create volume dir: failed to mount nfs: %v, output: %s", err, output)
			}
			ret[volume.Name] = dir
		}
	}
	return ret, nil
}

func (rm *runtimeManager) injectSideCar(pauseID string, tag string, pod *v1.Pod) error {
	if tag == "" {
		tag = "latest"
	}
	envoyInitImage := "sjtuzc/envoy-init:" + tag
	envoyImage := "sjtuzc/envoy:" + tag
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	defer cli.Close()

	exist, err := rm.checkImages(envoyInitImage)
	if err != nil {
		return err
	}
	if !exist {
		_, err = cli.ImagePull(context.Background(), envoyInitImage, image.PullOptions{})
		if err != nil {
			return err
		}
	}
	exist, err = rm.checkImages(envoyImage)
	if err != nil {
		return err
	}
	if !exist {
		_, err = cli.ImagePull(context.Background(), envoyImage, image.PullOptions{})
		if err != nil {
			return err
		}
	}

	// docker run --rm --net container:pauseID --privileged envoy-init
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: envoyInitImage,
		Tty:   true,
	}, &container.HostConfig{
		NetworkMode: container.NetworkMode("container:" + pauseID),
		Privileged:  true,
	}, nil, nil, "")
	if err != nil {
		return err
	}
	if err = cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	// wait for envoy-init to finish
	statusCh, errCh := cli.ContainerWait(context.Background(), resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			return err
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("envoy-init exited with status %d", status.StatusCode)
		}
	}

	// docker run --net container:pauseID -u 1337 envoy
	labels := make(map[string]string)
	labels["IsEnvoy"] = "true"
	labels["EnvoyPodID"] = string(pod.UID)
	resp, err = cli.ContainerCreate(context.Background(), &container.Config{
		Image:  envoyImage,
		Tty:    true,
		User:   envoy.UID,
		Labels: labels,
	}, &container.HostConfig{
		NetworkMode: container.NetworkMode("container:" + pauseID),
	}, nil, nil, "")
	if err != nil {
		return err
	}

	if err = cli.ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	return nil
}

func (rm *runtimeManager) CleanVolumes(pod *v1.Pod) error {
	rm.lock.Lock()
	defer rm.lock.Unlock()
	for _, volume := range pod.Spec.Volumes {
		if volume.HostPath != nil {
			continue
		}
		dir := fmt.Sprintf("/tmp/minikubernetes/volumes/%s/%s", string(pod.UID), volume.Name)
		if volume.PersistentVolumeClaim != nil {
			// umount dir
			output, err := exec.Command("umount", dir).CombinedOutput()
			if err != nil {
				return fmt.Errorf("clean volumes: failed to umount nfs: %v, output: %s", err, output)
			}
		}
		err := os.RemoveAll(dir)
		if err != nil {
			return fmt.Errorf("clean volumes: failed to remove volume dir %s: %v", dir, err)
		}
	}
	dir := fmt.Sprintf("/tmp/minikubernetes/volumes/%s", string(pod.UID))
	_ = os.RemoveAll(dir)
	return nil
}
