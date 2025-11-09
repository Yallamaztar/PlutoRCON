package rcon

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// requireResponse is a commandOption that sets the command to require a successful response
func requireResponse() commandOption {
	return func(s *commandSettings) {
		s.requireSuccess = true
		if s.retries == 0 {
			s.retries = 2
		}
	}
}

// withReadExtension overrides the readExtension window for a command
func withReadExtension(d time.Duration) commandOption {
	return func(s *commandSettings) {
		if d < 0 {
			d = 0
		}
		s.readExtension = d
	}
}

// timeoutOrDefault returns the clients timeout or the default if not set
func (rc *RCONClient) timeoutOrDefault() time.Duration {
	if rc.Timeout <= 0 {
		return defaultReadTimeout
	}
	return rc.Timeout
}

// atoi is a helper to convert string to int, ignoring errors
func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// boolSafe converts a string to bool safely
func boolSafe(s string) bool {
	v := strings.TrimSpace(s)
	return v == "1" || strings.EqualFold(v, "true")
}

// stripColorCodes removes color codes from a string
func stripColorCodes(s string) string {
	if s == "" {
		return s
	}
	// Remove ^<code> where code is an alphanumeric (covers ^0-^9 and potential ^a-^z variants)
	re := regexp.MustCompile(`\^[0-9A-Za-z]`)
	return re.ReplaceAllString(s, "")
}

// normalizeRCON cleans up RCON response strings
func normalizeRCON(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for {
		changed := false
		if strings.HasPrefix(s, "\xFF\xFF\xFF\xFF") {
			s = s[4:]
			changed = true
		}
		if strings.HasPrefix(s, "print\n") {
			s = s[6:]
			changed = true
		}
		if !changed {
			break
		}
	}

	s = strings.ReplaceAll(s, "\n\xFF\xFF\xFF\xFF", "\n")
	s = strings.ReplaceAll(s, "\nprint\n", "\n")
	return strings.TrimSpace(s)
}

// splitNonEmptyLines splits a string into non-empty lines
func splitNonEmptyLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
