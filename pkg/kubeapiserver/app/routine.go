package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeapiserver/etcd"
	"minikubernetes/pkg/kubeapiserver/metrics"
	"minikubernetes/pkg/kubeapiserver/utils"
	"minikubernetes/tools/timestamp"
	"minikubernetes/tools/uuid"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	"net/http"
	"time"

	gin "github.com/gin-gonic/gin"
	kubeutils "minikubernetes/pkg/utils"
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

	AllDNSURL       = "/api/v1/dns"
	NamespaceDNSURL = "/api/v1/namespaces/:namespace/dns"
	SingleDNSURL    = "/api/v1/namespaces/:namespace/dns/:dnsname"

	RegisterNodeURL    = "/api/v1/nodes/register"
	UnregisterNodeURL  = "/api/v1/nodes/unregister"
	AllNodesURL        = "/api/v1/nodes"
	SchedulePodURL     = "/api/v1/schedule"
	UnscheduledPodsURL = "/api/v1/pods/unscheduled"

	AllReplicaSetsURL       = "/api/v1/replicasets"
	NamespaceReplicaSetsURL = "/api/v1/namespaces/:namespace/replicasets"
	SingleReplicaSetURL     = "/api/v1/namespaces/:namespace/replicasets/:replicasetname"

	StatsDataURL         = "/api/v1/stats/data"
	AllScalingURL        = "/api/v1/scaling"
	NamespaceScalingsURL = "/api/v1/namespaces/:namespace/scaling"
	SingleScalingURL     = "/api/v1/namespaces/:namespace/scaling/scalingname/:name"

	AllVirtualServicesURL       = "/api/v1/virtualservices"
	NamespaceVirtualServicesURL = "/api/v1/namespaces/:namespace/virtualservices"
	SingleVirtualServiceURL     = "/api/v1/namespaces/:namespace/virtualservices/:virtualservicename"

	AllSubsetsURL       = "/api/v1/subsets"
	NamespaceSubsetsURL = "/api/v1/namespaces/:namespace/subsets"
	SingleSubsetURL     = "/api/v1/namespaces/:namespace/subsets/:subsetname"

	SidecarMappingURL            = "/api/v1/sidecar-mapping"
	SidecarServiceNameMappingURL = "/api/v1/sidecar-service-name-mapping"

	AllPVURL       = "/api/v1/persistentvolumes"
	NamespacePVURL = "/api/v1/namespaces/:namespace/persistentvolumes"
	SinglePVURL    = "/api/v1/namespaces/:namespace/persistentvolumes/:pvname"
	PVStatusURL    = "/api/v1/namespaces/:namespace/persistentvolumes/:pvname/status"

	AllPVCURL       = "/api/v1/persistentvolumeclaims"
	NamespacePVCURL = "/api/v1/namespaces/:namespace/persistentvolumeclaims"
	SinglePVCURL    = "/api/v1/namespaces/:namespace/persistentvolumeclaims/:pvcname"
	PVCStatusURL    = "/api/v1/namespaces/:namespace/persistentvolumeclaims/:pvcname/status"
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
	router      *gin.Engine
	listen_ip   string
	port        int
	store_cli   etcd.Store
	metrics_cli metrics.MetricsDatabase

	lock sync.Mutex
}

type KubeApiServer interface {
	Run()
}

func (ser *kubeApiServer) init() {
	var err error
	newStore, err := etcd.NewEtcdStore()
	if err != nil {
		log.Panicln("etcd store init failed")
		return
	}
	ser.store_cli = newStore

	metricsDb, err := metrics.NewMetricsDb()
	if err != nil {
		log.Panicln("metrics db init failed")
		return
	}

	ser.metrics_cli = metricsDb

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

	ser.router.GET(AllDNSURL, ser.GetAllDNSHandler)
	ser.router.POST(NamespaceDNSURL, ser.AddDNSHandler)
	ser.router.DELETE(SingleDNSURL, ser.DeleteDNSHandler)

	ser.router.GET(AllNodesURL, ser.GetAllNodesHandler)
	ser.router.POST(RegisterNodeURL, ser.RegisterNodeHandler)
	ser.router.POST(UnregisterNodeURL, ser.UnregisterNodeHandler)
	ser.router.POST(SchedulePodURL, ser.SchedulePodToNodeHandler)
	ser.router.GET(UnscheduledPodsURL, ser.GetUnscheduledPodHandler)

	ser.router.GET(AllReplicaSetsURL, ser.GetAllReplicaSetsHandler)
	ser.router.POST(NamespaceReplicaSetsURL, ser.AddReplicaSetHandler)
	ser.router.GET(SingleReplicaSetURL, ser.GetReplicaSetHandler)
	ser.router.PUT(SingleReplicaSetURL, ser.UpdateReplicaSetHandler)
	ser.router.DELETE(SingleReplicaSetURL, ser.DeleteReplicaSetHandler)

	ser.router.GET(StatsDataURL, ser.GetStatsDataHandler)
	ser.router.POST(StatsDataURL, ser.AddStatsDataHandler)

	ser.router.GET(AllScalingURL, ser.GetAllScalingHandler)
	ser.router.POST(NamespaceScalingsURL, ser.AddScalingHandler)
	ser.router.DELETE(SingleScalingURL, ser.DeleteScalingHandler)

	ser.router.GET(AllVirtualServicesURL, ser.GetAllVirtualServicesHandler)
	ser.router.POST(NamespaceVirtualServicesURL, ser.AddVirtualServiceHandler)
	ser.router.DELETE(SingleVirtualServiceURL, ser.DeleteVirtualServiceHandler)

	ser.router.GET(AllSubsetsURL, ser.GetAllSubsetsHandler)
	ser.router.POST(NamespaceSubsetsURL, ser.AddSubsetHandler)
	ser.router.DELETE(SingleSubsetURL, ser.DeleteSubsetHandler)
	ser.router.GET(SingleSubsetURL, ser.GetSubsetHandler)

	ser.router.GET(SidecarMappingURL, ser.GetSidecarMapping)
	ser.router.POST(SidecarMappingURL, ser.SaveSidecarMapping)
	ser.router.GET(SidecarServiceNameMappingURL, ser.GetSidecarServiceNameMapping)

	ser.router.GET(AllPVURL, ser.GetAllPVsHandler)
	ser.router.POST(NamespacePVURL, ser.AddPVHandler)
	ser.router.GET(SinglePVURL, ser.GetPVHandler)
	ser.router.DELETE(SinglePVURL, ser.DeletePVHandler)
	ser.router.POST(PVStatusURL, ser.UpdatePVStatusHandler)

	ser.router.GET(AllPVCURL, ser.GetAllPVCsHandler)
	ser.router.POST(NamespacePVCURL, ser.AddPVCHandler)
	ser.router.GET(SinglePVCURL, ser.GetPVCHandler)
	ser.router.DELETE(SinglePVCURL, ser.DeletePVCHandler)
	ser.router.POST(PVCStatusURL, ser.UpdatePVCStatusHandler)
}

func (s *kubeApiServer) GetStatsDataHandler(c *gin.Context) {
	// 通过结构体来获取查询请求
	var query v1.MetricsQuery
	err := c.ShouldBindQuery(&query)
	if err != nil {
		c.JSON(http.StatusBadRequest,
			v1.BaseResponse[*v1.PodRawMetrics]{Error: "error in parsing query"},
		)
		return
	}
	// 通过query来获取数据
	fmt.Printf("get query: %v\n", query)
	fmt.Printf("get query uid: %v\n", query.UID)
	data, err := s.metrics_cli.GetPodMetrics(query.UID, query.TimeStamp, query.Window)

	if err != nil {
		fmt.Printf("error in getting metrics: %v\n", err)
		c.JSON(http.StatusInternalServerError,
			v1.BaseResponse[*v1.PodRawMetrics]{Error: fmt.Sprintf("error in getting metrics: %v", err)},
		)
		return
	}
	c.JSON(http.StatusOK,
		v1.BaseResponse[*v1.PodRawMetrics]{Data: data},
	)

}
func (s *kubeApiServer) AddStatsDataHandler(c *gin.Context) {
	// 获取需要保存的metrics
	// contentStr, _ := c.GetRawData()
	// fmt.Printf("get content str: %v\n", string(contentStr))

	var metrics []*v1.PodRawMetrics
	err := c.ShouldBind(&metrics)
	fmt.Printf("get metrics str: %v\n", metrics)
	if err != nil {
		log.Printf("error in parsing metrics: %v", err)
		c.JSON(http.StatusBadRequest,
			v1.BaseResponse[[]*v1.PodRawMetrics]{Error: "error in parsing metrics"},
		)
		return
	}
	// 保存metrics
	err = s.metrics_cli.SavePodMetrics(metrics)

	if err != nil {
		c.JSON(http.StatusInternalServerError,
			v1.BaseResponse[[]*v1.PodRawMetrics]{Error: fmt.Sprintf("error in saving metrics: %v", err)},
		)
		return
	}
	c.JSON(http.StatusOK,
		v1.BaseResponse[[]*v1.PodRawMetrics]{Data: metrics},
	)
}

