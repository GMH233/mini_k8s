package uuid

import (
	"fmt"

	uuid "github.com/google/uuid"
)

func NewUUID() string {
	id := uuid.New()
	str := fmt.Sprint(id)
	return str
}
