#!/bin/bash

go build ./cmd/kube-apiserver/apiserver.go
go build ./cmd/kubelet/kubelet.go
go build ./cmd/kubeproxy/kubeproxy.go
go build ./cmd/scheduler/scheduler.go
go build ./cmd/kube-controller-manager/controllerManager.go
go build ./cmd/pilot/pilot.go
go build ./cmd/kubectl/kubectl.go