func (s *kubeApiServer) GetAllScalingHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allSca, err := s.getAllScalingsFromEtcd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.HorizontalPodAutoscaler]{
			Error: fmt.Sprintf("error in getting all scalings: %v", err),
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.HorizontalPodAutoscaler]{Data: allSca})

}
func (s *kubeApiServer) getAllScalingsFromEtcd() ([]*v1.HorizontalPodAutoscaler, error) {
	allScaKeyPrefix := "/registry/scaling"
	res, err := s.store_cli.GetSubKeysValues(allScaKeyPrefix)
	if err != nil {
		return nil, err
	}
	allSca := make([]*v1.HorizontalPodAutoscaler, 0)
	for _, v := range res {
		var sca v1.HorizontalPodAutoscaler
		err = json.Unmarshal([]byte(v), &sca)
		if err != nil {
			return nil, err
		}
		allSca = append(allSca, &sca)
	}
	return allSca, nil
}

func (ser *kubeApiServer) AddScalingHandler(c *gin.Context) {
	ser.lock.Lock()
	defer ser.lock.Unlock()
	var hpa v1.HorizontalPodAutoscaler
	err := c.ShouldBind(&hpa)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in parsing hpa",
		})
		return
	}
	if hpa.Kind != string(v1.ScalerTypeHPA) {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "scaling type not supported",
		})
		return
	}
	if hpa.Name == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "hpa name is required",
		})
		return
	}
	if hpa.Namespace == "" {
		hpa.Namespace = Default_Namespace
	}
	hpa.CreationTimestamp = timestamp.NewTimestamp()
	hpa.UID = (v1.UID)(uuid.NewUUID())
	if hpa.Spec.MinReplicas == 0 {
		hpa.Spec.MinReplicas = 1
	}

	hpaKey := fmt.Sprintf("/registry/namespaces/%s/scaling/%s", hpa.Namespace, hpa.Name)
	allhpaKey := fmt.Sprintf("/registry/scaling/%s", hpa.UID)

	// 检查是否有重复的
	hpaUid, err := ser.store_cli.Get(hpaKey)
	if hpaUid != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "scaling already exists",
		})
		return
	}

	// 没有重复，可以添加

	hpaStr, err := json.Marshal(hpa)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in json marshal",
		})
		return
	}
	err = ser.store_cli.Set(hpaKey, string(hpa.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in writing scaling uid to etcd",
		})
		return
	}

	err = ser.store_cli.Set(allhpaKey, string(hpaStr))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in writing scaling to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{Data: &hpa})

}

func (ser *kubeApiServer) DeleteScalingHandler(c *gin.Context) {
	ser.lock.Lock()
	defer ser.lock.Unlock()
	namespace := c.Params.ByName("namespace")
	name := c.Params.ByName("name")
	if namespace == "" || name == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "namespace and name are required",
		})
		return
	}
	hpaKey := fmt.Sprintf("/registry/namespaces/%s/scaling/%s", namespace, name)
	hpaUid, err := ser.store_cli.Get(hpaKey)

	if hpaUid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "scaling not found",
		})
		return

	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in getting scaling uid from etcd",
		})
		return
	}
	err = ser.store_cli.Delete(hpaKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in deleting scaling from etcd",
		})
	}
	allhpaKey := fmt.Sprintf("/registry/scaling/%s", hpaUid)

	var hpa v1.HorizontalPodAutoscaler
	hpaStr, err := ser.store_cli.Get(allhpaKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in getting scaling from etcd",
		})
		return
	}
	err = json.Unmarshal([]byte(hpaStr), &hpa)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in json unmarshal",
		})
		return
	}

	err = ser.store_cli.Delete(allhpaKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{
			Error: "error in deleting scaling from etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.HorizontalPodAutoscaler]{Data: &hpa})

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
	ser.lock.Lock()
	defer ser.lock.Unlock()
	log.Println("GetAllPods")

	all_pod_str := make([]v1.Pod, 0)

	prefix := "/registry"

	all_pod_keystr := prefix + "/pods"

	res, err := ser.store_cli.GetSubKeysValues(all_pod_keystr)

	//if res == nil || len(res) == 0 || err != nil {
	//	log.Println("no pod exists")
	//	con.JSON(http.StatusNotFound, gin.H{
	//		"error": "no pod exists",
	//	})
	//	return
	//}
	if err != nil {
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in reading all pods from etcd",
		})
		return
	}
	if len(res) == 0 {
		con.JSON(http.StatusOK, v1.BaseResponse[[]*v1.Pod]{Data: nil})
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
	ser.lock.Lock()
	defer ser.lock.Unlock()
	log.Println("GetPodsByNamespace")

	np := con.Params.ByName("namespace")
	if np == "" {
		//log.Panicln("error in parsing namespace ")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "error in parsing namespace ",
		})
		return
	}

	all_pod_str := make([]v1.Pod, 0)

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods"

	res, err := ser.store_cli.GetSubKeysValues(namespace_pod_keystr)

	//if res == nil || len(res) == 0 || err != nil {
	//	log.Println("no pod exists")
	//	con.JSON(http.StatusNotFound, gin.H{
	//		"error": "no pod exists",
	//	})
	//	return
	//}
	if err != nil {
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in reading all pods from etcd",
		})
		return
	}
	if len(res) == 0 {
		con.JSON(http.StatusOK, v1.BaseResponse[[]*v1.Pod]{Data: nil})
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
	ser.lock.Lock()
	defer ser.lock.Unlock()
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
	ser.lock.Lock()
	defer ser.lock.Unlock()
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

	namespace := con.Param("namespace")
	if namespace == "" {
		con.JSON(http.StatusBadRequest, gin.H{
			"error": "namespace is required",
		})
		return
	}
	if pod.Namespace == "" {
		if namespace != Default_Namespace {
			con.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace does not match",
			})
			return
		}
	} else {
		if pod.Namespace != namespace {
			con.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace does not match",
			})
			return
		}
	}
	pod.Namespace = namespace

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
	namespace_pod_keystr := prefix + "/namespaces/" + namespace + "/pods/" + pod_name

	// node里面对应的也是podname和uid的映射
	// node_pod_keystr := prefix + "/nodes/" + Default_Nodename + "/pods/" + pod_name

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
	//err = ser.store_cli.Set(node_pod_keystr, string(pod.ObjectMeta.UID))
	//if err != nil {
	//	log.Println("error in writing to etcd")
	//	con.JSON(http.StatusInternalServerError, gin.H{
	//		"error": "error in writing to etcd",
	//	})
	//	return
	//}

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
	ser.lock.Lock()
	defer ser.lock.Unlock()
	log.Println("DeletePod")

	np := con.Params.ByName("namespace")
	pod_name := con.Params.ByName("podname")

	prefix := "/registry"

	namespace_pod_keystr := prefix + "/namespaces/" + np + "/pods/" + pod_name

	// node_pod_keystr := prefix + "/nodes/" + Default_Nodename + "/pods/" + pod_name

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

	//res, err = ser.store_cli.Get(node_pod_keystr)
	//// or change return type to DeleteResponse so there is no need to check Get result
	//if res == "" || err != nil {
	//	log.Println("pod name does not exist in node")
	//	con.JSON(http.StatusNotFound, gin.H{
	//		"error": "pod name does not exist in node",
	//	})
	//	return
	//}

	//err = ser.store_cli.Delete(node_pod_keystr)
	//if err != nil {
	//	log.Println("error in deleting from etcd")
	//	con.JSON(http.StatusInternalServerError, gin.H{
	//		"error": "error in deleting from etcd",
	//	})
	//	return
	//}

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
	ser.lock.Lock()
	defer ser.lock.Unlock()
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
	ser.lock.Lock()
	defer ser.lock.Unlock()
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
	ser.lock.Lock()
	defer ser.lock.Unlock()
	// first parse nodename
	log.Println("GetPodsByNode")

	node_name := con.Params.ByName("nodename")
	if node_name == "" {
		//log.Panicln("error in parsing nodename ")
		con.JSON(http.StatusNotFound, gin.H{
			"error": "error in parsing nodename ",
		})
		return
	}
	log.Printf("getting info of node: %v", node_name)

	all_pod_str := make([]v1.Pod, 0)

	prefix := "/registry"

	node_pod_keystr := prefix + "/host-nodes/" + node_name + "/pods"

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

	var keysToDelete []string
	//res返回的是pod的uid，回到etcd里面找到pod的信息
	for k, v := range res {
		pod_id := v
		all_pod_keystr := prefix + "/pods/" + pod_id

		res, err := ser.store_cli.Get(all_pod_keystr)
		//if res == "" || err != nil {
		//	log.Println("pod does not exist")
		//	con.JSON(http.StatusNotFound, gin.H{
		//		"error": "pod does not exist",
		//	})
		//	return
		//}
		if res == "" || err != nil {
			// lazy deleting
			keysToDelete = append(keysToDelete, k)
			continue
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

	for _, mapping := range keysToDelete {
		_ = ser.store_cli.Delete(mapping)
	}

	// then return all of them
	con.JSON(http.StatusOK,
		gin.H{
			"data": all_pod_str,
		},
	)
}

func (s *kubeApiServer) GetAllServicesHandler(c *gin.Context) {
	//allSvcKey := "/registry/services"
	//res, err := s.store_cli.GetSubKeysValues(allSvcKey)
	//if err != nil {
	//	c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
	//		Error: "error in reading from etcd",
	//	})
	//	return
	//}
	//services := make([]v1.Service, 0)
	//for _, v := range res {
	//	var service v1.Service
	//	err = json.Unmarshal([]byte(v), &service)
	//	if err != nil {
	//		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
	//			Error: "error in json unmarshal",
	//		})
	//		return
	//	}
	//	services = append(services, service)
	//}
	s.lock.Lock()
	defer s.lock.Unlock()
	services, err := s.getAllServicesFromEtcd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]v1.Service]{
			Error: fmt.Sprintf("error in reading all services from etcd: %v", err),
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.Service]{
		Data: services,
	})
}

