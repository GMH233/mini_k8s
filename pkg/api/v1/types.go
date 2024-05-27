package v1

import (
	"time"
)

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

	SecurityContext *SecurityContext `json:"securityContext,omitempty"`
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

	InitContainers []Container `json:"initContainers,omitempty"`
	// 重启策略：仅由kubelet实现
	RestartPolicy RestartPolicy `json:"restartPolicy,omitempty"`

	//Sidecar *SidecarSpec `json:"sidecar,omitempty"`
}

type SecurityContext struct {
	Privileged *bool  `json:"privileged,omitempty"`
	RunAsUser  *int64 `json:"runAsUser,omitempty"`
}

//type SidecarSpec struct {
//	Inject bool `json:"inject,omitempty"`
//}

type PodStatus struct {
	Phase PodPhase `json:"phase,omitempty"`
	PodIP string   `json:"podIP,omitempty"`
}

type MetricsQuery struct {
	// 查询的资源
	UID UID `form:"uid" json:"uid,omitempty"`
	// 当前的时间戳
	TimeStamp time.Time `form:"timestamp" json:"timestamp,omitempty"`
	// 查询的时间窗口 (秒)
	Window int32 `form:"window" json:"window,omitempty"`
}

// PodRawMetrics 用于标记Pod的资源使用情况, 作为stat接口的传输结构
type PodRawMetrics struct {
	UID UID `json:"uid,omitempty"`
	// 各个容器的资源使用情况
	ContainerInfo map[string][]PodRawMetricsItem `json:"containers,omitempty"`
}

// 会再增加统计的条目的
type PodRawMetricsItem struct {
	// 保证切片单元的时间戳是对齐的
	TimeStamp time.Time `json:"timestamp,omitempty"`
	// 单个容器的CPU使用率，以百分比计算
	CPUUsage float32 `json:"cpuUsage"`
	// 单个容器内存使用量,以MB计算
	MemoryUsage float32 `json:"memoryUsage"`
}

type ScalerType string

const (
	// // ReplicaSet
	// ScalerTypeReplicaSet ScalerType = "ReplicaSet"
	// // Deployment
	// ScalerTypeDeployment ScalerType = "Deployment"
	// HorizontalPodAutoscaler
	ScalerTypeHPA ScalerType = "HorizontalPodAutoscaler"
)

type CrossVersionObjectReference struct {
	// 例如ReplicaSet，Deployment等
	Kind string `json:"kind"`
	// 对象名称
	Name string `json:"name"`
	// 默认v1
	APIVersion string `json:"apiVersion,omitempty"`
}

