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
	RegisterNode(address string) (*v1.Node, error)
	UnregisterNode(nodeName string) error
	GetPVByPVCName(pvcName, pvcNamespace string) (*v1.PersistentVolume, error)
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

func (c *kubeletClient) RegisterNode(address string) (*v1.Node, error) {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:8001/api/v1/nodes/register", c.apiServerIP), nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	query.Add("address", address)
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.Node]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("register node failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *kubeletClient) UnregisterNode(nodeName string) error {
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s:8001/api/v1/nodes/unregister", c.apiServerIP), nil)
	if err != nil {
		return err
	}
	query := req.URL.Query()
	query.Add("nodename", nodeName)
	req.URL.RawQuery = query.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[interface{}]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unregister node failed, error: %s", baseResponse.Error)
	}
	return nil
}

func (c *kubeletClient) GetPVByPVCName(pvcName, pvcNamespace string) (*v1.PersistentVolume, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/persistentvolumeclaims/%s", c.apiServerIP, pvcNamespace, pvcName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.PersistentVolumeClaim]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pv by pvc name failed, error: %s", baseResponse.Error)
	}
	pvc := baseResponse.Data
	if pvc.Status.VolumeName == "" {
		return nil, fmt.Errorf("pv not found")
	}
	resp, err = http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/persistentvolumes/%s", c.apiServerIP, pvcNamespace, pvc.Status.VolumeName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var pvBaseResponse v1.BaseResponse[*v1.PersistentVolume]
	err = json.Unmarshal(body, &pvBaseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pv by pvc name failed, error: %s", pvBaseResponse.Error)
	}
	return pvBaseResponse.Data, nil
}