func (s *kubeApiServer) getAllServicesFromEtcd() ([]*v1.Service, error) {
	allSvcKey := "/registry/services"
	res, err := s.store_cli.GetSubKeysValues(allSvcKey)
	if err != nil {
		return nil, err
	}
	services := make([]*v1.Service, 0)
	for _, v := range res {
		var service v1.Service
		err = json.Unmarshal([]byte(v), &service)
		if err != nil {
			return nil, err
		}
		services = append(services, &service)
	}
	return services, nil
}

func (s *kubeApiServer) checkTypeAndPorts(service *v1.Service) error {
	// 默认类型为ClusterIP
	if service.Spec.Type == "" {
		service.Spec.Type = v1.ServiceTypeClusterIP
	} else if service.Spec.Type != v1.ServiceTypeClusterIP && service.Spec.Type != v1.ServiceTypeNodePort {
		return fmt.Errorf("invalid service type %s", service.Spec.Type)
	}
	// 默认协议为TCP
	nodePortSet := make(map[int32]struct{})
	for i, port := range service.Spec.Ports {
		if port.Protocol == "" {
			service.Spec.Ports[i].Protocol = v1.ProtocolTCP
		}
		if port.TargetPort < v1.PortMin || port.TargetPort > v1.PortMax {
			return fmt.Errorf("invalid target port %d", port.TargetPort)
		}
		if port.Port < v1.PortMin || port.Port > v1.PortMax {
			return fmt.Errorf("invalid port %d", port.Port)
		}
		if service.Spec.Type == v1.ServiceTypeNodePort {
			if port.NodePort < v1.NodePortMin || port.NodePort > v1.NodePortMax {
				return fmt.Errorf("invalid node port %d", port.NodePort)
			}
			if _, ok := nodePortSet[port.NodePort]; ok {
				return fmt.Errorf("there are conflicting node ports: %v", port.NodePort)
			}
			nodePortSet[port.NodePort] = struct{}{}
		}
	}
	// 检查所有service node port是否有冲突
	if service.Spec.Type == v1.ServiceTypeNodePort {
		allServices, err := s.getAllServicesFromEtcd()
		if err != nil {
			return fmt.Errorf("error in reading all services from etcd: %v", err)
		}
		for _, svc := range allServices {
			if svc.Spec.Type != v1.ServiceTypeNodePort {
				continue
			}
			for _, port := range svc.Spec.Ports {
				if _, ok := nodePortSet[port.NodePort]; ok {
					return fmt.Errorf("node port %v conflicts with service %v", port.NodePort, svc.Name)
				}
			}
		}
	}
	return nil
}

func (s *kubeApiServer) AddServiceHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
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
	namespaceSvcKey := fmt.Sprintf("/registry/namespaces/%s/services/%s", namespace, service.Name)

	// 检查service是否已经存在
	result, err := s.store_cli.Get(namespaceSvcKey)
	if err == nil && result != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.Service]{
			Error: fmt.Sprintf("service %s/%s already exists", namespace, service.Name),
		})
		return
	}

	err = s.checkTypeAndPorts(&service)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: err.Error(),
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
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	serviceName := c.Param("servicename")
	if namespace == "" || serviceName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Service]{
			Error: "namespace and service name cannot be empty",
		})
		return
	}
	namespaceSvcKey := fmt.Sprintf("/registry/namespaces/%s/services/%s", namespace, serviceName)
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

func (s *kubeApiServer) getAllDNSFromEtcd() ([]*v1.DNS, error) {
	allDNSKey := "/registry/dns"
	res, err := s.store_cli.GetSubKeysValues(allDNSKey)
	if err != nil {
		return nil, err
	}
	dnsSlice := make([]*v1.DNS, 0)
	for _, v := range res {
		var dns v1.DNS
		err = json.Unmarshal([]byte(v), &dns)
		if err != nil {
			return nil, err
		}
		dnsSlice = append(dnsSlice, &dns)
	}
	return dnsSlice, nil
}

func (s *kubeApiServer) GetAllDNSHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allDNS, err := s.getAllDNSFromEtcd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.DNS]{
			Error: fmt.Sprintf("error in reading all dns from etcd: %v", err),
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.DNS]{
		Data: allDNS,
	})
}

func (s *kubeApiServer) validateDNS(dns *v1.DNS, urlNamespace string) error {
	if dns.Name == "" {
		return fmt.Errorf("dns name is required")
	}
	if dns.Namespace == "" {
		if urlNamespace != "default" {
			return fmt.Errorf("namespace mismatch, spec: empty(using default), url: %s", urlNamespace)
		}
	} else {
		if dns.Namespace != urlNamespace {
			return fmt.Errorf("namespace mismatch, spec: %s, url: %s", dns.Namespace, urlNamespace)
		}
	}
	if dns.Kind != "DNS" {
		return fmt.Errorf("invalid api object kind")
	}
	if len(dns.Spec.Rules) == 0 {
		return fmt.Errorf("no rules for this dns")
	}
	for _, rule := range dns.Spec.Rules {
		if rule.Host == "" {
			return fmt.Errorf("host cannot be empty")
		}
		for _, path := range rule.Paths {
			if path.Path == "" {
				return fmt.Errorf("path cannot be empty")
			}
			if path.Backend.Service.Name == "" {
				return fmt.Errorf("service name cannot be empty")
			}
			if path.Backend.Service.Port < v1.PortMin || path.Backend.Service.Port > v1.PortMax {
				return fmt.Errorf("invalid service port %d", path.Backend.Service.Port)
			}
		}
	}
	return nil
}

