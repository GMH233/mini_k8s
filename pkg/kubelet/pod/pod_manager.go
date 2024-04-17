package pod

import (
	v1 "minikubernetes/pkg/api/v1"
	"sync"
)

type Manager interface {
	GetPodByFullName(podFullName string) (*v1.Pod, bool)
	GetPodByUid(podUid v1.UID) (*v1.Pod, bool)
	GetPods() []*v1.Pod
	UpdatePod(pod *v1.Pod)
	DeletePod(pod *v1.Pod)
}

type podManager struct {
	lock           sync.RWMutex
	podUidMap      map[v1.UID]*v1.Pod
	podFullNameMap map[string]*v1.Pod
}

func NewPodManager() Manager {
	return &podManager{
		podUidMap:      make(map[v1.UID]*v1.Pod),
		podFullNameMap: make(map[string]*v1.Pod),
	}
}

func (pm *podManager) GetPodByFullName(podFullName string) (*v1.Pod, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	pod, ok := pm.podFullNameMap[podFullName]
	return pod, ok
}

func (pm *podManager) GetPodByUid(podUid v1.UID) (*v1.Pod, bool) {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	pod, ok := pm.podUidMap[podUid]
	return pod, ok
}

func (pm *podManager) GetPods() []*v1.Pod {
	pm.lock.RLock()
	defer pm.lock.RUnlock()
	pods := make([]*v1.Pod, 0, len(pm.podUidMap))
	for _, pod := range pm.podUidMap {
		pods = append(pods, pod)
	}
	return pods
}

func getPodFullName(pod *v1.Pod) string {
	return pod.ObjectMeta.Namespace + "/" + pod.ObjectMeta.Name
}

func (pm *podManager) UpdatePod(pod *v1.Pod) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	pm.podUidMap[pod.ObjectMeta.UID] = pod
	pm.podFullNameMap[getPodFullName(pod)] = pod
}

func (pm *podManager) DeletePod(pod *v1.Pod) {
	pm.lock.Lock()
	defer pm.lock.Unlock()
	delete(pm.podUidMap, pod.ObjectMeta.UID)
	delete(pm.podFullNameMap, getPodFullName(pod))
}
