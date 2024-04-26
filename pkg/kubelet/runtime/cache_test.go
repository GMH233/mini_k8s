package runtime

import (
	"fmt"
	v1 "minikubernetes/pkg/api/v1"
	"sync"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	c := NewCache()
	id := "test-id"
	ts := time.Now()

	status, err := c.Get(v1.UID(id))
	if status != nil || err == nil {
		t.Fatalf("Get() = %v, %v, want nil, non-nil", status, err)
	}

	c.Set(v1.UID(id), &PodStatus{}, fmt.Errorf(""), ts)
	status, err = c.Get(v1.UID(id))
	if status == nil || err == nil {
		t.Fatalf("Get() = %v, %v, want non-nil, non-nil", status, err)
	}

	c.UpdateTime(ts)
	status, err = c.GetNewerThan(v1.UID(id), ts.Add(-time.Second))
	if status == nil || err == nil {
		t.Fatalf("GetNewerThan() = %v, %v, want non-nil, non-nil", status, err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func(wg *sync.WaitGroup) {
		t.Log("GetNewerThan() start")
		c.GetNewerThan(v1.UID(id), ts.Add(2*time.Second))
		t.Log("GetNewerThan() done")
		wg.Done()
	}(&wg)
	go func(wg *sync.WaitGroup) {
		t.Log("Set1 start")
		c.Set(v1.UID(id), &PodStatus{}, nil, ts.Add(time.Second))
		t.Log("Set1 done")
		time.Sleep(time.Second)
		t.Log("Set2 start")
		c.Set(v1.UID(id), &PodStatus{}, nil, ts.Add(2*time.Second))
		t.Log("Set2 done")
		wg.Done()
	}(&wg)
	wg.Wait()
}
