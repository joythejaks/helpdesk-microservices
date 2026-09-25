package ws

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// trustedProxies are the networks whose X-Forwarded-For header we believe
// (e.g. the Caddy pod). Empty by default, which means "never trust
// X-Forwarded-For": a client that reaches this service directly can put
// anything it likes in that header, and honouring it would let it mint a
// fresh rate-limit bucket per request.
var trustedProxies []*net.IPNet

// SetTrustedProxies parses a list of CIDRs (a bare IP is treated as a
// single-host network) and installs them. Call once at startup.
func SetTrustedProxies(cidrs []string) error {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			if ip := net.ParseIP(c); ip != nil && ip.To4() != nil {
				c += "/32"
			} else {
				c += "/128"
			}
		}
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return fmt.Errorf("invalid trusted proxy %q: %w", c, err)
		}
		nets = append(nets, n)
	}
	trustedProxies = nets
	return nil
}

func isTrusted(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range trustedProxies {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

// clientIP returns the address to rate-limit on. Behind a reverse proxy
// (Caddy) the TCP peer is the proxy for every client, so all WebSocket
// clients would share one bucket; when — and only when — the peer is a
// trusted proxy, the real client is the right-most X-Forwarded-For entry
// that is not itself a trusted proxy. Walking from the right ignores any
// entries the client prepended to fake its address.
func clientIP(r *http.Request) string {
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		peer = r.RemoteAddr
	}

	if !isTrusted(peer) {
		return peer
	}

	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return peer
	}

	parts := strings.Split(xff, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		ip := strings.TrimSpace(parts[i])
		if net.ParseIP(ip) == nil {
			return peer // malformed header: don't guess
		}
		if !isTrusted(ip) {
			return ip
		}
	}

	// Every hop is a trusted proxy (a client that is itself inside the
	// trusted range, or a source address rewritten by the platform). The
	// outermost address is still a better key than the immediate peer,
	// which is one of possibly many proxy replicas.
	return strings.TrimSpace(parts[0])
}
