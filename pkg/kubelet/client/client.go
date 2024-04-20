package client

import (
	"encoding/json"
	"io"
	v1 "minikubernetes/pkg/api/v1"
	"net/http"
)

type KubeletClient interface {
	GetPodsByNodeName(nodeId string) ([]*v1.Pod, error)
}

type kubeletClient struct {
	apiServerIP string
}

func NewKubeletClient(apiServerIP string) KubeletClient {
	return &kubeletClient{
		apiServerIP: apiServerIP,
	}
}

func (kc *kubeletClient) GetPodsByNodeName(nodeName string) ([]*v1.Pod, error) {
	resp, err := http.Get(kc.apiServerIP + "/pods")
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var respPods []*v1.Pod
	err = json.Unmarshal(body, &respPods)
	if err != nil {
		return nil, err
	}
	return respPods, nil
}
