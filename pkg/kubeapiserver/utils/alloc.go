package utils

import "fmt"

const IPPrefix = "100.0.0."

const IPPoolSize = 256

const NodePrefix = "node-"

const NodePoolSize = 64

func AllocIP(bitmap []byte) (string, error) {
	if len(bitmap) != IPPoolSize/8 {
		return "", fmt.Errorf("invalid bitmap size")
	}
	for idx, b := range bitmap {
		if b == 0xff {
			continue
		}
		for i := 0; i < 8; i++ {
			if b&(1<<uint(i)) != 0 {
				continue
			}
			ip := fmt.Sprintf("%s%d", IPPrefix, idx*8+i)
			b |= 1 << uint(i)
			bitmap[idx] = b
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available ip")
}

func FreeIP(ip string, bitmap []byte) error {
	if len(bitmap) != IPPoolSize/8 {
		return fmt.Errorf("invalid bitmap size")
	}
	idx := 0
	_, err := fmt.Sscanf(ip, IPPrefix+"%d", &idx)
	if err != nil {
		return err
	}
	if idx < 0 || idx >= IPPoolSize {
		return fmt.Errorf("invalid ip")
	}
	b := bitmap[idx/8]
	if b&(1<<uint(idx%8)) == 0 {
		return fmt.Errorf("ip is not allocated")
	}
	bitmap[idx/8] = b &^ (1 << uint(idx%8))
	return nil
}

func AllocNode(bitmap []byte) (string, error) {
	nodeNum, err := allocInternal(bitmap, NodePoolSize)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", NodePrefix, nodeNum), nil
}

func FreeNode(node string, bitmap []byte) error {
	nodeNum := 0
	_, err := fmt.Sscanf(node, NodePrefix+"%d", &nodeNum)
	if err != nil {
		return err
	}
	return freeInternal(bitmap, NodePoolSize, nodeNum)
}

func allocInternal(bitmap []byte, size int) (int, error) {
	if len(bitmap) != size/8 {
		return 0, fmt.Errorf("invalid bitmap size")
	}
	for idx, b := range bitmap {
		if b == 0xff {
			continue
		}
		for i := 0; i < 8; i++ {
			if b&(1<<uint(i)) != 0 {
				continue
			}
			b |= 1 << uint(i)
			bitmap[idx] = b
			return idx*8 + i, nil
		}
	}
	return 0, fmt.Errorf("no available slot")
}

func freeInternal(bitmap []byte, size, idx int) error {
	if len(bitmap) != size/8 {
		return fmt.Errorf("invalid bitmap size")
	}
	if idx < 0 || idx >= size {
		return fmt.Errorf("invalid index")
	}
	b := bitmap[idx/8]
	if b&(1<<uint(idx%8)) == 0 {
		return fmt.Errorf("slot is not allocated")
	}
	bitmap[idx/8] = b &^ (1 << uint(idx%8))
	return nil
}
