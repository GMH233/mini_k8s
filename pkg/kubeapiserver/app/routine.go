package app

import (
	"encoding/json"
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeapiserver/etcd"
	"minikubernetes/pkg/kubeapiserver/utils"
	"minikubernetes/tools/timestamp"
	"minikubernetes/tools/uuid"

	"net/http"
	"time"

	gin "github.com/gin-gonic/gin"
)

/* this is the simple routine of apiserver */

/* For The apis: */

/* API: [SYSTEM INFO]
   /metrics
   /healthz
   Handle:
   pass to handlers like "metrics.go" by route.
*/

/*
	 API: [CORE GROUPS]
	   /api/v1/...
			  /pods/...

	   Descriptions:
	   Basic command apis of k8s.

	   Handle:
	   Now in KubeApiServer
*/

/* URL Consts
 */
const (
	All_nodes_url   = "/api/v1/node"
	Node_status_url = "/api/v1/nodes/:nodename/status"

	All_pods_url       = "/api/v1/pods"
	Namespace_Pods_url = "/api/v1/namespaces/:namespace/pods"
	Single_pod_url     = "/api/v1/namespaces/:namespace/pods/:podname"
	Pod_status_url     = "/api/v1/namespaces/:namespace/pods/:podname/status"

	Node_pods_url = "/api/v1/nodes/:nodename/pods"

	AllServicesURL       = "/api/v1/services"
	NamespaceServicesURL = "/api/v1/namespaces/:namespace/services"
	SingleServiceURL     = "/api/v1/namespaces/:namespace/services/:servicename"
)

/* NAMESPACE
 * and NODE HARDSHIT
 */
const (
	Default_Namespace = "default"
	Default_Nodename  = "node-0"
	Default_Podname   = "example-pod"
)

type kubeApiServer struct {
	router    *gin.Engine
	listen_ip string
	port      int
	store_cli etcd.Store
}

type KubeApiServer interface {
	Run()
}

/* Simulate etcd.
 */

func (ser *kubeApiServer) init() {
	var err error
	newStore, err := etcd.NewEtcdStore()
	if err != nil {
		log.Panicln("etcd store init failed")
		return
	}
	ser.store_cli = newStore
	// assume that node-0 already registered

	// TODO:(unnecessary) 检查etcdcli是否有效

}

func (ser *kubeApiServer) Run() {
	// debugging
	now := time.Now()
	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()

	log.Println("starting the kubeApiServer")
	log.Printf("at time: %d:%d:%d\n", hour, minute, second)

	// fake init
	log.Println("kubeApiServer init")
	ser.init()

	// exactly initialing
	log.Println("kubeApiServer is binding handlers")
	ser.binder()

	log.Printf("binding ip: %v, listening port: %v\n", ser.listen_ip, ser.port)
	ser.router.Run(ser.listen_ip + ":" + fmt.Sprint(ser.port))

	defer log.Printf("server stop")
}

func NewKubeApiServer() (KubeApiServer, error) {
	// return an kubeapi server
	return &kubeApiServer{
		router:    gin.Default(),
		listen_ip: "0.0.0.0",
		port:      8001,
	}, nil
}

// binding Restful requests to urls
// could initialize with config

func (ser *kubeApiServer) binder() {
	// debug
	ser.router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	ser.router.GET(All_nodes_url, GetNodesHandler)
	ser.router.POST(All_nodes_url, AddNodeHandler)
	ser.router.GET(Node_status_url, GetNodeStatusHandler)
	ser.router.PUT(Node_status_url, PutNodeStatusHandler) // only modify the status of node

	ser.router.GET(All_pods_url, ser.GetAllPodsHandler)
	ser.router.GET(Namespace_Pods_url, ser.GetPodsByNamespaceHandler)
	ser.router.GET(Single_pod_url, ser.GetPodHandler)
	ser.router.POST(Namespace_Pods_url, ser.AddPodHandler) // for single-pod testing
	ser.router.PUT(Single_pod_url, UpdatePodHandler)
	ser.router.DELETE(Single_pod_url, ser.DeletePodHandler)
	ser.router.GET(Pod_status_url, ser.GetPodStatusHandler)
	ser.router.PUT(Pod_status_url, ser.PutPodStatusHandler) // only modify the status of a single pod

	ser.router.GET(Node_pods_url, ser.GetPodsByNodeHandler) // for single-pod testing

	ser.router.GET(AllServicesURL, ser.GetAllServicesHandler)
	ser.router.POST(NamespaceServicesURL, ser.AddServiceHandler)
	ser.router.DELETE(SingleServiceURL, ser.DeleteServiceHandler)
}

