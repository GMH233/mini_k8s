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
	GetAllPods() ([]*v1.Pod, error)
	GetPod(name, namespace string) (*v1.Pod, error)
	AddPod(pod v1.Pod) error
	DeletePod(name, namespace string) error

	GetAllUnscheduledPods() ([]*v1.Pod, error)

	GetAllDNS() ([]*v1.DNS, error)
	GetDNS(name, namespace string) (*v1.DNS, error)
	AddDNS(dns v1.DNS) error
	DeleteDNS(name, namespace string) error

	GetAllNodes() ([]*v1.Node, error)
	AddPodToNode(pod v1.Pod, node v1.Node) error

	GetAllServices() ([]*v1.Service, error)
	GetService(name, namespace string) (*v1.Service, error)
	AddService(service v1.Service) error
	DeleteService(name, namespace string) error

	GetAllReplicaSets() ([]*v1.ReplicaSet, error)
	GetReplicaSet(name, namespace string) (*v1.ReplicaSet, error)
	AddReplicaSet(replicaSet v1.ReplicaSet) error
	DeleteReplicaSet(name, namespace string) error
	UpdateReplicaSet(name, namespace string, repNum int32) error

	GetAllHPAScalers() ([]*v1.HorizontalPodAutoscaler, error)
	GetHPAScaler(name, namespace string) (*v1.HorizontalPodAutoscaler, error)
	AddHPAScaler(hpa v1.HorizontalPodAutoscaler) error
	DeleteHPAScaler(name, namespace string) error

	UploadPodMetrics(metrics []*v1.PodRawMetrics) error
	GetPodMetrics(v1.MetricsQuery) (*v1.PodRawMetrics, error)

	GetSidecarMapping() (v1.SidecarMapping, error)
	GetAllVirtualServices() ([]*v1.VirtualService, error)
	GetSubsetByName(name, namespace string) (*v1.Subset, error)

	AddSidecarMapping(maps v1.SidecarMapping) error

	GetSidecarServiceNameMapping() (v1.SidecarServiceNameMapping, error)

	GetAllRollingUpdates() ([]*v1.RollingUpdate, error)
	UpdateRollingUpdateStatus(name, namespace string, status *v1.RollingUpdateStatus) error

	GetAllSubsets() ([]*v1.Subset, error)
	AddSubset(subset *v1.Subset) error
	DeleteSubset(subset *v1.Subset) error
	DeleteSubsetByNameNp(subsetName, nameSpace string) error

	GetVirtualService(name, namespace string) (*v1.VirtualService, error)
	AddVirtualService(virtualService *v1.VirtualService) error
	DeleteVirtualService(virtualService *v1.VirtualService) error
	DeleteVirtualServiceByNameNp(vsName, nameSpace string) error
}

type client struct {
	apiServerIP string
}

func NewClient(apiServerIP string) Client {
	return &client{
		apiServerIP: apiServerIP,
	}
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

func (c *client) GetPod(name, namespace string) (*v1.Pod, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/pods/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.Pod]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pod error: %v", baseResponse.Error)
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

func (c *client) GetDNS(name, namespace string) (*v1.DNS, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/dns/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.DNS]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get dns error: %v", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddDNS(dns v1.DNS) error {
	if dns.Namespace == "" {
		dns.Namespace = "default"
	}

	dnsJson, _ := json.Marshal(dns)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/dns", c.apiServerIP, dns.Namespace), bytes.NewBuffer(dnsJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.DNS]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add dns error: %v", baseResponse.Error)
	}
	return nil
}

func (c *client) DeleteDNS(name, namespace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/dns/%s", c.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.DNS]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete dns error: %v", baseResponse.Error)
	}
	return nil
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

func (c *client) GetService(name, namespace string) (*v1.Service, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/services/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.Service]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get service error: %v", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddService(service v1.Service) error {
	if service.Namespace == "" {
		service.Namespace = "default"
	}

	serviceJson, err := json.Marshal(service)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/services", c.apiServerIP, service.Namespace), bytes.NewBuffer(serviceJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.Service]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add service error: %v", baseResponse.Error)
	}
	return nil
}
func (c *client) DeleteService(name, namespace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/services/%s", c.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.Service]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete service error: %v", baseResponse.Error)
	}
	return nil
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

func (c *client) GetReplicaSet(name, namespace string) (*v1.ReplicaSet, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/replicasets/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var baseResponse v1.BaseResponse[v1.ReplicaSet]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get replica set error: %v", baseResponse.Error)
	}
	return &baseResponse.Data, nil
}

func (c *client) AddReplicaSet(replicaSet v1.ReplicaSet) error {
	if replicaSet.Namespace == "" {
		replicaSet.Namespace = "default"
	}

	replicaSetJson, _ := json.Marshal(replicaSet)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/replicasets", c.apiServerIP, replicaSet.Namespace), bytes.NewBuffer(replicaSetJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.Service]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add replica set error: %v", baseResponse.Error)
	}
	return nil
}

func (c *client) DeleteReplicaSet(name, namespace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/replicasets/%s", c.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.ReplicaSet]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete replica set error: %v", baseResponse.Error)
	}
	return nil
}