func (s *kubeApiServer) AddDNSHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var dns v1.DNS
	err := c.ShouldBind(&dns)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
			Error: "invalid dns json",
		})
		return
	}
	namespace := c.Param("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
			Error: "namespace is required",
		})
		return
	}

	// 参数校验
	err = s.validateDNS(&dns, namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
			Error: err.Error(),
		})
		return
	}
	dns.Namespace = namespace

	// 检查dns是否已经存在
	namespaceDNSKey := fmt.Sprintf("/registry/namespaces/%s/dns/%s", dns.Namespace, dns.Name)
	uid, err := s.store_cli.Get(namespaceDNSKey)
	if err == nil && uid != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.DNS]{
			Error: fmt.Sprintf("dns %s/%s already exists", dns.Namespace, dns.Name),
		})
		return
	}

	// 检查域名冲突
	hostKey := "/registry/hosts"
	hosts, err := s.store_cli.Get(hostKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in reading hosts from etcd",
		})
		return
	}
	hostMap := make(map[string]struct{})
	scanner := bufio.NewScanner(strings.NewReader(hosts))
	for scanner.Scan() {
		hostMap[scanner.Text()] = struct{}{}
	}
	for _, rule := range dns.Spec.Rules {
		if _, ok := hostMap[rule.Host]; ok {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
				Error: fmt.Sprintf("host %s conflicts with existing host", rule.Host),
			})
			return
		}
		hosts += rule.Host + "\n"
	}

	// 检查每个path的service backend是否存在
	for _, rule := range dns.Spec.Rules {
		for _, path := range rule.Paths {
			svcName := path.Backend.Service.Name
			namespaceSvcKey := fmt.Sprintf("/registry/namespaces/%s/services/%s", namespace, svcName)
			uid, err := s.store_cli.Get(namespaceSvcKey)
			if err != nil || uid == "" {
				c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
					Error: fmt.Sprintf("service %s/%s not found", namespace, svcName),
				})
				return
			}
			allSvcKey := fmt.Sprintf("/registry/services/%s", uid)
			svcJson, err := s.store_cli.Get(allSvcKey)
			if err != nil || svcJson == "" {
				c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
					Error: "error in reading service from etcd",
				})
				return
			}
			var svc v1.Service
			err = json.Unmarshal([]byte(svcJson), &svc)
			if err != nil {
				c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
					Error: "error in json unmarshal",
				})
				return
			}
			isPortMatched := false
			for _, port := range svc.Spec.Ports {
				if port.Port == path.Backend.Service.Port {
					isPortMatched = true
					break
				}
			}
			if !isPortMatched {
				c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
					Error: fmt.Sprintf("service %s does not have port %d", svcName, path.Backend.Service.Port),
				})
				return
			}
		}
	}

	// now create it!
	dns.CreationTimestamp = timestamp.NewTimestamp()
	dns.UID = v1.UID(uuid.NewUUID())

	// 存hosts
	err = s.store_cli.Set(hostKey, hosts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in writing hosts to etcd",
		})
		return
	}

	// 存uuid
	err = s.store_cli.Set(namespaceDNSKey, string(dns.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in writing to etcd",
		})
		return
	}

	// 存dns json
	allDNSKey := fmt.Sprintf("/registry/dns/%s", dns.UID)
	dnsJson, err := json.Marshal(dns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in json marshal",
		})
		return
	}
	err = s.store_cli.Set(allDNSKey, string(dnsJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in writing to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.DNS]{
		Data: &dns,
	})
}

func (s *kubeApiServer) DeleteDNSHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	dnsName := c.Param("dnsname")
	if namespace == "" || dnsName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.DNS]{
			Error: "namespace and dns name cannot be empty",
		})
		return
	}
	namespaceDNSKey := fmt.Sprintf("/registry/namespaces/%s/dns/%s", namespace, dnsName)
	uid, err := s.store_cli.Get(namespaceDNSKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.DNS]{
			Error: fmt.Sprintf("dns %s/%s not found", namespace, dnsName),
		})
		return
	}

	allDNSKey := fmt.Sprintf("/registry/dns/%s", uid)
	dnsJson, err := s.store_cli.Get(allDNSKey)
	if err != nil || dnsJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in reading dns from etcd",
		})
		return
	}

	var dns v1.DNS
	err = json.Unmarshal([]byte(dnsJson), &dns)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in json unmarshal",
		})
		return
	}

	// 删除host
	hostKey := "/registry/hosts"
	hosts, err := s.store_cli.Get(hostKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in reading hosts from etcd",
		})
		return
	}
	hostsToDelete := make(map[string]struct{})
	for _, rule := range dns.Spec.Rules {
		hostsToDelete[rule.Host] = struct{}{}
	}
	scanner := bufio.NewScanner(strings.NewReader(hosts))
	newHosts := ""
	for scanner.Scan() {
		host := scanner.Text()
		if _, ok := hostsToDelete[host]; !ok {
			newHosts += host + "\n"
		}
	}
	err = s.store_cli.Set(hostKey, newHosts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in writing hosts to etcd",
		})
		return
	}

	// 删除dns
	err = s.store_cli.Delete(namespaceDNSKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in deleting dns from etcd",
		})
		return
	}
	err = s.store_cli.Delete(allDNSKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.DNS]{
			Error: "error in deleting dns from etcd",
		})
		return
	}

	c.JSON(http.StatusOK, v1.BaseResponse[*v1.DNS]{
		Data: &dns,
	})
}

func (s *kubeApiServer) RegisterNodeHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	address := c.Query("address")
	if net.ParseIP(address) == nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Node]{
			Error: "invalid ip address",
		})
	}
	bitmapKey := "/registry/NodePool/bitmap"
	bitmapStr, err := s.store_cli.Get(bitmapKey)
	bitmap := []byte(bitmapStr)
	if err != nil || bitmapStr == "" {
		initialBitmap := make([]byte, utils.NodePoolSize/8)
		bitmap = initialBitmap
		err = s.store_cli.Set(bitmapKey, string(initialBitmap))
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
				Error: "error in writing bitmap to etcd",
			})
			return
		}
	}
	nodeName, err := utils.AllocNode(bitmap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: err.Error(),
		})
		return
	}
	err = s.store_cli.Set(bitmapKey, string(bitmap))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Service]{
			Error: "error in writing bitmap to etcd",
		})
		return
	}
	node := &v1.Node{
		TypeMeta: v1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:              nodeName,
			Namespace:         Default_Namespace,
			UID:               v1.UID(uuid.NewUUID()),
			CreationTimestamp: timestamp.NewTimestamp(),
		},
		Status: v1.NodeStatus{
			Address: address,
		},
	}
	allNodeKey := fmt.Sprintf("/registry/nodes/%v", node.UID)
	namespaceNodeKey := fmt.Sprintf("/registry/namespaces/%v/nodes/%v", node.Namespace, node.Name)
	nodeJson, err := json.Marshal(node)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in json marshal",
		})
		return
	}
	err = s.store_cli.Set(allNodeKey, string(nodeJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in writing node to etcd",
		})
		return
	}
	err = s.store_cli.Set(namespaceNodeKey, string(node.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in writing node to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.Node]{
		Data: node,
	})
}

func (s *kubeApiServer) UnregisterNodeHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	nodeName := c.Query("nodename")
	namespace := Default_Namespace
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Node]{
			Error: "node name or namespace cannot be empty",
		})
		return
	}
	namespaceNodeKey := fmt.Sprintf("/registry/namespaces/%s/nodes/%s", namespace, nodeName)
	uid, err := s.store_cli.Get(namespaceNodeKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.Node]{
			Error: fmt.Sprintf("node %s/%s not found", namespace, nodeName),
		})
		return
	}
	// 把所有pod变为unscheduled
	nodePodKey := fmt.Sprintf("/registry/host-nodes/%s/pods", nodeName)
	err = s.store_cli.DeleteSubKeys(nodePodKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in deleting node-pod mappings from etcd",
		})
		return
	}
	// 删除pod
	err = s.store_cli.Delete(namespaceNodeKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in deleting node from etcd",
		})
		return
	}
	allNodeKey := fmt.Sprintf("/registry/nodes/%s", uid)
	err = s.store_cli.Delete(allNodeKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in deleting node from etcd",
		})
		return
	}
	// 释放node
	bitmapKey := "/registry/NodePool/bitmap"
	bitmapStr, err := s.store_cli.Get(bitmapKey)
	if err != nil || bitmapStr == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in reading node pool bitmap from etcd",
		})
		return
	}
	bitmap := []byte(bitmapStr)
	err = utils.FreeNode(nodeName, bitmap)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: err.Error(),
		})
		return
	}
	err = s.store_cli.Set(bitmapKey, string(bitmap))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in writing node pool bitmap to etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.Node]{})
}

func (s *kubeApiServer) GetAllNodesHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allNodeKey := "/registry/nodes"
	res, err := s.store_cli.GetSubKeysValues(allNodeKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
			Error: "error in reading from etcd",
		})
		return
	}
	nodes := make([]*v1.Node, 0)
	for _, v := range res {
		var node v1.Node
		err = json.Unmarshal([]byte(v), &node)
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Node]{
				Error: "error in json unmarshal",
			})
			return
		}
		nodes = append(nodes, &node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Name < nodes[j].Name
	})
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.Node]{
		Data: nodes,
	})
}