// handlers (trivial)

// Nodes have no namespace.
// TODO: 增加Node支持
func GetNodesHandler(con *gin.Context) {

}
func AddNodeHandler(con *gin.Context) {

}

func GetNodeStatusHandler(con *gin.Context) {

}
func PutNodeStatusHandler(con *gin.Context) {

}

// For pods
// We set namespace to "default" right now.

func (ser *kubeApiServer) GetAllPodsHandler(con *gin.Context) {
	log.Println("GetAllPods")

	all_pod_str := make([]v1.Pod, 0)

	prefix := "/registry"

	all_pod_keystr := prefix + "/pods"

	res, err := ser.store_cli.GetSubKeysValues(all_pod_keystr)

	if res == nil || len(res) == 0 || err != nil {
		log.Println("no pod exists")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "no pod exists",
		})
		return
	}
	for _, v := range res {
		var pod v1.Pod
		err = json.Unmarshal([]byte(v), &pod)
		if err != nil {
			log.Println("error in json unmarshal")
			con.JSON(http.StatusInternalServerError, gin.H{
				"error": "error in json unmarshal",
			})
			return
		}
		all_pod_str = append(all_pod_str, pod)
	}

	con.JSON(http.StatusOK,
		gin.H{
			"data": all_pod_str,
		},
	)

}
func (ser *kubeApiServer) GetPodsByNamespaceHandler(con *gin.Context) {
	log.Println("GetPodsByNamespace")

	np := con.Params.ByName("namespace")
	if np == "" {
		log.Panicln("error in parsing namespace ")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "error in parsing namespace ",
		})
		return
	}

	all_pod_str := make([]v1.Pod, 0)

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods"

	res, err := ser.store_cli.GetSubKeysValues(namespace_pod_keystr)

	if res == nil || len(res) == 0 || err != nil {
		log.Println("no pod exists")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "no pod exists",
		})
		return
	}

	for _, v := range res {
		pod_id := v
		all_pod_keystr := prefix + "/pods/" + pod_id

		res, err := ser.store_cli.Get(all_pod_keystr)
		if res == "" || err != nil {
			log.Println("pod does not exist")
			con.JSON(http.StatusNotFound, gin.H{
				"error": "pod does not exist",
			})
			return
		}
		var pod v1.Pod
		err = json.Unmarshal([]byte(res), &pod)
		if err != nil {
			log.Println("error in json unmarshal")
			con.JSON(http.StatusInternalServerError, gin.H{
				"error": "error in json unmarshal",
			})
			return
		}
		all_pod_str = append(all_pod_str, pod)
	}

	con.JSON(http.StatusOK,
		gin.H{
			"data": all_pod_str,
		},
	)

}
func (ser *kubeApiServer) GetPodHandler(con *gin.Context) {
	log.Println("GetPod")

	np := con.Params.ByName("namespace")
	pod_name := con.Params.ByName("podname")

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods/" + pod_name

	res, err := ser.store_cli.Get(namespace_pod_keystr)
	if res == "" || err != nil {
		log.Println("pod name does not exist in namespace")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod name does not exist in namespace",
		})
		return
	}

	pod_id := res
	all_pod_keystr := prefix + "/pods/" + pod_id

	res, err = ser.store_cli.Get(all_pod_keystr)
	if res == "" || err != nil {
		log.Println("pod does not exist")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod does not exist",
		})
		return
	}

	var pod v1.Pod
	err = json.Unmarshal([]byte(res), &pod)
	if err != nil {
		log.Println("error in json unmarshal")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in json unmarshal",
		})
		return
	}

	con.JSON(http.StatusOK, gin.H{
		"data": pod,
	})
}
func (ser *kubeApiServer) AddPodHandler(con *gin.Context) {
	// assign a pod to a node
	log.Println("Adding a new pod")

	var pod v1.Pod
	err := con.ShouldBind(&pod)
	if err != nil {
		log.Panicln("something is wrong when parsing Pod")
		return
	}
	pod_name := pod.ObjectMeta.Name
	if pod_name == "" {
		pod_name = Default_Podname
	}

	pod.ObjectMeta.UID = (v1.UID)(uuid.NewUUID())

	pod.ObjectMeta.CreationTimestamp = timestamp.NewTimestamp()

	pod.Status.Phase = v1.PodPending

	/* fake store pod to:
	1. namespace , store the binding of podname and uid
	2. node , only uid
	*/
	// TODO: 加入shim 解析所谓的api格式到registry前缀的映射
	// TODO: 加入一个keystr解析函数
	// 现在可以简单的理解为将/api/v1/ 替换成/registry/

	prefix := "/registry"

	// 全局用uid而不是podname来标识
	all_pod_keystr := prefix + "/pods/" + string(pod.ObjectMeta.UID)

	// namespace里面对应的是podname和uid的映射
	namespace_pod_keystr := prefix + "/namespaces/" + Default_Namespace + "/pods/" + pod_name

	// node里面对应的也是podname和uid的映射
	node_pod_keystr := prefix + "/nodes/" + Default_Nodename + "/pods/" + pod_name

	// 首先查看namespace里面是否已经存在
	res, err := ser.store_cli.Get(namespace_pod_keystr)
	if res != "" || err != nil {
		log.Println("pod name already exists")
		con.JSON(http.StatusConflict, gin.H{
			"error": "pod name already exists",
		})
		return
	}
	// 然后写入namespace_pod_map
	err = ser.store_cli.Set(namespace_pod_keystr, string(pod.ObjectMeta.UID))
	if err != nil {
		log.Println("error in writing to etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in writing to etcd",
		})
		return
	}

	// 然后写入node_pod_map
	err = ser.store_cli.Set(node_pod_keystr, string(pod.ObjectMeta.UID))
	if err != nil {
		log.Println("error in writing to etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in writing to etcd",
		})
		return
	}

	// 最后写入pod_hub
	// JSON序列化pod
	pod_str, err := json.Marshal(pod)

	if err != nil {
		log.Println("error in json marshal")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in json marshal",
		})
		return
	}

	err = ser.store_cli.Set(all_pod_keystr, string(pod_str))
	if err != nil {
		log.Println("error in writing to etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in writing to etcd",
		})
		return
	}

	con.JSON(http.StatusCreated, gin.H{
		"message": "successfully created pod",
		"UUID":    pod.ObjectMeta.UID,
	})

}

