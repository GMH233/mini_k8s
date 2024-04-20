package client

import (
	"encoding/json"
	"fmt"
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

type BaseResponse[T any] struct {
	Data  T      `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func (kc *kubeletClient) GetPodsByNodeName(nodeName string) ([]*v1.Pod, error) {
	url := fmt.Sprintf("http://%s:8001/api/v1/nodes/%s/pods", kc.apiServerIP, nodeName)
	resp, err := http.Get(url)
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
	var baseResponse BaseResponse[[]*v1.Pod]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pods failed, error: %s", baseResponse.Error)
	}

	return baseResponse.Data, nil
}
