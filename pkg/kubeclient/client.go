package kubeclient

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
	GetAllServices() ([]*v1.Service, error)
	GetAllPods() ([]*v1.Pod, error)
	GetAllDNS() ([]*v1.DNS, error)
	AddPod(pod v1.Pod) error
	DeletePod(name, namespace string) error
	GetAllReplicaSets() ([]*v1.ReplicaSet, error)
	GetAllUnscheduledPods() ([]*v1.Pod, error)
	GetAllNodes() ([]*v1.Node, error)
	AddPodToNode(pod v1.Pod, node v1.Node) error

	// GetReplicaSet(name, namespace string) (*v1.ReplicaSet, error)
	UpdateReplicaSet(name, namespace string, repNum int) error
	GetAllHPAScalers() ([]*v1.HorizontalPodAutoscaler, error)
	UploadPodMetrics(metrics []*v1.PodRawMetrics) error
	GetPodMetrics(v1.MetricsQuery) ([]*v1.PodRawMetrics, error)
}

type client struct {
	apiServerIP string
}

func NewClient(apiServerIP string) Client {
	return &client{
		apiServerIP: apiServerIP,
	}
}

func (c *client) GetAllServices() ([]*v1.Service, error) {
	url := fmt.Sprintf("http://%s:8001/api/v1/services", c.apiServerIP)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.Service]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get services failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetAllPods() ([]*v1.Pod, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/pods", c.apiServerIP))
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

func (c *client) GetAllDNS() ([]*v1.DNS, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/dns", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.DNS]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get dns failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetAllReplicaSets() ([]*v1.ReplicaSet, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/replicasets", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.ReplicaSet]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get replica sets failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddPod(pod v1.Pod) error {

	var namespace string
	if pod.Namespace == "" {
		namespace = "default"
	} else {
		namespace = pod.Namespace
	}
	jsonBytes, err := utils.Pod2JSON(&pod)
	if err != nil {
		return err
	}
	// POST to API server
	url := fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/pods", c.apiServerIP, namespace)
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
func (c *client) DeletePod(name, namespace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/pods/%s", c.apiServerIP, namespace, name), nil)
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

func (c *client) GetAllUnscheduledPods() ([]*v1.Pod, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/pods/unscheduled", c.apiServerIP))
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
		return nil, fmt.Errorf("get unscheduled pods failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetAllNodes() ([]*v1.Node, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/nodes", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.Node]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get nodes failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddPodToNode(pod v1.Pod, node v1.Node) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/schedule?podUid=%s&nodename=%s", c.apiServerIP, pod.UID, node.Name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("add pod to node error: %v", resp.Status)
	}
	return nil
}

// 可能没用到
// func (c *client) GetReplicaSet(name, namespace string) (*v1.ReplicaSet, error) {
// 	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/replicasets/%s", c.apiServerIP, namespace, name))
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var baseResponse v1.BaseResponse[v1.ReplicaSet]
// 	err = json.Unmarshal(body, &baseResponse)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		return nil, fmt.Errorf("get replica set error: %v", baseResponse.Error)
// 	}
// 	return &baseResponse.Data, nil
// }

func (c *client) UpdateReplicaSet(name, namespace string, repNum int) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/replicasets/%s", c.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update replica set error: %v", resp.Status)
	}
	return nil
}

func (c *client) GetAllHPAScalers() ([]*v1.HorizontalPodAutoscaler, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/scaling", c.apiServerIP))

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.HorizontalPodAutoscaler]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get hpa scalers failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) UploadPodMetrics(metrics []*v1.PodRawMetrics) error {
	url := fmt.Sprintf("http://%s:8001/api/v1/stats/data", c.apiServerIP)

	metricsStr, _ := json.Marshal(metrics)

	fmt.Printf("upload metrics str: %s\n", string(metricsStr))

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(metricsStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload metrics failed, statusCode: %d", resp.StatusCode)
	}

	return nil
}

func (c *client) GetPodMetrics(metricsQry v1.MetricsQuery) ([]*v1.PodRawMetrics, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s:8001/api/v1/stats/data", c.apiServerIP), nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()

	query.Add("uid", fmt.Sprint(metricsQry.UID))
	query.Add("timestamp", metricsQry.TimeStamp.UTC().Format("2006-01-02T15:04:05.99999999Z"))
	query.Add("window", fmt.Sprint(metricsQry.Window))
	req.URL.RawQuery = query.Encode()

	resp, err := http.DefaultClient.Do(req)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var baseResponse v1.BaseResponse[[]*v1.PodRawMetrics]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pod metrics failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}