func (s *kubeApiServer) SchedulePodToNodeHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	podUid := c.Query("podUid")
	nodeName := c.Query("nodename")
	if podUid == "" || nodeName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[interface{}]{
			Error: "pod uid or node name cannot be empty",
		})
		return
	}
	podKey := fmt.Sprintf("/registry/pods/%s", podUid)
	podJson, err := s.store_cli.Get(podKey)
	if err != nil || podJson == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[interface{}]{
			Error: fmt.Sprintf("pod %s not found", podUid),
		})
		return
	}

	var pod v1.Pod
	err = json.Unmarshal([]byte(podJson), &pod)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[interface{}]{
			Error: "error in json unmarshal",
		})
		return
	}

	res, err := s.store_cli.GetSubKeysValues("/registry/host-nodes")
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[interface{}]{
			Error: "error in reading from etcd",
		})
		return
	}
	for k, v := range res {
		if v == podUid {
			err = s.store_cli.Delete(k)
			if err != nil {
				c.JSON(http.StatusInternalServerError, v1.BaseResponse[interface{}]{
					Error: "error in deleting old node-pod mapping from etcd",
				})
				return
			}
		}
	}

	nodePodKey := fmt.Sprintf("/registry/host-nodes/%s/pods/%s_%s", nodeName, pod.Namespace, pod.Name)
	err = s.store_cli.Set(nodePodKey, podUid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[interface{}]{
			Error: "error in writing new node-pod mapping to etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[interface{}]{})
}

func (s *kubeApiServer) GetUnscheduledPodHandler(c *gin.Context) {
	// 获取所有pod
	s.lock.Lock()
	defer s.lock.Unlock()
	allPodKey := "/registry/pods"
	res, err := s.store_cli.GetSubKeysValues(allPodKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.Pod]{
			Error: "error in reading from etcd",
		})
		return
	}
	// 通过/registry/host-nodes获取所有已被调度的pod uid
	hostKeyPrefix := "/registry/host-nodes"
	hostRes, err := s.store_cli.GetSubKeysValues(hostKeyPrefix)
	scheduledUidSet := make(map[string]struct{})
	for _, uid := range hostRes {
		scheduledUidSet[uid] = struct{}{}
	}
	var unscheduledPods []*v1.Pod
	for _, v := range res {
		var pod v1.Pod
		err = json.Unmarshal([]byte(v), &pod)
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.Pod]{
				Error: "error in json unmarshal",
			})
			return
		}
		if _, ok := scheduledUidSet[string(pod.UID)]; !ok {
			unscheduledPods = append(unscheduledPods, &pod)
		}
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.Pod]{
		Data: unscheduledPods,
	})
}

func (ser *kubeApiServer) GetAllReplicaSetsHandler(con *gin.Context) {
	ser.lock.Lock()
	defer ser.lock.Unlock()
	log.Println("GetAllReplicaSets")
	all_replicaset_str := make([]v1.ReplicaSet, 0)
	prefix := "/registry"
	all_replicaset_keystr := prefix + "/replicaset"

	res, err := ser.store_cli.GetSubKeysValues(all_replicaset_keystr)
	if err != nil {
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in reading all replicas sets in etcd",
		})
		return
	}

	//namespace_replicaset_keystr := prefix + "/namespace/" + Default_Namespace + "/replicasets/"
	//
	//res2, err2 := ser.store_cli.Get(namespace_replicaset_keystr)
	//if err2 != nil {
	//	log.Println("replica set name already exists")
	//	con.JSON(http.StatusConflict, gin.H{
	//		"error": "replica set name already exists",
	//	})
	//	return
	//}
	//for k, v := range res2 {
	//	println("-----------------------------------")
	//	println(k)
	//	println("~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
	//	println(v)
	//
	//}

	if len(res) == 0 {
		con.JSON(http.StatusOK, v1.BaseResponse[[]*v1.ReplicaSet]{Data: nil})
		return
	}

	for _, v := range res {
		var rps v1.ReplicaSet
		err = json.Unmarshal([]byte(v), &rps)
		if err != nil {
			log.Println("error in json unmarshal")
			con.JSON(http.StatusInternalServerError, gin.H{
				"error": "error in json unmarshal",
			})
			return
		}
		println(rps.UID)
		all_replicaset_str = append(all_replicaset_str, rps)
	}

	con.JSON(http.StatusOK,
		gin.H{
			"data": all_replicaset_str,
		},
	)
}

func (ser *kubeApiServer) AddReplicaSetHandler(con *gin.Context) {
	ser.lock.Lock()
	defer ser.lock.Unlock()
	log.Println("Adding a new replica set")
	var rps v1.ReplicaSet
	err := con.ShouldBind(&rps)
	if err != nil {
		log.Panicln("something is wrong when parsing replica set")
		return
	}
	rps_name := rps.ObjectMeta.Name
	if rps_name == "" {
		rps_name = Default_Podname
	}

	rps_label := rps.Spec.Selector.MatchLabels
	if rps_label == nil {
		con.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "replica set labels are required",
		})
		return
	}

	//rps_template := rps.Template
	//if rps_template == empty {
	//	con.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.ReplicaSet]{
	//		Error: "replica set template is required",
	//	})
	//	return
	//}

	rps.ObjectMeta.UID = (v1.UID)(uuid.NewUUID())
	rps.ObjectMeta.CreationTimestamp = timestamp.NewTimestamp()
	prefix := "/registry"

	all_replicaset_keystr := prefix + "/replicaset/" + string(rps.ObjectMeta.UID)
	namespace_replicaset_keystr := prefix + "/namespaces/" + Default_Namespace + "/replicasets/" + rps_name

	res, err := ser.store_cli.Get(namespace_replicaset_keystr)
	if res != "" || err != nil {
		log.Println("replica set name already exists")
		con.JSON(http.StatusConflict, gin.H{
			"error": "replica set name already exists",
		})
		return
	}

	err = ser.store_cli.Set(namespace_replicaset_keystr, string(rps.ObjectMeta.UID))
	if err != nil {
		log.Println("error in writing to etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in writing to etcd",
		})
		return
	}

	rps_str, err := json.Marshal(rps)

	if err != nil {
		log.Println("error in json marshal")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in json marshal",
		})
		return
	}

	err = ser.store_cli.Set(all_replicaset_keystr, string(rps_str))
	if err != nil {
		log.Println("error in writing to etcd")
		con.JSON(http.StatusInternalServerError, gin.H{
			"error": "error in writing to etcd",
		})
		return
	}

	con.JSON(http.StatusCreated, gin.H{
		"message": "successfully created replica set",
		"UUID":    rps.ObjectMeta.UID,
	})

}

func (s *kubeApiServer) GetReplicaSetHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	rpsName := c.Param("replicasetname")
	if namespace == "" || rpsName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "namespace and replica set name cannot be empty",
		})
		return
	}
	namespaceRpsKey := fmt.Sprintf("/registry/namespaces/%s/replicasets/%s", namespace, rpsName)
	uid, err := s.store_cli.Get(namespaceRpsKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.ReplicaSet]{
			Error: fmt.Sprintf("replica set %s/%s not found", namespace, rpsName),
		})
		return
	}

	allRpsKey := fmt.Sprintf("/registry/replicaset/%s", uid)
	rpsJson, err := s.store_cli.Get(allRpsKey)
	if err != nil || rpsJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in reading replica set from etcd",
		})
		return
	}
	var rps v1.ReplicaSet
	err = json.Unmarshal([]byte(rpsJson), &rps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in json unmarshal",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.ReplicaSet]{
		Data: &rps,
	})

}

func (s *kubeApiServer) UpdateReplicaSetHandler(c *gin.Context) {
	// 目前的更新方式
	// 1.更新replica set的replicas数量
	s.lock.Lock()
	defer s.lock.Unlock()
	// 获取name . namespace 和待更新的数量int
	namespace := c.Param("namespace")
	rpsName := c.Param("replicasetname")
	if namespace == "" || rpsName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "namespace and replica set name cannot be empty",
		})
		return
	}
	//  PUT方法获取int
	replicas, err := strconv.Atoi(c.Query("replicas"))

	fmt.Printf("next replicas: %d\n", replicas)

	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "replicas must be an integer",
		})
		return
	}

	// 从etcd中获取replica set
	namespaceRpsKey := fmt.Sprintf("/registry/namespaces/%s/replicasets/%s", namespace, rpsName)
	uid, err := s.store_cli.Get(namespaceRpsKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.ReplicaSet]{
			Error: fmt.Sprintf("replica set %s/%s not found", namespace, rpsName),
		})
		return
	}

	allRpsKey := fmt.Sprintf("/registry/replicaset/%s", uid)
	rpsJson, err := s.store_cli.Get(allRpsKey)
	if err != nil || rpsJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in reading replica set from etcd",
		})
		return
	}
	var rps v1.ReplicaSet
	err = json.Unmarshal([]byte(rpsJson), &rps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in json unmarshal",
		})
		return
	}

	// 更新replicas数量
	rps.Spec.Replicas = (int32)(replicas)

	rpsRawStr, err := json.Marshal(rps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in json marshal",
		})
		return
	}
	// 存回
	err = s.store_cli.Set(allRpsKey, string(rpsRawStr))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in writing replica set to etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.ReplicaSet]{
		Data: &rps,
	})
}

