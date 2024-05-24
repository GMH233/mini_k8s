package envoy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"math/rand"
	v1 "minikubernetes/pkg/api/v1"
	"minikubernetes/pkg/kubeclient"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"time"
)

const (
	InboundPort  = "15006"
	OutboundPort = "15001"
	UID          = "1337"
	GID          = "1337"
)

type Envoy struct {
	inboundRouter  *gin.Engine
	outboundRouter *gin.Engine
	mapping        v1.SidecarMapping
	kubeClient     kubeclient.Client
}

func NewEnvoy(apiServerIP string) (*Envoy, error) {
	envoy := &Envoy{}
	envoy.inboundRouter = gin.Default()
	envoy.outboundRouter = gin.Default()
	envoy.mapping = make(v1.SidecarMapping)
	envoy.kubeClient = kubeclient.NewClient(apiServerIP)
	return envoy, nil
}

func (e *Envoy) inboundProxy(c *gin.Context) {
	log.Printf("Inbound proxy receive request: %v %v %v\n", c.Request.Method, c.Request.URL, c.Request.Host)
	host := c.Request.Host
	ip, port, err := net.SplitHostPort(host)
	if err != nil {
		if net.ParseIP(host) != nil {
			// host仅有ip，使用默认端口
			port = "80"
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Domain name not supported"})
			return
		}
	}
	ip = "127.0.0.1"
	target := c.Request.URL
	if target.Scheme == "" {
		target.Scheme = "http"
	}
	target.Host = fmt.Sprintf("%s:%s", ip, port)
	log.Printf("Target: %v\n", target)
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Director = func(req *http.Request) {
		//req.URL = target
		req.Host = target.Host
		//req.Method = c.Request.Method
		//req.Header = c.Request.Header
	}
	reverseProxy.ServeHTTP(c.Writer, c.Request)
}

func (e *Envoy) outboundProxy(c *gin.Context) {
	log.Printf("Outbound proxy receive request: %v %v %v\n", c.Request.Method, c.Request.URL, c.Request.Host)
	host := c.Request.Host
	ip, port, err := net.SplitHostPort(host)
	useOriginal := false
	if err != nil {
		if net.ParseIP(host) != nil {
			// host仅有ip，使用默认端口
			ip = host
			port = "80"
		} else {
			// host为域名
			useOriginal = true
		}
	}
	if _, ok := e.mapping[fmt.Sprintf("%s:%s", ip, port)]; !ok {
		useOriginal = true
	}
	var target *url.URL
	if useOriginal {
		// 不进行转发
		target = c.Request.URL
		if target.Scheme == "" {
			target.Scheme = "http"
		}
		if target.Host == "" {
			target.Host = c.Request.Host
		}
	} else {
		endpoints := e.mapping[fmt.Sprintf("%s:%s", ip, port)]
		if len(endpoints) == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available endpoints"})
			return
		}
		useWeight := endpoints[0].Weight != nil
		var dest string
		if useWeight {
			dest, err = getDestinationWithWeight(endpoints)
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
		} else {
			dest, err = getDestinationWithURL(endpoints, c.Param("path"))
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
				return
			}
		}
		target = c.Request.URL
		if target.Scheme == "" {
			target.Scheme = "http"
		}
		target.Host = dest
	}
	log.Printf("Target: %v\n", target)
	//c.JSON(http.StatusOK, "OK")
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Director = func(req *http.Request) {
		//req.URL = target
		req.Host = target.Host
		//req.Method = c.Request.Method
		//req.Header = c.Request.Header
	}
	reverseProxy.ServeHTTP(c.Writer, c.Request)
}

func getDestinationWithWeight(endpoints []v1.SidecarEndpoints) (string, error) {
	var totalWeight int32 = 0
	for _, endpoint := range endpoints {
		if endpoint.Weight == nil {
			return "", fmt.Errorf("weight not set")
		}
		totalWeight += *endpoint.Weight
	}
	randWeight := int32(rand.Intn(int(totalWeight)))
	for _, endpoint := range endpoints {
		for _, ep := range endpoint.Endpoints {
			if randWeight < *endpoint.Weight {
				return fmt.Sprintf("%v:%v", ep.IP, ep.TargetPort), nil
			}
			randWeight -= *endpoint.Weight
		}
	}
	return "", fmt.Errorf("no available endpoint")
}

func getDestinationWithURL(endpoints []v1.SidecarEndpoints, requestURL string) (string, error) {
	for _, endpoint := range endpoints {
		if endpoint.URL == nil {
			return "", fmt.Errorf("url not set")
		}
		match, err := regexp.MatchString(*endpoint.URL, requestURL)
		if err != nil {
			return "", err
		}
		if match {
			if len(endpoint.Endpoints) == 0 {
				return "", fmt.Errorf("no available endpoint")
			}
			randIdx := rand.Intn(len(endpoint.Endpoints))
			return fmt.Sprintf("%v:%v", endpoint.Endpoints[randIdx].IP, endpoint.Endpoints[randIdx].TargetPort), nil
		}
	}
	return "", fmt.Errorf("no available endpoint")
}

func getMockMapping() v1.SidecarMapping {
	return v1.SidecarMapping{
		"100.0.0.0:80": []v1.SidecarEndpoints{
			{
				Weight: &[]int32{1}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 801,
					},
					{
						IP:         "127.0.0.1",
						TargetPort: 802,
					},
				},
			},
			{
				Weight: &[]int32{2}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 803,
					},
				},
			},
		},
		"100.0.0.0:90": []v1.SidecarEndpoints{
			{
				Weight: &[]int32{0}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 801,
					},
					{
						IP:         "127.0.0.1",
						TargetPort: 802,
					},
				},
			},
			{
				Weight: &[]int32{1}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 803,
					},
				},
			},
		},
		"100.0.0.1:80": []v1.SidecarEndpoints{
			{
				URL: &[]string{"^/api/v1/.*$"}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 801,
					},
					{
						IP:         "127.0.0.1",
						TargetPort: 802,
					},
				},
			},
			{
				URL: &[]string{"^/api/v2/.*$"}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 803,
					},
				},
			},
		},
		"100.0.0.1:90": []v1.SidecarEndpoints{
			{
				URL: &[]string{"^/api/v1/.*$"}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 801,
					},
					{
						IP:         "127.0.0.1",
						TargetPort: 802,
					},
				},
			},
			{
				URL: &[]string{"^/api/v2/.*$"}[0],
				Endpoints: []v1.SingleEndpoint{
					{
						IP:         "127.0.0.1",
						TargetPort: 803,
					},
				},
			},
		},
	}
}

func (e *Envoy) watchMapping() {
	for {
		mapping, err := e.kubeClient.GetSidecarMapping()
		if err != nil {
			log.Printf("Get sidecar mapping failed: %v\n", err)
		}
		// mock
		//mapping := getMockMapping()
		e.mapping = mapping
		time.Sleep(3560 * time.Millisecond)
	}
}

func (e *Envoy) Run() {
	e.inboundRouter.Any("/*path", e.inboundProxy)
	e.outboundRouter.Any("/*path", e.outboundProxy)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		log.Printf("Listening inbound port %s\n", InboundPort)
		log.Fatal(e.inboundRouter.Run(fmt.Sprintf(":%s", InboundPort)))
	}()
	go func() {
		log.Printf("Listening outbound port %s\n", OutboundPort)
		log.Fatal(e.outboundRouter.Run(fmt.Sprintf(":%s", OutboundPort)))
	}()
	go e.watchMapping()
	<-signalChan
	log.Printf("Envoy shut down.")
}
