package types

import v1 "minikubernetes/pkg/api/v1"


// 一个Endpoint指向一个Pod
type Endpoint struct {
	IP    string
	Ports []EndpointPort
}

type EndpointPort struct {
	Port     int32
	Protocol v1.Protocol
}

type ServiceAndEndpoints struct {
	Service   *v1.Service
	Endpoints []*Endpoint
}
