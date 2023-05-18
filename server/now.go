package server

import "time"

// Now returns the current time in ms.
func Now() int64 {
	return time.Now().UnixMilli()
}
