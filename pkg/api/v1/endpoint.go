package v1

// 一个Endpoint指向一个Pod
type Endpoint struct {
	//PodName string
	IP    string
	Ports []EndpointPort
}

type EndpointPort struct {
	Port     int32
	Protocol Protocol
}

type ServiceAndEndpoints struct {
	Service                 *Service
	EndpointsMapWithPodName map[string]Endpoint
}
