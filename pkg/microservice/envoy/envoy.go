package envoy

import (
	"log"
	"time"
)

type Envoy struct {
}

func (e *Envoy) Run() {
	for {
		log.Printf("Envoy sync loop...\n")
		time.Sleep(5 * time.Second)
	}
}
