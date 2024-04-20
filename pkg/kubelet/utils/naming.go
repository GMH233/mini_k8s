package utils

import v1 "minikubernetes/pkg/api/v1"

func GetPodFullName(pod *v1.Pod) string {
	return pod.ObjectMeta.Name + "_" + pod.ObjectMeta.Namespace
}
