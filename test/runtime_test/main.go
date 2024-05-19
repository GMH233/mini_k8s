package main

import (
	v1 "minikubernetes/pkg/api/v1"
	rt "minikubernetes/pkg/kubelet/runtime"
)

func main() {
	conport := &v1.ContainerPort{ContainerPort: 8080}
	conport2 := &v1.ContainerPort{ContainerPort: 8090}
	contain := &v1.Container{
		Image:   "alpine:latest",
		Command: []string{},
		Ports:   []v1.ContainerPort{*conport},
	}
	contain2 := &v1.Container{
		Image:   "alpine:latest",
		Command: []string{},
		Ports:   []v1.ContainerPort{*conport2},
	}
	podSpec := &v1.PodSpec{
		Containers: []v1.Container{*contain, *contain2},
	}
	pod := &v1.Pod{
		Spec: *podSpec,
	}
	rm := rt.NewRuntimeManager("1.1.1.1")
	rt.RuntimeManager.AddPod(rm, pod)
}
