package main

import (
	"github.com/google/uuid"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubelet/runtime"
)

func main() {
	pod1 := &v1.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "pod1",
			Namespace: "default",
			UID:       v1.UID(uuid.New().String()),
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "volume1",
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:  "container1",
					Image: "alpine:latest",
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "volume1",
							MountPath: "/tmp",
						},
					},
				},
			},
		},
	}
	pod2 := &v1.Pod{
		TypeMeta: v1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "pod2",
			Namespace: "default",
			UID:       v1.UID(uuid.New().String()),
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "volume2",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/root/zc/mini_k8s",
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:  "container1",
					Image: "alpine:latest",
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "volume2",
							MountPath: "/tmp",
						},
					},
				},
			},
		},
	}
	runtimeManager := runtime.NewRuntimeManager()
	err := runtimeManager.AddPod(pod1)
	if err != nil {
		panic(err)
	}
	err = runtimeManager.AddPod(pod2)
	if err != nil {
		panic(err)
	}
}
