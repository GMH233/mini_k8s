package types

import v1 "minikubernetes/pkg/api/v1"

type DNSOperation string

const (
	DNSAdd    DNSOperation = "DNSAdd"
	DNSDelete DNSOperation = "DNSDelete"
)

type DNSUpdate struct {
	DNS []*v1.DNS
	Op  DNSOperation
}
