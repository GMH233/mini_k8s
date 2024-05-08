package v1

import "time"

type UID string

type PodPhase string

const (
	// 已创建，未运行
	PodPending PodPhase = "Pending"
	// 正在运行
	PodRunning PodPhase = "Running"
	// 已退出，退出代码0
	PodSucceeded PodPhase = "Succeeded"
	// 已退出，退出代码非0
	PodFailed PodPhase = "Failed"
)

type ResourceName string

const (
	// CPU核数
	ResourceCPU ResourceName = "cpu"
	// 内存大小
	ResourceMemory ResourceName = "memory"
)

// 暂时用string表示资源，动态解析
type ResourceList map[ResourceName]string

type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "Always"
	RestartPolicyOnFailure RestartPolicy = "OnFailure"
	RestartPolicyNever     RestartPolicy = "Never"
)

type TypeMeta struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

type ObjectMeta struct {
	Name string `json:"name,omitempty"`

	Namespace string `json:"namespace,omitempty"`

	// 由apiserver生成
	UID UID `json:"uid,omitempty"`

	// 由apiserver生成
	CreationTimestamp time.Time `json:"creationTimestamp,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`
}

type Volume struct {
	Name         string `json:"name"`
	VolumeSource `json:",inline"`
}

type VolumeSource struct {
	HostPath *HostPathVolumeSource `json:"hostPath,omitempty"`
	EmptyDir *EmptyDirVolumeSource `json:"emptyDir,omitempty"`
}

// 挂载主机目录
// volume:
//   - name: xxx
//     hostPath:
//     path: /data
type HostPathVolumeSource struct {
	Path string `json:"path"`
}

// 自动创建空目录
// volume:
//   - name: xxx
//     emptyDir: {}
type EmptyDirVolumeSource struct {
}

type Container struct {
	// 容器名称
	Name string `json:"name"`
	// 容器镜像（包括版本）
	Image string `json:"image,omitempty"`
	// entrypoint命令
	Command []string `json:"command,omitempty"`
	// 容器暴露端口
	Ports []ContainerPort `json:"ports,omitempty"`
	// 容器资源限制
	Resources ResourceRequirements `json:"resources,omitempty"`
	// 卷挂载点
	VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`
}

// ports:
//   - containerPort: 80
type ContainerPort struct {
	ContainerPort int32    `json:"containerPort"`
	Protocol      Protocol `json:"protocol,omitempty"`
}

// resources:
//
//	limits:
//	  cpu: 2
//	requests:
//	  cpu: 1
type ResourceRequirements struct {
	// 资源上限
	Limits ResourceList `json:"limits,omitempty"`
	// 资源下限
	Requests ResourceList `json:"requests,omitempty"`
}

// volumeMounts:
//   - name: xxx
//     mountPath: /data
type VolumeMount struct {
	// 挂载卷的名称
	Name string `json:"name"`
	// 挂载在容器内的路径
	MountPath string `json:"mountPath,omitempty"`
}

type Pod struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PodSpec   `json:"spec,omitempty"`
	Status     PodStatus `json:"status,omitempty"`
}

type PodSpec struct {
	// 卷声明
	Volumes []Volume `json:"volumes,omitempty"`
	// 容器声明
	Containers []Container `json:"containers,omitempty"`
	// 重启策略：仅由kubelet实现
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`
}

type PodStatus struct {
	Phase PodPhase `json:"phase,omitempty"`
	PodIP string   `json:"podIP,omitempty"`
}

type Service struct {
	TypeMeta `json:",inline"`

	ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceSpec `json:"spec,omitempty"`

	Status ServiceStatus `json:"status,omitempty"`
}

type ServiceSpec struct {
	// 类型，默认为ClusterIP
	Type ServiceType `json:"type,omitempty"`
	// 端口声明
	Ports []ServicePort `json:"ports,omitempty"`
	// 选择器，对应pod label
	Selector map[string]string `json:"selector,omitempty"`
	// 创建时由apiserver随机分配
	ClusterIP string `json:"clusterIP,omitempty"`
}

type ServiceStatus struct {
}

type ServiceType string

const (
	ServiceTypeClusterIP ServiceType = "ClusterIP"
	ServiceTypeNodePort  ServiceType = "NodePort"
)

type ServicePort struct {
	Name string `json:"name,omitempty"`
	// 默认为TCP
	Protocol Protocol
	// 端口号
	Port int32
	// 目标端口号
	TargetPort int32
	// type为NodePort时，指定的端口号
	NodePort int32 `json:"nodePort,omitempty"`
}

type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)
