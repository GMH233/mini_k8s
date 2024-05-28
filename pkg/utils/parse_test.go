package utils

import "testing"

func TestParseSize(t *testing.T) {
	sizeStrs := []string{
		"100",
		"100Ki",
		"100Mi",
		"100Gi",
	}
	unit := uint64(1)
	for _, sizeStr := range sizeStrs {
		size, err := ParseSize(sizeStr)
		if err != nil {
			t.Errorf("ParseSize failed, err: %v", err)
		}
		if size != 100*unit {
			t.Errorf("ParseSize failed, got: %d, expected: 100", size)
		}
		unit *= 1024
	}
	errSizeStrs := []string{
		"100K",
		"100M",
		"100G",
	}
	for _, sizeStr := range errSizeStrs {
		_, err := ParseSize(sizeStr)
		if err == nil {
			t.Errorf("ParseSize failed, expected error")
		}
	}
}