func (c *client) UpdateReplicaSet(name, namespace string, repNum int32) error {
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/replicasets/%s", c.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}

	query := req.URL.Query()
	query.Add("replicas", fmt.Sprint(repNum))
	req.URL.RawQuery = query.Encode()

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
func (c *client) GetHPAScaler(name, namespace string) (*v1.HorizontalPodAutoscaler, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/scaling/scalingname/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.HorizontalPodAutoscaler]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get hpa scaler error: %v", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddHPAScaler(hpa v1.HorizontalPodAutoscaler) error {
	if hpa.Namespace == "" {
		hpa.Namespace = "default"
	}

	hpaJson, _ := json.Marshal(hpa)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/scaling", c.apiServerIP, hpa.Namespace), bytes.NewBuffer(hpaJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.Service]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add hpa scaler error: %v", baseResponse.Error)
	}
	return nil
}

func (c *client) DeleteHPAScaler(name, namespace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/scaling/scalingname/%s", c.apiServerIP, namespace, name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var baseResponse v1.BaseResponse[v1.HorizontalPodAutoscaler]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete hpa scaler error: %v", baseResponse.Error)
	}
	return nil
}

func (c *client) UploadPodMetrics(metrics []*v1.PodRawMetrics) error {
	url := fmt.Sprintf("http://%s:8001/api/v1/stats/data", c.apiServerIP)

	metricsStr, _ := json.Marshal(metrics)

	// fmt.Printf("upload metrics str: %s\n", string(metricsStr))

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

func (c *client) GetPodMetrics(metricsQry v1.MetricsQuery) (*v1.PodRawMetrics, error) {
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
	var baseResponse v1.BaseResponse[*v1.PodRawMetrics]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get pod metrics failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetSidecarMapping() (v1.SidecarMapping, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/sidecar-mapping", c.apiServerIP))
	if err != nil {
		return v1.SidecarMapping{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return v1.SidecarMapping{}, err
	}
	var baseResponse v1.BaseResponse[v1.SidecarMapping]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return v1.SidecarMapping{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return v1.SidecarMapping{}, fmt.Errorf("get sidecar mapping failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetAllVirtualServices() ([]*v1.VirtualService, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/virtualservices", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.VirtualService]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get virtual services failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetSubsetByName(name, namespace string) (*v1.Subset, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/subsets/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.Subset]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get virtual services failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddSidecarMapping(maps v1.SidecarMapping) error {
	jsonBytes, err := json.Marshal(&maps)
	if err != nil {
		return err
	}
	// POST to API server
	url := fmt.Sprintf("http://%s:8001/api/v1/sidecar-mapping", c.apiServerIP)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error: %v", resp.Status)
	}
	return nil
}

func (c *client) GetSidecarServiceNameMapping() (v1.SidecarServiceNameMapping, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/sidecar-service-name-mapping", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[v1.SidecarServiceNameMapping]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get sidecar service name mapping failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) GetAllRollingUpdates() ([]*v1.RollingUpdate, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/rollingupdates", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.RollingUpdate]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get rolling updates failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) UpdateRollingUpdateStatus(name, namespace string, status *v1.RollingUpdateStatus) error {
	statusJson, _ := json.Marshal(status)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/rollingupdates/%s", c.apiServerIP, namespace, name), bytes.NewBuffer(statusJson))
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
		return fmt.Errorf("update rolling update status error: %v", resp.Status)
	}
	return nil
}

func (c *client) GetAllSubsets() ([]*v1.Subset, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/subsets", c.apiServerIP))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[[]*v1.Subset]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get subsets failed, error: %s", baseResponse.Error)
	}
	return baseResponse.Data, nil

}

func (c *client) AddSubset(subset *v1.Subset) error {
	subsetJson, _ := json.Marshal(subset)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/subsets", c.apiServerIP, subset.Namespace), bytes.NewBuffer(subsetJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var baseResponse v1.BaseResponse[v1.Subset]
	err = json.NewDecoder(resp.Body).Decode(&baseResponse)
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add subset error: %v", baseResponse.Error)
	}
	return nil
}
func (c *client) GetVirtualService(name, namespace string) (*v1.VirtualService, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/virtualservices/%s", c.apiServerIP, namespace, name))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var baseResponse v1.BaseResponse[*v1.VirtualService]
	err = json.Unmarshal(body, &baseResponse)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get virtual service error: %v", baseResponse.Error)
	}
	return baseResponse.Data, nil
}

func (c *client) AddVirtualService(virtualService *v1.VirtualService) error {
	vsJson, _ := json.Marshal(virtualService)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/virtualservices", c.apiServerIP, virtualService.Namespace), bytes.NewBuffer(vsJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var baseResponse v1.BaseResponse[v1.VirtualService]
	err = json.NewDecoder(resp.Body).Decode(&baseResponse)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("add virtual service error: %v", baseResponse.Error)
	}
	return nil
}

func (c *client) DeleteSubset(subset *v1.Subset) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/subsets/%s", c.apiServerIP, subset.Namespace, subset.Name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete subset error: %v", resp.Status)
	}
	return nil
}

func (c *client) DeleteSubsetByNameNp(subsetName, nameSpace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/subsets/%s", c.apiServerIP, nameSpace, subsetName), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete subset error: %v", resp.Status)
	}
	return nil
}

func (c *client) DeleteVirtualService(virtualService *v1.VirtualService) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/virtualservices/%s", c.apiServerIP, virtualService.Namespace, virtualService.Name), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete virtual service error: %v", resp.Status)
	}
	return nil
}

func (c *client) DeleteVirtualServiceByNameNp(vsName, nameSpace string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:8001/api/v1/namespaces/%s/virtualservices/%s", c.apiServerIP, nameSpace, vsName), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete virtual service error: %v", resp.Status)
	}
	return nil
}
