package ui

import (
	"fmt"
	"strconv"
	"time"
)

// FormatInt formats an integer with thousand separators.
func FormatInt(n uint64) string {
	s := strconv.FormatUint(n, 10)
	if len(s) <= 3 {
		return s
	}

	var out []byte
	for i, j := len(s)-1, 1; i >= 0; i, j = i-1, j+1 {
		out = append(out, s[i])
		if j%3 == 0 && i != 0 {
			out = append(out, ',')
		}
	}

	for l, r := 0, len(out)-1; l < r; l, r = l+1, r-1 {
		out[l], out[r] = out[r], out[l]
	}

	return string(out)
}

// Ago formats a time as a human-readable duration since now.
func Ago(t *time.Time) string {
	if t == nil || (*t).IsZero() {
		return "never"
	}
	return fmt.Sprintf("%s ago", time.Since(*t).Round(time.Second))
}
