package types

import v1 "minikubernetes/pkg/api/v1"

type ServiceOperation string

const (
	ServiceAdd    ServiceOperation = "ServiceAdd"
	ServiceDelete ServiceOperation = "ServiceDelete"
	// delete和add放在同一个事件中，防止冲突
	ServiceEndpointsUpdate ServiceOperation = "ServiceEndpointsUpdate"
)

type ServiceUpdate struct {
	//Service           *v1.Service
	//EndpointAdditions []*Endpoint
	//EndpointDeletions []*Endpoint
	Updates []*ServiceUpdateSingle
	Op      ServiceOperation
}

type ServiceUpdateSingle struct {
	Service           *v1.Service
	EndpointAdditions []*Endpoint
	EndpointDeletions []*Endpoint
}
