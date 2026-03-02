package handler

import (
	"net"
	"net/http"
	"strings"
)

func auditContext(r *http.Request) (ip, userAgent string) {
	userAgent = r.Header.Get("User-Agent")

	// Check X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip = strings.TrimSpace(parts[0])
		return ip, userAgent
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return ip, userAgent
}
