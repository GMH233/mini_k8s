package types

import "minikubernetes/pkg/api/v1"

type PodOperation string

const (
	ADD    PodOperation = "ADD"
	DELETE PodOperation = "DELETE"
	UPDATE PodOperation = "UPDATE"
)

type PodUpdate struct {
	Pods []*v1.Pod
	Op   PodOperation
}
