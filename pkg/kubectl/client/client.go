package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubectl/utils"
	"net/http"
)

type Client interface {
	AddPod(jsonBytes []byte) error
	DeletePod(name, namespace string) error
	GetPods() ([]*v1.Pod, error)
}

type kubeletClient struct {
	apiServerIP string
}

func NewKubectlClient(apiServerIP string) Client {
	return &kubeletClient{
		apiServerIP: apiServerIP,
	}
}

func (kc *kubeletClient) AddPod(jsonBytes []byte) error {
	pod, err := utils.JSON2Pod(jsonBytes)
	if err != nil {
		return err
	}
	var namespace string
	if pod.Namespace == "" {
		namespace = "default"
	} else {
		namespace = pod.Namespace
	}
	// POST to API server
	url := fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/pods", kc.apiServerIP, namespace)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("error: %v", resp.Status)
	}
	return nil
}

func (kc *kubeletClient) DeletePod(name, namespace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/pods/%s", kc.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete pod error: %v", resp.Status)
	}
	return nil
}

func (kc *kubeletClient) GetPods() ([]*v1.Pod, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/pods", kc.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.Pod]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pods failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}
