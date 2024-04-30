package utils

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	v1 "minikubernetes/pkg/api/v1"
)

func JSON2YAML(jsonBytes []byte) ([]byte, error) {
	var intermediate map[string]interface{}
	err := json.Unmarshal(jsonBytes, &intermediate)
	if err != nil {
		return nil, err
	}
	yamlBytes, err := yaml.Marshal(intermediate)
	if err != nil {
		return nil, err
	}
	return yamlBytes, nil
}

func YAML2JSON(yamlBytes []byte) ([]byte, error) {
	var intermediate map[string]interface{}
	err := yaml.Unmarshal(yamlBytes, &intermediate)
	if err != nil {
		return nil, err
	}
	jsonBytes, err := json.Marshal(intermediate)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func Pod2YAML(pod *v1.Pod) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(pod)
	if err != nil {
		return nil, err
	}
	return yamlBytes, nil
}

func YAML2Pod(yamlBytes []byte) (*v1.Pod, error) {
	var pod v1.Pod
	err := yaml.Unmarshal(yamlBytes, &pod)
	if err != nil {
		return nil, err
	}
	return &pod, nil
}

func Pod2JSON(pod *v1.Pod) ([]byte, error) {
	jsonBytes, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func JSON2Pod(jsonBytes []byte) (*v1.Pod, error) {
	var pod v1.Pod
	err := json.Unmarshal(jsonBytes, &pod)
	if err != nil {
		return nil, err
	}
	return &pod, nil
}