func (ser *kubeApiServer) DeleteReplicaSetHandler(con *gin.Context) {
	ser.lock.Lock()
	defer ser.lock.Unlock()
	namespace := con.Param("namespace")
	rpsName := con.Param("replicasetname")
	if namespace == "" || rpsName == "" {
		con.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "namespace and replica set name cannot be empty",
		})
		return
	}
	namespaceRpsKey := fmt.Sprintf("/registry/namespaces/%s/replicasets/%s", Default_Namespace, rpsName)
	uid, err := ser.store_cli.Get(namespaceRpsKey)
	if err != nil || uid == "" {
		con.JSON(http.StatusNotFound, v1.BaseResponse[*v1.ReplicaSet]{
			Error: fmt.Sprintf("replica set %s/%s not found", namespace, rpsName),
		})
		return
	}

	allRpsKey := fmt.Sprintf("/registry/replicaset/%s", uid)
	rpsJson, err := ser.store_cli.Get(allRpsKey)
	if err != nil || rpsJson == "" {
		con.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in reading replica set from etcd",
		})
		return
	}
	var rps v1.ReplicaSet
	err = json.Unmarshal([]byte(rpsJson), &rps)
	if err != nil {
		con.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in json unmarshal",
		})
		return
	}

	err = ser.store_cli.Delete(namespaceRpsKey)
	if err != nil {
		con.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in deleting replica set from etcd",
		})
		return
	}
	err = ser.store_cli.Delete(allRpsKey)
	if err != nil {
		con.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.ReplicaSet]{
			Error: "error in deleting replica set from etcd",
		})
		return
	}
	con.JSON(http.StatusOK, v1.BaseResponse[*v1.ReplicaSet]{
		Data: &rps,
	})
}

// // 持久化scaled Podname && PodUID
// func (s *kubeApiServer) storeScaledPod(namespace, deploymentName, podName, podUID string) error {

// }

func (s *kubeApiServer) validateVirtualService(vs *v1.VirtualService, urlNamespace string) error {
	if vs.Name == "" {
		return fmt.Errorf("virtual service name is required")
	}
	if vs.Namespace == "" {
		if urlNamespace != "default" {
			return fmt.Errorf("namespace mismatch, spec: empty(using default), url: %s", urlNamespace)
		}
	} else {
		if vs.Namespace != urlNamespace {
			return fmt.Errorf("namespace mismatch, spec: %s, url: %s", vs.Namespace, urlNamespace)
		}
	}
	if vs.Kind != "VirtualService" {
		return fmt.Errorf("invalid api object kind")
	}
	return nil
}

func (s *kubeApiServer) AddVirtualServiceHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var vs v1.VirtualService
	err := c.ShouldBind(&vs)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
			Error: "invalid virtual service json",
		})
		return
	}
	namespace := c.Param("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
			Error: "namespace is required",
		})
		return
	}
	err = s.validateVirtualService(&vs, namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
			Error: err.Error(),
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/virtualservices/%s", namespace, vs.Name)
	uid, err := s.store_cli.Get(namespaceKey)
	if err == nil && uid != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.VirtualService]{
			Error: fmt.Sprintf("virtual service %s/%s already exists", namespace, vs.Name),
		})
		return
	}

	svcName := vs.Spec.ServiceRef
	svcUID, err := s.store_cli.Get(fmt.Sprintf("/registry/namespaces/%s/services/%s", namespace, svcName))
	if err != nil || svcUID == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
			Error: fmt.Sprintf("service %s/%s not found", namespace, svcName),
		})
		return
	}
	svcJson, err := s.store_cli.Get(fmt.Sprintf("/registry/services/%s", svcUID))
	if err != nil || svcJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.VirtualService]{
			Error: "error in reading service from etcd",
		})
		return
	}
	var svc v1.Service
	_ = json.Unmarshal([]byte(svcJson), &svc)
	isPortMatched := false
	for _, port := range svc.Spec.Ports {
		if port.Port == vs.Spec.Port && port.Protocol == v1.ProtocolTCP {
			isPortMatched = true
			break
		}
	}
	if !isPortMatched {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
			Error: fmt.Sprintf("service %s does not have tcp port %d", svcName, vs.Spec.Port),
		})
		return
	}

	for _, subset := range vs.Spec.Subsets {
		subsetUID, err := s.store_cli.Get(fmt.Sprintf("/registry/namespaces/%s/subsets/%s", namespace, subset.Name))
		if err != nil || subsetUID == "" {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
				Error: fmt.Sprintf("subset %s/%s not found", namespace, subset.Name),
			})
			return
		}
		if subset.URL == nil && subset.Weight == nil {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
				Error: "subset url and weight cannot be both empty",
			})
			return
		}
	}

	vs.Namespace = namespace
	vs.CreationTimestamp = timestamp.NewTimestamp()
	vs.UID = v1.UID(uuid.NewUUID())
	allKey := fmt.Sprintf("/registry/virtualservices/%s", vs.UID)
	vsJson, _ := json.Marshal(vs)
	err = s.store_cli.Set(namespaceKey, string(vs.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.VirtualService]{
			Error: "error in writing virtual service to etcd",
		})
		return
	}
	err = s.store_cli.Set(allKey, string(vsJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.VirtualService]{
			Error: "error in writing virtual service to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.VirtualService]{
		Data: &vs,
	})
}

func (s *kubeApiServer) DeleteVirtualServiceHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	vsName := c.Param("virtualservicename")
	if namespace == "" || vsName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.VirtualService]{
			Error: "namespace and virtual service name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/virtualservices/%s", namespace, vsName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.VirtualService]{
			Error: fmt.Sprintf("virtual service %s/%s not found", namespace, vsName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/virtualservices/%s", uid)
	vsJson, _ := s.store_cli.Get(allKey)
	var vs v1.VirtualService
	_ = json.Unmarshal([]byte(vsJson), &vs)
	err = s.store_cli.Delete(namespaceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.VirtualService]{
			Error: "error in deleting virtual service from etcd",
		})
		return
	}
	err = s.store_cli.Delete(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.VirtualService]{
			Error: "error in deleting virtual service from etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.VirtualService]{
		Data: &vs,
	})
}

func (s *kubeApiServer) GetAllVirtualServicesHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allKey := "/registry/virtualservices"
	res, err := s.store_cli.GetSubKeysValues(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.VirtualService]{
			Error: "error in reading from etcd",
		})
		return
	}
	vss := make([]*v1.VirtualService, 0)
	for _, v := range res {
		var vs v1.VirtualService
		err = json.Unmarshal([]byte(v), &vs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.VirtualService]{
				Error: "error in json unmarshal",
			})
			return
		}
		vss = append(vss, &vs)
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.VirtualService]{
		Data: vss,
	})
}

func (s *kubeApiServer) validateSubset(subset *v1.Subset, namespace string) error {
	if subset.Name == "" {
		return fmt.Errorf("subset name is required")
	}
	if subset.Namespace == "" {
		if namespace != "default" {
			return fmt.Errorf("namespace mismatch, spec: empty(using default), url: %s", namespace)
		}
	} else {
		if subset.Namespace != namespace {
			return fmt.Errorf("namespace mismatch, spec: %s, url: %s", subset.Namespace, namespace)
		}
	}
	if subset.Kind != "Subset" {
		return fmt.Errorf("invalid api object kind")
	}
	return nil
}

