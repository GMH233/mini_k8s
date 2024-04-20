package app

import (
	"fmt"
	"log"
	v1 "minikubernetes/pkg/api/v1"
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
)

/* NAMESPACE
 * and NODE HARDSHIT
 */
const (
	Default_Namespace = "default"
	Default_Nodename  = "node-0"
)

type kubeApiServer struct {
	router    *gin.Engine
	listen_ip string
	port      int
}

type KubeApiServer interface {
	Run()
}

/* Simulate etcd.
 */

// TODO add etcd handler

var np_pod_map map[string][]v1.UID
var node_pod_map map[string][]v1.UID
var pod_hub map[v1.UID]v1.Pod

func (ser *kubeApiServer) init() {
	// TODO

	// assume that node-0 already registered

	np_pod_map = make(map[string][]v1.UID)
	node_pod_map = make(map[string][]v1.UID)
	pod_hub = make(map[v1.UID]v1.Pod)

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
	ser.init()

	// exactly initialing
	log.Println("kubeApiServer is binding handlers")
	ser.binder()

	log.Printf("binding ip: %v, listening port: %v\n", ser.listen_ip, ser.port)
	ser.router.Run(ser.listen_ip + ":" + fmt.Sprint(ser.port))

}

func NewKubeApiServer() (KubeApiServer, error) {
	// TODO: return an kubeapi server
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

	ser.router.GET(All_pods_url, GetAllPodsHandler)
	ser.router.GET(Namespace_Pods_url, GetPodsByNamespaceHandler)
	ser.router.GET(Single_pod_url, GetPodHandler)
	ser.router.POST(Namespace_Pods_url, AddPodHandler) // ** for single-pod testing
	ser.router.PUT(Single_pod_url, UpdatePodHandler)
	ser.router.DELETE(Single_pod_url, DeletePodHandler)
	ser.router.GET(Pod_status_url, GetPodStatusHandler)
	ser.router.PUT(Pod_status_url, PutPodStatusHandler) // only modify the status of a single pod

	ser.router.GET(Node_pods_url, GetPodsByNodeHandler) // ** for single-pod testing

}

// handlers (trivial)

// Nodes have no namespace.
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

func GetAllPodsHandler(con *gin.Context) {

}
func GetPodsByNamespaceHandler(con *gin.Context) {

}
func GetPodHandler(con *gin.Context) {

}
func AddPodHandler(con *gin.Context) {
	// assign a pod to a node
	log.Println("Adding a new pod")

	var pod v1.Pod
	err := con.ShouldBind(&pod)
	if err != nil {
		log.Panicln("something is wrong when parsing Pod")
		return
	}

	pod.ObjectMeta.UID = (v1.UID)(uuid.NewUUID())

	pod.ObjectMeta.CreationTimestamp = timestamp.NewTimestamp()

	pod.Status.Phase = v1.PodPending

	/* fake store pod to:
	1. namespace it belongs
	2. node it belongs
	*/

	// assume no collision
	np_pod_map["default"] = append(np_pod_map["default"], pod.ObjectMeta.UID)
	node_pod_map["node-0"] = append(node_pod_map["node-0"], pod.ObjectMeta.UID)
	pod_hub[pod.ObjectMeta.UID] = pod

	con.JSON(http.StatusCreated, gin.H{
		"message": "successfully created pod",
		"UUID":    pod.ObjectMeta.UID,
	})

}
func PutPodStatusHandler(con *gin.Context) {

}
func DeletePodHandler(con *gin.Context) {

}

func GetPodStatusHandler(con *gin.Context) {

}

func UpdatePodHandler(con *gin.Context) {

}

func GetPodsByNodeHandler(con *gin.Context) {

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
	log.Println("getting info of node: %v", node_name)
	// find all v1.Pod s in the pod_hub
	all_pod_str := make([]v1.Pod, 0)

	for _, v := range node_pod_map[node_name] {

		pod := pod_hub[v]
		all_pod_str = append(all_pod_str, pod)
	}

	// then return all of them
	con.JSON(http.StatusOK,
		gin.H{
			"data": all_pod_str,
		},
	)
}
