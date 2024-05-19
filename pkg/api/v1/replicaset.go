package v1

// ReplicaSet : 用来存储备份的ReplicaSet类
type ReplicaSet struct {
	TypeMeta `json:",inline"`

	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// ReplicaSet内规格
	Spec ReplicaSetSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// ReplicaSet内创建pod的模板
	Template PodTemplateSpec `json:"template,omitempty" yaml:"template" protobuf:"bytes,3,opt,name=template"`
}

// ReplicaSetSpec : ReplicaSet内规格类
type ReplicaSetSpec struct {
	// 备份数
	Replicas int32 `json:"replicas,omitempty" protobuf:"varint,1,opt,name=replicas"`

	// 筛选相关联pod的selector
	Selector *LabelSelector `json:"selector" protobuf:"bytes,2,opt,name=selector"`

	// ReplicaSet内创建pod的模板
	Template PodTemplateSpec `json:"template,omitempty" protobuf:"bytes,3,opt,name=template"`
}

type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty" yaml:"matchLabels" protobuf:"bytes,1,rep,name=matchLabels"`
}

type PodTemplateSpec struct {
	ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec PodSpec `json:"spec,omitempty"`
}
