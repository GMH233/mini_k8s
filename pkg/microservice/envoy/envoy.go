package envoy

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
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
}

func NewEnvoy() (*Envoy, error) {
	envoy := &Envoy{}
	envoy.inboundRouter = gin.Default()
	envoy.outboundRouter = gin.Default()
	return envoy, nil
}

func (e *Envoy) inboundProxy(c *gin.Context) {
	log.Printf("Inbound proxy receive request: %v %v %v\n", c.Request.Method, c.Request.URL, c.Request.Host)
	target, _ := url.Parse("http://example.com")
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = c.Param("path")
		req.Host = target.Host
	}
	reverseProxy.ServeHTTP(c.Writer, c.Request)
}

func (e *Envoy) outboundProxy(c *gin.Context) {
	log.Printf("Outbound proxy receive request: %v %v %v\n", c.Request.Method, c.Request.URL, c.Request.Host)
	target, _ := url.Parse("http://example.com")
	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	reverseProxy.Director = func(req *http.Request) {
		req.Header = c.Request.Header
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = c.Param("path")
		req.Host = target.Host
	}
	reverseProxy.ServeHTTP(c.Writer, c.Request)
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
	<-signalChan
	log.Printf("Envoy shut down.")
}
