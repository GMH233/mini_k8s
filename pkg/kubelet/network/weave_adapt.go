package network

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

func Attach(containerId string) (string, error) {
	cmd := exec.Command("weave", "attach", containerId)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("weave attach err: %v, stderr: %v", err.Error(), stderr.String())
	}
	IPString := stdout.String()
	IPString = strings.TrimSpace(IPString)
	if address := net.ParseIP(IPString); address == nil {
		return "", fmt.Errorf("weave returns invalid IP address: %s", IPString)
	}
	return IPString, nil
}

func Detach(containerId string) error {
	cmd := exec.Command("weave", "detach", containerId)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("weave detach err: %v, stderr: %v", err.Error(), stderr.String())
	}
	IPString := stdout.String()
	IPString = strings.TrimSpace(IPString)
	if address := net.ParseIP(IPString); address == nil {
		return fmt.Errorf("weave detach failed, expect detached IP, actual: %s", IPString)
	}
	return nil
}
