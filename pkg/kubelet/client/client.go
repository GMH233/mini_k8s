package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	v1 "minikubernetes/pkg/api/v1"
	"net/http"
)

type KubeletClient interface {
	GetPodsByNodeName(nodeId string) ([]*v1.Pod, error)
	UpdatePodStatus(pod *v1.Pod, status *v1.PodStatus) error
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

func (kc *kubeletClient) UpdatePodStatus(pod *v1.Pod, status *v1.PodStatus) error {
	url := fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/pods/%s/status", kc.apiServerIP, pod.Namespace, pod.Name)

	statusJson, err := json.Marshal(status)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(statusJson))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update pod status failed, statusCode: %d", resp.StatusCode)
	}

	return nil
}
