package init

import (
	"github.com/coreos/go-iptables/iptables"
	"minikubernetes/pkg/microservice/envoy"
)

type EnvoyInit struct {
	ipt *iptables.IPTables
}

func NewEnvoyInit() (*EnvoyInit, error) {
	envoy := &EnvoyInit{}
	ipt, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		return nil, err
	}
	envoy.ipt = ipt
	return envoy, nil
}

// 在当前容器的网络命名空间中配置iptables规则，使得所有出入站流量都被重定向到Envoy代理
func (e *EnvoyInit) Init() error {
	// 生成MISTIO_REDIRECT链
	err := e.ipt.NewChain("nat", "MISTIO_REDIRECT")
	if err != nil {
		return err
	}
	// 在MISTIO_REDIRECT链中追加一条REDIRECT规则，重定向到Envoy的Outbound端口
	err = e.ipt.Append("nat", "MISTIO_REDIRECT", "-p", "tcp", "-j", "REDIRECT", "--to-port", envoy.OutboundPort)
	if err != nil {
		return err
	}
	// 生成MISTIO_IN_REDIRECT链
	err = e.ipt.NewChain("nat", "MISTIO_IN_REDIRECT")
	if err != nil {
		return err
	}
	// 在MISTIO_IN_REDIRECT链中追加一条REDIRECT规则，重定向到Envoy的Inbound端口
	err = e.ipt.Append("nat", "MISTIO_IN_REDIRECT", "-p", "tcp", "-j", "REDIRECT", "--to-port", envoy.InboundPort)
	if err != nil {
		return err
	}
	// 生成MISTIO_INBOUND链
	err = e.ipt.NewChain("nat", "MISTIO_INBOUND")
	if err != nil {
		return err
	}
	// 在PREROUTING链中追加一条规则，将所有入站流量重定向到MISTIO_INBOUND链
	err = e.ipt.Append("nat", "PREROUTING", "-p", "tcp", "-j", "MISTIO_INBOUND")
	if err != nil {
		return err
	}
	// 没做：ssh端口22
	// 在MISTIO_INBOUND链中追加一条规则，将所有入站流量重定向到MISTIO_IN_REDIRECT链
	err = e.ipt.Append("nat", "MISTIO_INBOUND", "-p", "tcp", "-j", "MISTIO_IN_REDIRECT")
	if err != nil {
		return err
	}
	// 生成MISTIO_OUTPUT链
	err = e.ipt.NewChain("nat", "MISTIO_OUTPUT")
	if err != nil {
		return err
	}
	// 在OUTPUT链中追加一条规则，将所有出站流量重定向到MISTIO_OUTPUT链
	err = e.ipt.Append("nat", "OUTPUT", "-p", "tcp", "-j", "MISTIO_OUTPUT")
	if err != nil {
		return err
	}

	// 没做：对源ip为127.0.0.6的处理

	// Redirect app calls back to itself via Envoy when using the service VIP
	// e.g. appN => Envoy (client) => Envoy (server) => appN.
	err = e.ipt.Append("nat", "MISTIO_OUTPUT",
		"-o", "lo",
		"!", "-d", "127.0.0.1/32",
		"-m", "owner", "--uid-owner", envoy.UID,
		"-j", "MISTIO_IN_REDIRECT")
	if err != nil {
		return err
	}
	// Do not redirect app calls to back itself via Envoy when using the endpoint address
	// e.g. appN => appN by lo
	err = e.ipt.Append("nat", "MISTIO_OUTPUT",
		"-o", "lo",
		"-m", "owner", "!", "--uid-owner", envoy.UID,
		"-j", "RETURN")
	if err != nil {
		return err
	}
	// Avoid infinite loops. Don't redirect Envoy traffic directly back to
	// Envoy for non-loopback traffic.
	err = e.ipt.Append("nat", "MISTIO_OUTPUT",
		"-m", "owner", "--uid-owner", envoy.UID,
		"-j", "RETURN")
	if err != nil {
		return err
	}

	// 对于gid进行相同操作
	err = e.ipt.Append("nat", "MISTIO_OUTPUT",
		"-o", "lo",
		"!", "-d", "127.0.0.1/32",
		"-m", "owner", "--gid-owner", envoy.GID,
		"-j", "MISTIO_IN_REDIRECT")
	if err != nil {
		return err
	}
	err = e.ipt.Append("nat", "MISTIO_OUTPUT",
		"-o", "lo",
		"-m", "owner", "!", "--gid-owner", envoy.GID,
		"-j", "RETURN")
	if err != nil {
		return err
	}
	err = e.ipt.Append("nat", "MISTIO_OUTPUT",
		"-m", "owner", "--gid-owner", envoy.GID,
		"-j", "RETURN")
	if err != nil {
		return err
	}

	// 对于目标地址为127.0.0.1的数据包，不进行任何处理并返回上一级链。
	err = e.ipt.Append("nat", "MISTIO_OUTPUT", "-d", "127.0.0.1/32", "-j", "RETURN")
	if err != nil {
		return err
	}
	// 剩余的数据包重定向到MISTIO_REDIRECT链
	err = e.ipt.Append("nat", "MISTIO_OUTPUT", "-j", "MISTIO_REDIRECT")
	return nil
}
