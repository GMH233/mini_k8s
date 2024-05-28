package pv

import (
	"bufio"
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	kubeletutils "minikubernetes/pkg/kubelet/utils"
	"minikubernetes/pkg/utils"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Controller interface {
	Run()
}

type pvController struct {
	client kubeclient.Client
	hostIP string
}

func NewPVController(apiServerIP string) (Controller, error) {
	pc := &pvController{}
	pc.client = kubeclient.NewClient(apiServerIP)
	hostIP, err := kubeletutils.GetHostIP()
	if err != nil {
		return nil, err
	}
	pc.hostIP = hostIP
	return pc, nil
}

func (pc *pvController) Run() {
	syncPV := true
	for {
		pc.syncLoopIteration(syncPV)
		syncPV = !syncPV
		time.Sleep(2396 * time.Millisecond)
	}
}

func (pc *pvController) syncLoopIteration(syncPV bool) {
	// 交替执行pv和pvc的同步
	if syncPV {
		pc.syncLoopIterationPV()
	} else {
		pc.syncLoopIterationPVC()
	}
}

func (pc *pvController) syncLoopIterationPV() {
	// 获取所有pv
	log.Printf("Checking PV...")
	allPV, err := pc.client.GetAllPVs()
	if err != nil {
		log.Printf("get all pv failed, err: %v\n", err)
		return
	}
	// 遍历pv，若处于Pending状态，则创建
	var pvUpdates []*v1.PersistentVolume
	for _, pv := range allPV {
		if pv.Status.Phase == v1.VolumePending {
			path, err := pc.createPV(pv)
			if err != nil {
				log.Printf("create pv %v failed, err: %v\n", pv.Name, err)
				continue
			}
			pv.Status.Path = path
			pv.Status.Server = pc.hostIP
			pv.Status.Phase = v1.VolumeAvailable
			pvUpdates = append(pvUpdates, pv)
			log.Printf("pv %v created\n", pv.Name)
		}
	}
	for _, pv := range pvUpdates {
		err := pc.client.UpdatePVStatus(pv)
		if err != nil {
			log.Printf("update pv %v failed, err: %v\n", pv.Name, err)
		}
	}
	log.Println("PV sync done")
}

func (pc *pvController) createPV(pv *v1.PersistentVolume) (string, error) {
	// 修改/etc/exports
	hostRule := "192.168.1.0/24(rw,sync,no_root_squash,no_subtree_check)"
	dirName := "/data/minik8s/pv/" + string(pv.UID)
	filename := "/etc/exports"
	exportsFile, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(exportsFile)
	content := ""
	// 检查文件内容
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			content += line + "\n"
			continue
		}
		lineFields := strings.Fields(line)
		if len(lineFields) != 2 {
			return "", fmt.Errorf("invalid /etc/exports file")
		}
		if lineFields[0] == dirName {
			return "", fmt.Errorf("pv already exists in /etc/exports")
		}
		content += line + "\n"
	}
	// 创建目录
	err = os.MkdirAll(dirName, 0755)
	if err != nil {
		return "", err
	}
	// 写入文件
	content += dirName + " " + hostRule + "\n"
	err = os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	// 执行exportfs -ra应用修改
	output, err := exec.Command("exportfs", "-ra").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("exportfs -ra failed, err: %v, output: %s", err, output)
	}
	return dirName, nil
}

func (pc *pvController) syncLoopIterationPVC() {
	log.Println("Checking PVC...")
	// 获取所有pv和pvc
	allPV, err := pc.client.GetAllPVs()
	if err != nil {
		log.Printf("get all pv failed, err: %v\n", err)
		return
	}
	allPVC, err := pc.client.GetAllPVCs()
	if err != nil {
		log.Printf("get all pvc failed, err: %v\n", err)
		return
	}
	var pvUpdates []*v1.PersistentVolume
	var pvcUpdates []*v1.PersistentVolumeClaim
	var newPVs []*v1.PersistentVolume
	pvStorageClassNameMap := make(map[string][]*v1.PersistentVolume)
	for _, pv := range allPV {
		if pv.Spec.StorageClassName == "" {
			continue
		}
		key := pv.Spec.StorageClassName
		pvStorageClassNameMap[key] = append(pvStorageClassNameMap[key], pv)
	}
	// 遍历pvc，若对应pv存在且为available状态，则绑定
	// 若为pending，忽略
	// 若不存在，动态创建
	for _, pvc := range allPVC {
		if pvc.Status.Phase != v1.ClaimPending {
			continue
		}
		storageClass := pvc.Spec.StorageClassName
		if storageClass == "" {
			// 不可动态创建
			continue
		}
		var targetPV *v1.PersistentVolume
		if pvs, ok := pvStorageClassNameMap[storageClass]; ok {
			targetPV = findMatchedPV(pvc, pvs)
		}
		if targetPV == nil {
			// 动态创建
			newPV := &v1.PersistentVolume{
				TypeMeta: v1.TypeMeta{
					Kind:       "PersistentVolume",
					APIVersion: "v1",
				},
				ObjectMeta: v1.ObjectMeta{
					Name: fmt.Sprintf("pv-%s", pvc.Name),
					Namespace: pvc.Namespace,
				},
				Spec: v1.PersistentVolumeSpec{
					Capacity: pvc.Spec.Request,
					StorageClassName: storageClass,
				},
			}
			newPVs = append(newPVs, newPV)
			log.Printf("dynamically created pv %v\n", newPV.Name)
		} else {
			// 绑定
			pvc.Status.VolumeName = targetPV.Name
			pvc.Status.Phase = v1.ClaimBound
			pvcUpdates = append(pvcUpdates, pvc)
			targetPV.Status.ClaimName = pvc.Name
			targetPV.Status.Phase = v1.VolumeBound
			pvUpdates = append(pvUpdates, targetPV)
			log.Printf("pvc %v bound to pv %v\n", pvc.Name, targetPV.Name)
		}
	}
	for _, pv := range newPVs {
		err := pc.client.AddPV(pv)
		if err != nil {
			log.Printf("add pv %v failed, err: %v\n", pv.Name, err)
		}
	}
	for _, pv := range pvUpdates {
		err := pc.client.UpdatePVStatus(pv)
		if err != nil {
			log.Printf("update pv %v failed, err: %v\n", pv.Name, err)
		}
	}
	for _, pvc := range pvcUpdates {
		err := pc.client.UpdatePVCStatus(pvc)
		if err != nil {
			log.Printf("update pvc %v failed, err: %v\n", pvc.Name, err)
		}
	}
	log.Println("PVC sync done")
}

func findMatchedPV(pvc *v1.PersistentVolumeClaim, pvs []*v1.PersistentVolume) *v1.PersistentVolume {
	for _, pv := range pvs {
		if pv.Namespace != pvc.Namespace {
			continue
		}
		if pv.Status.Phase != v1.VolumeAvailable {
			continue
		}
		request, err := utils.ParseSize(pvc.Spec.Request)
		if err != nil {
			log.Printf("parse size failed, err: %v\n", err)
			continue
		}
		capacity, err := utils.ParseSize(pv.Spec.Capacity)
		if err != nil {
			log.Printf("parse size failed, err: %v\n", err)
			continue
		}
		if request <= capacity {
			return pv
		}
	}
	return nil
}