type HorizontalPodAutoscaler struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`

	Spec HorizontalPodAutoscalerSpec `json:"spec,omitempty"`
}

type HorizontalPodAutoscalerSpec struct {
	ScaleTargetRef CrossVersionObjectReference `json:"scaleTargetRef"`

	// 期望的最小副本数
	MinReplicas int32 `json:"minReplicas,omitempty"`
	// 期望的最大副本数
	MaxReplicas int32 `json:"maxReplicas"`

	// 扩缩容的时间窗口
	ScaleWindowSeconds int32 `json:"scaleWindowSeconds,omitempty"`

	// 监控的资源指标
	// Metrics []MetricSpec `json:"metrics,omitempty"`
	Metrics []ResourceMetricSource `json:"metrics,omitempty"`
	// 行为
	Behavior HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
}

type ResourceMetricSource struct {
	// name is the name of the resource in question.
	Name ResourceName `json:"name"`

	// target specifies the target value for the given metric
	Target MetricTarget `json:"target"`
}

// MetricTarget defines the target value, average value, or average utilization of a specific metric
type MetricTarget struct {
	// type represents whether the metric type is Utilization, AverageValue
	Type MetricTargetType `json:"type"`

	// averageValue is the target value of the average of the
	// metric across all relevant pods (as a quantity)
	// 和具体的指标单位有关，内存是MB
	AverageValue float32 `json:"averageValue,omitempty"`

	// averageUtilization is the target value of the average of the
	// resource metric across all relevant pods, represented as a percentage of
	// the requested value of the resource for the pods.
	// Currently only valid for Resource metric source type
	// 0 ~ 100 (百分比)
	AverageUtilization float32 `json:"averageUtilization,omitempty"`

	// 触发扩缩容的阈值（百分比，可能大于100）
	// 上限，默认是min(0.5*（100+目标使用率）, 1.5*目标使用率)
	UpperThreshold float32 `json:"upperThreshold,omitempty"`
	// 下限，默认是0.5*目标使用率
	LowerThreshold float32 `json:"lowerThreshold,omitempty"`
}

// MetricTargetType specifies the type of metric being targeted, and should be either
// "AverageValue", or "Utilization"
// AverageValue指的是使用量的真值，Utilization指的是使用率
// 完全匹配的value现在不支持
// 也许之后会通过维持偏差范围来扩缩容 不保真
type MetricTargetType string

const (
	// UtilizationMetricType declares a MetricTarget is an AverageUtilization value
	UtilizationMetricType MetricTargetType = "Utilization"
	// AverageValueMetricType declares a MetricTarget is an
	AverageValueMetricType MetricTargetType = "AverageValue"
)

// HorizontalPodAutoscalerBehavior configures the scaling behavior of the target
// in both Up and Down directions (scaleUp and scaleDown fields respectively).
type HorizontalPodAutoscalerBehavior struct {
	// 扩容的策略
	// 现在扩缩容都只各支持一种策略
	ScaleUp *HPAScalingPolicy `json:"scaleUp,omitempty"`

	// scaleDown is scaling policy for scaling Down.
	// If not set, the default value is to allow to scale down to minReplicas pods, with a
	// 300 second stabilization window (i.e., the highest recommendation for
	// the last 300sec is used).
	// 缩容的策略
	ScaleDown *HPAScalingPolicy `json:"scaleDown,omitempty"`
}

// HPAScalingPolicyType is the type of the policy which could be used while making scaling decisions.
type HPAScalingPolicyType string

const (
	// PodsScalingPolicy is a policy used to specify a change in absolute number of pods.
	PodsScalingPolicy HPAScalingPolicyType = "Pods"

	// PercentScalingPolicy is a policy used to specify a relative amount of change with respect to
	// the current number of pods.
	// 百分比的支持
	PercentScalingPolicy HPAScalingPolicyType = "Percent"
)

// HPAScalingPolicy is a single policy which must hold true for a specified past interval.
type HPAScalingPolicy struct {
	// type is used to specify the scaling policy.
	Type HPAScalingPolicyType `json:"type"`

	// value contains the amount of change which is permitted by the policy.
	// It must be greater than zero
	Value int32 `json:"value"`

	// periodSeconds specifies the window of time for which the policy should hold true.
	// PeriodSeconds must be greater than zero and less than or equal to 1800 (30 min).
	PeriodSeconds int32 `json:"periodSeconds"`
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

const (
	NodePortMin = 30000
	NodePortMax = 32767
	PortMin     = 1
	PortMax     = 65535
)

type ServicePort struct {
	Name string `json:"name,omitempty"`
	// 默认为TCP
	Protocol Protocol `json:"protocol,omitempty"`
	// 端口号， 1-65535
	Port int32 `json:"port"`
	// 目标端口号，1-65535
	TargetPort int32 `json:"targetPort"`
	// type为NodePort时，指定的端口号
	NodePort int32 `json:"nodePort,omitempty"`
}

type Protocol string

const (
	ProtocolTCP Protocol = "tcp"
	ProtocolUDP Protocol = "udp"
)

type DNS struct {
	TypeMeta `json:",inline"`

	ObjectMeta `json:"metadata,omitempty"`

	Spec DNSSpec `json:"spec,omitempty"`

	Status DNSStatus `json:"status,omitempty"`
}

//spec:
//	rules:
//	- host: mywebsite.com
//	  paths:
//    - path: /demo
//		backend:
//		  service:
//			name: myservice
//			port: 8080

type DNSSpec struct {
	Rules []DNSRule `json:"rules,omitempty"`
}

type DNSRule struct {
	Host  string    `json:"host,omitempty"`
	Paths []DNSPath `json:"paths,omitempty"`
}

type DNSPath struct {
	Path    string     `json:"path,omitempty"`
	Backend DNSBackend `json:"backend,omitempty"`
}

type DNSBackend struct {
	Service DNSServiceBackend `json:"service,omitempty"`
}

type DNSServiceBackend struct {
	Name string `json:"name,omitempty"`
	Port int32  `json:"port,omitempty"`
}

type DNSStatus struct {
}

type Node struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       NodeSpec   `json:"spec,omitempty"`
	Status     NodeStatus `json:"status,omitempty"`
}

type NodeSpec struct {
}

type NodeStatus struct {
	// IP地址
	Address string `json:"address,omitempty"`
}

// ServiceName -> ClusterIP
type SidecarServiceNameMapping map[string]string

type SidecarMapping map[string][]SidecarEndpoints

type SidecarEndpoints struct {
	Weight    *int32           `json:"weight,omitempty"`
	URL       *string          `json:"url,omitempty"`
	Endpoints []SingleEndpoint `json:"endpoints,omitempty"`
}

type SingleEndpoint struct {
	IP         string `json:"ip"`
	TargetPort int32  `json:"targetPort"`
}

type VirtualService struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       VirtualServiceSpec `json:"spec,omitempty"`
}

type VirtualServiceSpec struct {
	ServiceRef string                 `json:"serviceRef,omitempty"`
	Port       int32                  `json:"port,omitempty"`
	Subsets    []VirtualServiceSubset `json:"subsets,omitempty"`
}

type VirtualServiceSubset struct {
	Name string `json:"name,omitempty"`
	// Weight subset中的所有endpoint都有相同的权重
	Weight *int32 `json:"weight,omitempty"`
	// URL 路径为URL时，转发到本subset中的endpoint，支持带正则表达式的路径
	URL *string `json:"url,omitempty"`
}

type Subset struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata,omitempty"`
	Spec       SubsetSpec `json:"spec,omitempty"`
}

type SubsetSpec struct {
	// pod名
	Pods []string `json:"pods,omitempty"`
}
