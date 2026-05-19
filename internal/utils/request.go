package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetRequestIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		if first, _, found := strings.Cut(ip, ","); found {
			return strings.TrimSpace(first)
		}
		return ip
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
