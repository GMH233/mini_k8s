package client

import (
	"encoding/json"
	"testing"
)

const (
	apiServerIP = "10.119.12.123"
)

func TestKubeletClient_GetPodsByNodeName(t *testing.T) {
	kubeClient := NewKubeletClient(apiServerIP)
	pods, err := kubeClient.GetPodsByNodeName("node-0")
	if err != nil {
		// CI/CD无法访问到api server
		t.Log(err)
		return
	}
	podsJson, err := json.Marshal(pods)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(podsJson))
}
