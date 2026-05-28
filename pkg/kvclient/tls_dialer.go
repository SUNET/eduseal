package kvclient

import (
	"context"
	"crypto/tls"
	"net"
	"time"
)

// buildIPToHostMap resolves every address in nodes (host:port) and returns a
// map from resolved ip:port to the original hostname.  When the cluster
// topology comes back with bare IPs we can look up the hostname the
// certificate was issued for and set TLS ServerName accordingly.
// The key includes the port so that two different hostnames sharing an IP
// on different ports are not confused.
// The caller controls the overall deadline via ctx; each individual lookup
// is additionally capped at 2 seconds.
func buildIPToHostMap(ctx context.Context, nodes []string) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, addr := range nodes {
		host, port, err := net.SplitHostPort(addr)
		if err != nil || host == "" {
			continue
		}
		// If the configured value is already an IP there is nothing to map.
		if net.ParseIP(host) != nil {
			continue
		}
		lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		ips, err := net.DefaultResolver.LookupHost(lookupCtx, host)
		cancel()
		if err != nil {
			continue
		}
		for _, ip := range ips {
			m[net.JoinHostPort(ip, port)] = host
		}
	}
	return m
}

// tlsDialer returns a dial function that behaves like the default TLS
// dialer but, when the destination ip:port appears in ipToHost, clones
// the tls.Config and sets ServerName to the original hostname.  This
// makes cluster-discovered IPs verify against the DNS SANs in the
// node's certificate.
func tlsDialer(baseCfg *tls.Config, ipToHost map[string]string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		cfg := baseCfg

		if hostname, ok := ipToHost[addr]; ok {
			cfg = baseCfg.Clone()
			cfg.ServerName = hostname
		}

		d := tls.Dialer{NetDialer: &net.Dialer{}, Config: cfg}
		return d.DialContext(ctx, network, addr)
	}
}
