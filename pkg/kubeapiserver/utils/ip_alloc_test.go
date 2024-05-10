package utils

import (
	"fmt"
	"testing"
)

func TestAllocIP(t *testing.T) {
	bitmap := make([]byte, IPPoolSize/8)
	for i := 0; i < IPPoolSize; i++ {
		ip, err := AllocIP(bitmap)
		if err != nil {
			t.Fatalf("alloc ip failed: %v", err)
		}
		t.Logf("alloc ip: %s", ip)
	}
	str := string(bitmap)
	bitmap = []byte(str)
	_, err := AllocIP(bitmap)
	if err == nil {
		t.Fatalf("alloc ip should fail")
	}
}

func TestFreeIP(t *testing.T) {
	bitmap := make([]byte, IPPoolSize/8)
	for i := 0; i < IPPoolSize; i++ {
		_, err := AllocIP(bitmap)
		if err != nil {
			t.Fatalf("alloc ip failed: %v", err)
		}
	}
	for i := IPPoolSize - 1; i >= 0; i-- {
		ip := fmt.Sprintf("%s%d", IPPrefix, i)
		err := FreeIP(ip, bitmap)
		if err != nil {
			t.Fatalf("free ip failed: %v", err)
		}
		t.Logf("free ip: %s", ip)
	}
	for i := 0; i < IPPoolSize; i++ {
		_, err := AllocIP(bitmap)
		if err != nil {
			t.Fatalf("alloc ip failed: %v", err)
		}
	}
}

func TestFreeAndAlloc(t *testing.T) {
	bitmap := make([]byte, IPPoolSize/8)
	for i := 0; i < IPPoolSize; i++ {
		_, err := AllocIP(bitmap)
		if err != nil {
			t.Fatalf("alloc ip failed: %v", err)
		}
	}
	err := FreeIP("100.0.0.98", bitmap)
	if err != nil {
		t.Fatalf("free ip failed: %v", err)
	}
	ip, err := AllocIP(bitmap)
	if err != nil {
		t.Fatalf("alloc ip failed: %v", err)
	}
	if ip != "100.0.0.98" {
		t.Fatalf("free and alloc failed: %s", ip)
	}
}
