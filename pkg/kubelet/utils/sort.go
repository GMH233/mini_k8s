package utils

import (
	v1 "minikubernetes/pkg/api/v1"
	"sort"
)

func SortPodsByCreationTime(pods []*v1.Pod) {
	sort.Slice(pods, func(i, j int) bool {
		return pods[i].CreationTimestamp.Before(pods[j].CreationTimestamp)
	})
}
