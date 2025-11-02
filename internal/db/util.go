package db

import "time"

// nowUnix returns current Unix timestamp
func nowUnix() int64 {
	return time.Now().Unix()
}