func UpdatePodHandler(con *gin.Context) {

}
func (ser *kubeApiServer) DeletePodHandler(con *gin.Context) {
	log.Println("DeletePod")

	np := con.Params.ByName("namespace")
	pod_name := con.Params.ByName("podname")

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods/" + pod_name

	node_pod_keystr := prefix + "/nodes/" + Default_Nodename + "/pods/" + pod_name

	res, err := ser.store_cli.Get(namespace_pod_keystr)
	// or change return type to DeleteResponse so there is no need to check Get result
	if res == "" || err != nil {
		log.Println("pod name does not exist in namespace")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod name does not exist in namespace",
		})
		return
	}

	pod_id := res
	all_pod_keystr := prefix + "/pods/" + pod_id

	err = ser.store_cli.Delete(namespace_pod_keystr)
	if err != nil {
		log.Println("error in deleting from etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in deleting from etcd",
		})
		return
	}

	res, err = ser.store_cli.Get(node_pod_keystr)
	// or change return type to DeleteResponse so there is no need to check Get result
	if res == "" || err != nil {
		log.Println("pod name does not exist in node")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod name does not exist in node",
		})
		return
	}

	err = ser.store_cli.Delete(node_pod_keystr)
	if err != nil {
		log.Println("error in deleting from etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in deleting from etcd",
		})
		return
	}

	res, err = ser.store_cli.Get(all_pod_keystr)
	if res == "" || err != nil {
		log.Println("pod does not exist")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod does not exist",
		})
		return
	}

	err = ser.store_cli.Delete(all_pod_keystr)
	if err != nil {
		log.Println("error in deleting from etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in deleting from etcd",
		})
		return
	}

	con.JSON(http.StatusOK, gin.H{
		"message": "successfully deleted pod",
	})

}

func (ser *kubeApiServer) GetPodStatusHandler(con *gin.Context) {

	log.Println("GetPodStatus")

	// default here
	np := con.Params.ByName("namespace")
	pod_name := con.Params.ByName("podname")

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods/" + pod_name

	res, err := ser.store_cli.Get(namespace_pod_keystr)
	if res == "" || err != nil {
		log.Println("pod name does not exist in namespace")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod name does not exist in namespace",
		})
		return
	}

	pod_id := res
	all_pod_keystr := prefix + "/pods/" + pod_id

	res, err = ser.store_cli.Get(all_pod_keystr)
	if res == "" || err != nil {
		log.Println("pod does not exist")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod does not exist",
		})
		return
	}

	var pod v1.Pod
	err = json.Unmarshal([]byte(res), &pod)
	if err != nil {
		log.Println("error in json unmarshal")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in json unmarshal",
		})
		return
	}

	pod_status := pod.Status

	con.JSON(http.StatusOK, gin.H{
		"data": pod_status,
	})

}