func (s *kubeApiServer) AddSubsetHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var subset v1.Subset
	err := c.ShouldBind(&subset)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Subset]{
			Error: "invalid subset json",
		})
		return
	}
	namespace := c.Param("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Subset]{
			Error: "namespace is required",
		})
		return
	}
	err = s.validateSubset(&subset, namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Subset]{
			Error: err.Error(),
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/subsets/%s", namespace, subset.Name)
	isUpdate := false
	var oldUID v1.UID
	uid, err := s.store_cli.Get(namespaceKey)
	if err == nil && uid != "" {
		//c.JSON(http.StatusConflict, v1.BaseResponse[*v1.Subset]{
		//	Error: fmt.Sprintf("subset %s/%s already exists", namespace, subset.Name),
		//})
		//return
		isUpdate = true
		oldUID = v1.UID(uid)
	}

	for _, podName := range subset.Spec.Pods {
		podUID, err := s.store_cli.Get(fmt.Sprintf("/registry/namespaces/%s/pods/%s", namespace, podName))
		if err != nil || podUID == "" {
			c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Subset]{
				Error: fmt.Sprintf("pod %s/%s not found", namespace, podName),
			})
			return
		}
	}

	subset.Namespace = namespace
	subset.CreationTimestamp = timestamp.NewTimestamp()

	if isUpdate {
		subset.UID = oldUID
	} else {
		subset.UID = v1.UID(uuid.NewUUID())
	}

	allKey := fmt.Sprintf("/registry/subsets/%s", subset.UID)
	subsetJson, _ := json.Marshal(subset)
	err = s.store_cli.Set(namespaceKey, string(subset.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Subset]{
			Error: "error in writing subset to etcd",
		})
		return
	}
	err = s.store_cli.Set(allKey, string(subsetJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Subset]{
			Error: "error in writing subset to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.Subset]{
		Data: &subset,
	})
}

func (s *kubeApiServer) GetAllSubsetsHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allKey := "/registry/subsets"
	res, err := s.store_cli.GetSubKeysValues(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.Subset]{
			Error: "error in reading from etcd",
		})
		return
	}
	subsets := make([]*v1.Subset, 0)
	for _, v := range res {
		var subset v1.Subset
		err = json.Unmarshal([]byte(v), &subset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.Subset]{
				Error: "error in json unmarshal",
			})
			return
		}
		subsets = append(subsets, &subset)
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.Subset]{
		Data: subsets,
	})
}

func (s *kubeApiServer) GetSubsetHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	subsetName := c.Param("subsetname")
	if namespace == "" || subsetName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Subset]{
			Error: "namespace and subset name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/subsets/%s", namespace, subsetName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.Subset]{
			Error: fmt.Sprintf("subset %s/%s not found", namespace, subsetName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/subsets/%s", uid)
	subsetJson, _ := s.store_cli.Get(allKey)
	var subset v1.Subset
	_ = json.Unmarshal([]byte(subsetJson), &subset)
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.Subset]{
		Data: &subset,
	})
}

func (s *kubeApiServer) DeleteSubsetHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	subsetName := c.Param("subsetname")
	if namespace == "" || subsetName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.Subset]{
			Error: "namespace and subset name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/subsets/%s", namespace, subsetName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.Subset]{
			Error: fmt.Sprintf("subset %s/%s not found", namespace, subsetName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/subsets/%s", uid)
	subsetJson, _ := s.store_cli.Get(allKey)
	var subset v1.Subset
	_ = json.Unmarshal([]byte(subsetJson), &subset)
	err = s.store_cli.Delete(namespaceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Subset]{
			Error: "error in deleting subset from etcd",
		})
		return
	}
	err = s.store_cli.Delete(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.Subset]{
			Error: "error in deleting subset from etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.Subset]{
		Data: &subset,
	})
}

func (s *kubeApiServer) SaveSidecarMapping(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var mapping v1.SidecarMapping
	err := c.ShouldBind(&mapping)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.SidecarMapping]{
			Error: "invalid sidecar mapping json",
		})
		return
	}
	key := "/registry/sidecar-mapping"
	mappingJson, err := json.Marshal(mapping)
	err = s.store_cli.Set(key, string(mappingJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.SidecarMapping]{
			Error: "error in writing sidecar mapping to etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.SidecarMapping]{})
}

func (s *kubeApiServer) GetSidecarMapping(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	key := "/registry/sidecar-mapping"
	mappingJson, err := s.store_cli.Get(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.SidecarMapping]{
			Error: "error in reading sidecar mapping from etcd",
		})
		return
	}
	if mappingJson == "" {
		mappingJson = "{}"
	}
	var mapping v1.SidecarMapping
	err = json.Unmarshal([]byte(mappingJson), &mapping)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.SidecarMapping]{
			Error: "error in json unmarshal",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.SidecarMapping]{
		Data: &mapping,
	})
}

func (s *kubeApiServer) GetSidecarServiceNameMapping(c *gin.Context) {
	services, err := s.getAllServicesFromEtcd()
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[v1.SidecarServiceNameMapping]{
			Error: err.Error(),
		})
		return
	}
	mapping := make(v1.SidecarServiceNameMapping)
	for _, svc := range services {
		mapping[svc.Name] = svc.Spec.ClusterIP
	}
	c.JSON(http.StatusOK, v1.BaseResponse[v1.SidecarServiceNameMapping]{
		Data: mapping,
	})
}

func (s *kubeApiServer) validatePV(pv *v1.PersistentVolume, urlNamespace string) error {
	if pv.Name == "" {
		return fmt.Errorf("persistent volume name is required")
	}
	if pv.Namespace == "" {
		if urlNamespace != "default" {
			return fmt.Errorf("namespace mismatch, spec: empty(using default), url: %s", urlNamespace)
		}
	} else {
		if pv.Namespace != urlNamespace {
			return fmt.Errorf("namespace mismatch, spec: %s, url: %s", pv.Namespace, urlNamespace)
		}
	}
	if pv.Kind != "PersistentVolume" {
		return fmt.Errorf("invalid api object kind")
	}
	if pv.Spec.Capacity == "" {
		return fmt.Errorf("capacity is required")
	}
	if !strings.HasSuffix(pv.Spec.Capacity, "Gi") && !strings.HasSuffix(pv.Spec.Capacity, "Mi") && !strings.HasSuffix(pv.Spec.Capacity, "Ki") {
		return fmt.Errorf("invalid capacity format")
	}
	//var err error = nil
	//if strings.HasSuffix(pv.Spec.Capacity, "Gi") {
	//	_, err = strconv.Atoi(strings.TrimSuffix(pv.Spec.Capacity, "Gi"))
	//} else if strings.HasSuffix(pv.Spec.Capacity, "Mi") {
	//	_, err = strconv.Atoi(strings.TrimSuffix(pv.Spec.Capacity, "Mi"))
	//} else {
	//	_, err = strconv.Atoi(strings.TrimSuffix(pv.Spec.Capacity, "Ki"))
	//}
	//if err != nil {
	//	return fmt.Errorf("invalid capacity format")
	//}
	//if pv.Spec.NFS == nil {
	//	return fmt.Errorf("nfs spec is required")
	//}
	if _, err := kubeutils.ParseSize(pv.Spec.Capacity); err != nil {
		return fmt.Errorf("invalid capacity format")
	}
	return nil
}

func (s *kubeApiServer) AddPVHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var pv v1.PersistentVolume
	err := c.ShouldBind(&pv)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "invalid persistent volume json",
		})
		return
	}
	namespace := c.Param("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "namespace is required",
		})
	}
	err = s.validatePV(&pv, namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: err.Error(),
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumes/%s", namespace, pv.Name)
	uid, err := s.store_cli.Get(namespaceKey)
	if err == nil && uid != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.PersistentVolume]{
			Error: fmt.Sprintf("persistent volume %s/%s already exists", namespace, pv.Name),
		})
		return
	}
	pv.Namespace = namespace
	pv.CreationTimestamp = timestamp.NewTimestamp()
	pv.UID = v1.UID(uuid.NewUUID())
	pv.Status.Phase = v1.VolumePending
	allKey := fmt.Sprintf("/registry/persistentvolumes/%s", pv.UID)
	pvJson, _ := json.Marshal(pv)
	err = s.store_cli.Set(namespaceKey, string(pv.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "error in writing persistent volume to etcd",
		})
		return
	}
	err = s.store_cli.Set(allKey, string(pvJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "error in writing persistent volume to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.PersistentVolume]{
		Data: &pv,
	})
}

func (s *kubeApiServer) UpdatePVStatusHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	pvName := c.Param("pvname")
	if namespace == "" || pvName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "namespace and persistent volume name cannot be empty",
		})
		return
	}
	var status v1.PersistentVolumeStatus
	err := c.ShouldBind(&status)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "invalid persistent volume status json",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumes/%s", namespace, pvName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.PersistentVolume]{
			Error: fmt.Sprintf("persistent volume %s/%s not found", namespace, pvName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/persistentvolumes/%s", uid)
	pvJson, err := s.store_cli.Get(allKey)
	if err != nil || pvJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "error in reading persistent volume from etcd",
		})
		return
	}
	var pv v1.PersistentVolume
	_ = json.Unmarshal([]byte(pvJson), &pv)
	pv.Status = status
	newPVJson, _ := json.Marshal(pv)
	err = s.store_cli.Set(allKey, string(newPVJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "error in writing persistent volume to etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.PersistentVolume]{
		Data: &pv,
	})
}

