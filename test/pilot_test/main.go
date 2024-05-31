package main

import (
	"fmt"
	"minikubernetes/pkg/microservice/pilot"
)

func main() {
	rsm := pilot.NewPilot("192.168.1.10")
	err := rsm.Start()
	if err != nil {
		fmt.Println(err)
	}
}

//weight1 := int32(1)
//weight2 := int32(2)
////url1 := "/a"
////url2 := "/b"
//var services []*v1.Service
//var pods []*v1.Pod
//var virtualServices []*v1.VirtualService
//var subsets []*v1.Subset
//pods = append(pods, &v1.Pod{
//ObjectMeta: v1.ObjectMeta{
//Name: "pod-test0",
//Labels: map[string]string{
//"app": "tz",
//},
//},
//Spec: v1.PodSpec{
//Containers: []v1.Container{
//{
//Ports: []v1.ContainerPort{
//{
//ContainerPort: 8080,
//Protocol:      "TCP",
//},
//},
//},
//},
//},
//Status: v1.PodStatus{
//Phase: "Running",
//PodIP: "111.11.11.11",
//},
//})
//pods = append(pods, &v1.Pod{
//ObjectMeta: v1.ObjectMeta{
//Name: "pod-test1",
//Labels: map[string]string{
//"app": "tz",
//},
//},
//Spec: v1.PodSpec{
//Containers: []v1.Container{
//{
//Ports: []v1.ContainerPort{
//{
//ContainerPort: 8090,
//Protocol:      "TCP",
//},
//},
//},
//},
//},
//Status: v1.PodStatus{
//Phase: "Running",
//PodIP: "111.11.11.12",
//},
//})
//pods = append(pods, &v1.Pod{
//ObjectMeta: v1.ObjectMeta{
//Name: "pod-test2",
//Labels: map[string]string{
//"app": "tz",
//},
//},
//Spec: v1.PodSpec{
//Containers: []v1.Container{
//{
//Ports: []v1.ContainerPort{
//{
//ContainerPort: 8080,
//Protocol:      "TCP",
//},
//},
//},
//},
//},
//Status: v1.PodStatus{
//Phase: "Running",
//PodIP: "111.11.11.13",
//},
//})
//services = append(services, &v1.Service{
//ObjectMeta: v1.ObjectMeta{
//UID:       v1.UID("1234"),
//Name:      "service-test0",
//Namespace: "default",
//},
//Spec: v1.ServiceSpec{
//Ports: []v1.ServicePort{{
//Name:       "port0",
//Protocol:   "TCP",
//Port:       80,
//TargetPort: 8080,
//}, {
//Name:       "port1",
//Protocol:   "TCP",
//Port:       90,
//TargetPort: 8090,
//}},
//Selector: map[string]string{
//"app": "tz",
//},
//ClusterIP: "100.2.2.3",
//},
//})
//pods = append(pods, &v1.Pod{})
//virtualServices = append(virtualServices, &v1.VirtualService{
//TypeMeta: v1.TypeMeta{},
//ObjectMeta: v1.ObjectMeta{
//Namespace: "default",
//},
//Spec: v1.VirtualServiceSpec{
//ServiceRef: "service-test0",
//Port:       80,
//Subsets: []v1.VirtualServiceSubset{
//v1.VirtualServiceSubset{
//URL:    nil,
//Weight: &weight1,
//Name:   "subset-test0",
//},
//v1.VirtualServiceSubset{
//URL:    nil,
//Weight: &weight2,
//Name:   "subset-test1",
//},
//},
//},
//})
//subsets = append(subsets, &v1.Subset{
//TypeMeta: v1.TypeMeta{},
//ObjectMeta: v1.ObjectMeta{
//Namespace: "default",
//Name:      "subset-test0",
//},
//Spec: v1.SubsetSpec{
//Pods: []string{
//"pod-test0",
//},
//},
//})
//subsets = append(subsets, &v1.Subset{
//TypeMeta: v1.TypeMeta{},
//ObjectMeta: v1.ObjectMeta{
//Namespace: "default",
//Name:      "subset-test1",
//},
//Spec: v1.SubsetSpec{
//Pods: []string{
//"pod-test2",
//},
//},
//})