func (ser *kubeApiServer) PutPodStatusHandler(con *gin.Context) {
	log.Println("PutPodStatus")

	np := con.Params.ByName("namespace")
	pod_name := con.Params.ByName("podname")

	var pod_status v1.PodStatus
	err := con.ShouldBind(&pod_status)
	if err != nil {
		log.Panicln("something is wrong when parsing Pod")
		return
	}

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods/" + pod_name

	res, err := ser.store_cli.Get(namespace_pod_keystr)

	if res == "" || err != nil {
		log.Println("pod name does not exist in namespace")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod name does not exist in namespace",
		})
		return
	}

	pod_id := res
	all_pod_keystr := prefix + "/pods/" + pod_id

	res, err = ser.store_cli.Get(all_pod_keystr)
	if res == "" || err != nil {
		log.Println("pod does not exist")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "pod does not exist",
		})
		return
	}

	var pod v1.Pod

	err = json.Unmarshal([]byte(res), &pod)
	if err != nil {
		log.Println("error in json unmarshal")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in json unmarshal",
		})
		return
	}

	pod.Status = pod_status

	pod_str, err := json.Marshal(pod)
	if err != nil {
		log.Println("error in json marshal")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in json marshal",
		})
		return
	}

	err = ser.store_cli.Set(all_pod_keystr, string(pod_str))
	if err != nil {
		log.Println("error in writing to etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in writing to etcd",
		})
		return
	}

	con.JSON(http.StatusOK, gin.H{
		"message": "successfully updated pod status",
	})
}

func (ser *kubeApiServer) GetPodsByNodeHandler(con *gin.Context) {

	// first parse nodename
	log.Println("GetPodsByNode")

	node_name := con.Params.ByName("nodename")
	if node_name == "" {
		log.Panicln("error in parsing nodename ")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "error in parsing nodename ",
		})
		return
	}
	log.Printf("getting info of node: %v", node_name)

	all_pod_str := make([]v1.Pod, 0)

	prefix := "/registry"

	node_pod_keystr := prefix + "/nodes/" + node_name + "/pods"

	// 以这个前缀去搜索所有的pod
	// 得调整接口 加一个GetSubKeysValues
	res, err := ser.store_cli.GetSubKeysValues(node_pod_keystr)
	//
	if res == nil || err != nil {
		log.Println("node does not exist")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "node does not exist",
		})
		return
	}

	//res返回的是pod的uid，回到etcd里面找到pod的信息
	for _, v := range res {
		pod_id := v
		all_pod_keystr := prefix + "/pods/" + pod_id

		res, err := ser.store_cli.Get(all_pod_keystr)
		if res == "" || err != nil {
			log.Println("pod does not exist")
			con.JSON(http.StatusNotFound, gin.H{
				"error": "pod does not exist",
			})
			return
		}
		var pod v1.Pod
		err = json.Unmarshal([]byte(res), &pod)
		if err != nil {
			log.Println("error in json unmarshal")
			con.JSON(http.StatusInternalServerError, gin.H{
				"error": "error in json unmarshal",
			})
			return
		}
		all_pod_str = append(all_pod_str, pod)
	}

	// then return all of them
	con.JSON(http.StatusOK,
		gin.H{
			"data": all_pod_str,
		},
	)
}

func (s *kubeApiServer) GetAllServicesHandler(c *gin.Context) {
	allSvcKey := "/registry/services"
	res, err := s.store_cli.GetSubKeysValues(allSvcKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in reading from etcd",
		})
		return
	}
	services := make([]v1.Service, 0)
	for _, v := range res {
		var service v1.Service
		err = json.Unmarshal([]byte(v), &service)
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
				Error: "error in json unmarshal",
			})
			return
		}
		services = append(services, service)
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]v1.Service]{
		Data: services,
	})
}

