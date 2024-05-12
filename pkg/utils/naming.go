package utils

import v1 "minikubernetes/pkg/api/v1"

func GetObjectFullName(meta *v1.ObjectMeta) string {
	namespace := meta.Namespace
	if namespace == "" {
		namespace = "default"
	}
	return namespace + "_" + meta.Name
}
