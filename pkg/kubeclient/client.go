package kubeclient

import (
	"encoding/json"
	"fmt"
	"io"
	v1 "minikubernetes/pkg/api/v1"
	"net/http"
)

type Client interface {
	GetAllServices() ([]*v1.Service, error)
	GetAllPods() ([]*v1.Pod, error)
	GetAllDNS() ([]*v1.DNS, error)
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