func (s *kubeApiServer) AddServiceHandler(c *gin.Context) {
	var service v1.Service
	err := c.ShouldBind(&service)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "invalid service json",
		})
		return
	}
	if service.Name == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "service name is required",
		})
		return
	}
	if service.Kind != "Service" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "invalid api object kind",
		})
		return
	}
	// 默认协议为TCP
	for i, port := range service.Spec.Ports {
		if port.Protocol == "" {
			service.Spec.Ports[i].Protocol = v1.ProtocolTCP
		}
		if port.TargetPort < 1 || port.TargetPort > 65535 {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
				Error: "invalid target port",
			})
			return
		}
		if port.Port < 1 || port.Port > 65535 {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
				Error: "invalid port",
			})
			return
		}
	}

	namespace := c.Param("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "namespace is required",
		})
		return
	}
	if service.Namespace != "" {
		if service.Namespace != namespace {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
				Error: fmt.Sprintf("namespace mismatch, spec: %s, url: %s", service.Namespace, namespace),
			})
			return
		}
	} else {
		if namespace != "default" {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
				Error: fmt.Sprintf("namespace mismatch, spec: empty(using default), url: %s", namespace),
			})
			return
		}
	}

	// 存uid
	namespaceSvcKey := fmt.Sprintf("/registry/%s/services/%s", namespace, service.Name)

	// 检查service是否已经存在
	result, err := s.store_cli.Get(namespaceSvcKey)
	if err != nil || result != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.Service]{
			Error: fmt.Sprintf("service %s/%s already exists", namespace, service.Name),
		})
		return
	}

	// 查bitmap，分配ip
	bitmapKey := "/registry/IPPool/bitmap"
	bitmapString, err := s.store_cli.Get(bitmapKey)
	bitmap := []byte(bitmapString)
	// etcd里没存bitmap，则初始化bitmap
	if err != nil || bitmapString == "" {
		initialBitmap := make([]byte, utils.IPPoolSize/8)
		bitmap = initialBitmap
		err = s.store_cli.Set(bitmapKey, string(initialBitmap))
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
				Error: "error in writing to etcd",
			})
			return
		}
	}
	ip, err := utils.AllocIP(bitmap)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "no available ip",
		})
		return
	}
	err = s.store_cli.Set(bitmapKey, string(bitmap))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in writing to etcd",
		})
		return
	}
	service.Spec.ClusterIP = ip

	service.Namespace = namespace
	service.UID = v1.UID(uuid.NewUUID())
	service.CreationTimestamp = timestamp.NewTimestamp()

	serviceJson, err := json.Marshal(service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in json marshal",
		})
		return
	}

	// 存service json
	allSvcKey := fmt.Sprintf("/registry/services/%s", service.UID)

	// allSvcKey存service json
	err = s.store_cli.Set(allSvcKey, string(serviceJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in writing to etcd",
		})
		return
	}

	// namespaceSvcKey存uid
	err = s.store_cli.Set(namespaceSvcKey, string(service.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in writing to etcd",
		})
		return
	}

	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.Service]{
		Data: &service,
	})
}

func (s *kubeApiServer) DeleteServiceHandler(c *gin.Context) {
	namespace := c.Param("namespace")
	serviceName := c.Param("servicename")
	if namespace == "" || serviceName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "namespace and service name cannot be empty",
		})
		return
	}
	namespaceSvcKey := fmt.Sprintf("/registry/%s/services/%s", namespace, serviceName)
	uid, err := s.store_cli.Get(namespaceSvcKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.Service]{
			Error: fmt.Sprintf("service %s/%s not found", namespace, serviceName),
		})
		return
	}

	allSvcKey := fmt.Sprintf("/registry/services/%s", uid)
	serviceJson, err := s.store_cli.Get(allSvcKey)
	if err != nil || serviceJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in reading service from etcd",
		})
		return
	}

	var service v1.Service
	err = json.Unmarshal([]byte(serviceJson), &service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in json unmarshal",
		})
		return
	}

	// 释放ip
	bitmapKey := "/registry/IPPool/bitmap"
	bitmapString, err := s.store_cli.Get(bitmapKey)
	if err != nil || bitmapString == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in reading ip pool bit map from etcd",
		})
		return
	}
	bitmap := []byte(bitmapString)
	err = utils.FreeIP(service.Spec.ClusterIP, bitmap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: fmt.Sprintf("error in free ip %s", service.Spec.ClusterIP),
		})
		return
	}
	err = s.store_cli.Set(bitmapKey, string(bitmap))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in writing ip pool bitmap to etcd",
		})
		return
	}

	err = s.store_cli.Delete(namespaceSvcKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in deleting service from etcd",
		})
		return
	}
	err = s.store_cli.Delete(allSvcKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in deleting service from etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.Service]{
		Data: &service,
	})
}
