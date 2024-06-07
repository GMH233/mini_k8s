package timestamp

import (
	"time"
)

func NewTimestamp() time.Time {
	return time.Now()
}

func FormatUTCZ(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.99999999Z")
}
