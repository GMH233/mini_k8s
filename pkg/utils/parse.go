package utils

import (
	"fmt"
	"regexp"
	"strconv"
)

func ParseSize(sizeStr string) (uint64, error) {
	sizeUnits := map[string]uint64{
		"Ki": 1024,
		"Mi": 1024 * 1024,
		"Gi": 1024 * 1024 * 1024,
	}
	re := regexp.MustCompile(`^(\d+)([KMG]i)?$`)
	matches := re.FindStringSubmatch(sizeStr)

	// fmt.Printf("Matches: %v\n", matches)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}
	value, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return 0, err
	}
	unit := matches[2]
	if unit == "" {
		return value, nil
	}
	multiplier, ok := sizeUnits[unit]
	if !ok {
		return 0, fmt.Errorf("invalid size unit: %s", unit)
	}
	return value * multiplier, nil
}
