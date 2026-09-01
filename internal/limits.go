package internal

import "time"

const (
	// defaultMaxPayloadBytes caps Validate request data and inline schema size.
	defaultMaxPayloadBytes = 4 << 20 // 4 MiB
	defaultRegexTimeout    = 1 * time.Second
)