func (s *kubeApiServer) GetAllPVsHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allKey := "/registry/persistentvolumes"
	res, err := s.store_cli.GetSubKeysValues(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.PersistentVolume]{
			Error: "error in reading from etcd",
		})
		return
	}
	pvs := make([]*v1.PersistentVolume, 0)
	for _, v := range res {
		var pv v1.PersistentVolume
		_ = json.Unmarshal([]byte(v), &pv)
		pvs = append(pvs, &pv)
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.PersistentVolume]{
		Data: pvs,
	})
}

func (s *kubeApiServer) DeletePVHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	pvName := c.Param("pvname")
	if namespace == "" || pvName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "namespace and persistent volume name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumes/%s", namespace, pvName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.PersistentVolume]{
			Error: fmt.Sprintf("persistent volume %s/%s not found", namespace, pvName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/persistentvolumes/%s", uid)
	pvJson, _ := s.store_cli.Get(allKey)
	var pv v1.PersistentVolume
	_ = json.Unmarshal([]byte(pvJson), &pv)
	err = s.store_cli.Delete(namespaceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "error in deleting persistent volume from etcd",
		})
		return
	}
	err = s.store_cli.Delete(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "error in deleting persistent volume from etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.PersistentVolume]{
		Data: &pv,
	})
}

func (s *kubeApiServer) validatePVC(pvc *v1.PersistentVolumeClaim, urlNamespace string) error {
	if pvc.Name == "" {
		return fmt.Errorf("persistent volume claim name is required")
	}
	if pvc.Namespace == "" {
		if urlNamespace != "default" {
			return fmt.Errorf("namespace mismatch, spec: empty(using default), url: %s", urlNamespace)
		}
	} else {
		if pvc.Namespace != urlNamespace {
			return fmt.Errorf("namespace mismatch, spec: %s, url: %s", pvc.Namespace, urlNamespace)
		}
	}
	if pvc.Kind != "PersistentVolumeClaim" {
		return fmt.Errorf("invalid api object kind")
	}
	if _, err := kubeutils.ParseSize(pvc.Spec.Request); err != nil {
		return fmt.Errorf("invalid pvc request")
	}
	return nil
}

func (s *kubeApiServer) AddPVCHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var pvc v1.PersistentVolumeClaim
	err := c.ShouldBind(&pvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "invalid persistent volume claim json",
		})
		return
	}
	namespace := c.Param("namespace")
	if namespace == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "namespace is required",
		})
	}
	err = s.validatePVC(&pvc, namespace)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: err.Error(),
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumeclaims/%s", namespace, pvc.Name)
	uid, err := s.store_cli.Get(namespaceKey)
	if err == nil && uid != "" {
		c.JSON(http.StatusConflict, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: fmt.Sprintf("persistent volume claim %s/%s already exists", namespace, pvc.Name),
		})
		return
	}
	pvc.Namespace = namespace
	pvc.CreationTimestamp = timestamp.NewTimestamp()
	pvc.UID = v1.UID(uuid.NewUUID())
	pvc.Status.Phase = v1.ClaimPending
	allKey := fmt.Sprintf("/registry/persistentvolumeclaims/%s", pvc.UID)
	pvcJson, _ := json.Marshal(pvc)
	err = s.store_cli.Set(namespaceKey, string(pvc.UID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "error in writing persistent volume claim to etcd",
		})
		return
	}
	err = s.store_cli.Set(allKey, string(pvcJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "error in writing persistent volume claim to etcd",
		})
		return
	}
	c.JSON(http.StatusCreated, v1.BaseResponse[*v1.PersistentVolumeClaim]{
		Data: &pvc,
	})
}

func (s *kubeApiServer) UpdatePVCStatusHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	pvcName := c.Param("pvcname")
	if namespace == "" || pvcName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "namespace and persistent volume claim name cannot be empty",
		})
		return
	}
	var status v1.PersistentVolumeClaimStatus
	err := c.ShouldBind(&status)
	if err != nil {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "invalid persistent volume claim status json",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumeclaims/%s", namespace, pvcName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: fmt.Sprintf("persistent volume claim %s/%s not found", namespace, pvcName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/persistentvolumeclaims/%s", uid)
	pvcJson, err := s.store_cli.Get(allKey)
	if err != nil || pvcJson == "" {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "error in reading persistent volume claim from etcd",
		})
		return
	}
	var pvc v1.PersistentVolumeClaim
	_ = json.Unmarshal([]byte(pvcJson), &pvc)
	pvc.Status = status
	newPVCJson, _ := json.Marshal(pvc)
	err = s.store_cli.Set(allKey, string(newPVCJson))
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "error in writing persistent volume claim to etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.PersistentVolumeClaim]{
		Data: &pvc,
	})
}

func (s *kubeApiServer) GetAllPVCsHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	allKey := "/registry/persistentvolumeclaims"
	res, err := s.store_cli.GetSubKeysValues(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[[]*v1.PersistentVolumeClaim]{
			Error: "error in reading from etcd",
		})
		return
	}
	pvcs := make([]*v1.PersistentVolumeClaim, 0)
	for _, v := range res {
		var pvc v1.PersistentVolumeClaim
		_ = json.Unmarshal([]byte(v), &pvc)
		pvcs = append(pvcs, &pvc)
	}
	c.JSON(http.StatusOK, v1.BaseResponse[[]*v1.PersistentVolumeClaim]{
		Data: pvcs,
	})
}

func (s *kubeApiServer) DeletePVCHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	pvcName := c.Param("pvcname")
	if namespace == "" || pvcName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "namespace and persistent volume claim name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumeclaims/%s", namespace, pvcName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: fmt.Sprintf("persistent volume claim %s/%s not found", namespace, pvcName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/persistentvolumeclaims/%s", uid)
	pvcJson, _ := s.store_cli.Get(allKey)
	var pvc v1.PersistentVolumeClaim
	_ = json.Unmarshal([]byte(pvcJson), &pvc)
	err = s.store_cli.Delete(namespaceKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "error in deleting persistent volume claim from etcd",
		})
		return
	}
	err = s.store_cli.Delete(allKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "error in deleting persistent volume claim from etcd",
		})
		return
	}
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.PersistentVolumeClaim]{
		Data: &pvc,
	})
}

func (s *kubeApiServer) GetPVHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	pvName := c.Param("pvname")
	if namespace == "" || pvName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolume]{
			Error: "namespace and persistent volume name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumes/%s", namespace, pvName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.PersistentVolume]{
			Error: fmt.Sprintf("persistent volume %s/%s not found", namespace, pvName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/persistentvolumes/%s", uid)
	pvJson, _ := s.store_cli.Get(allKey)
	var pv v1.PersistentVolume
	_ = json.Unmarshal([]byte(pvJson), &pv)
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.PersistentVolume]{
		Data: &pv,
	})
}

func (s *kubeApiServer) GetPVCHandler(c *gin.Context) {
	s.lock.Lock()
	defer s.lock.Unlock()
	namespace := c.Param("namespace")
	pvcName := c.Param("pvcname")
	if namespace == "" || pvcName == "" {
		c.JSON(http.StatusBadRequest, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: "namespace and persistent volume claim name cannot be empty",
		})
		return
	}
	namespaceKey := fmt.Sprintf("/registry/namespaces/%s/persistentvolumeclaims/%s", namespace, pvcName)
	uid, err := s.store_cli.Get(namespaceKey)
	if err != nil || uid == "" {
		c.JSON(http.StatusNotFound, v1.BaseResponse[*v1.PersistentVolumeClaim]{
			Error: fmt.Sprintf("persistent volume claim %s/%s not found", namespace, pvcName),
		})
		return
	}
	allKey := fmt.Sprintf("/registry/persistentvolumeclaims/%s", uid)
	pvcJson, _ := s.store_cli.Get(allKey)
	var pvc v1.PersistentVolumeClaim
	_ = json.Unmarshal([]byte(pvcJson), &pvc)
	c.JSON(http.StatusOK, v1.BaseResponse[*v1.PersistentVolumeClaim]{
		Data: &pvc,
	})
}
