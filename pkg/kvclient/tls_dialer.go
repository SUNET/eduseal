package kvclient

import (
	"context"
	"crypto/tls"
	"net"
)

// buildIPToHostMap resolves every address in nodes (host:port) and returns a
// map from resolved IP to the original hostname.  When the cluster topology
// comes back with bare IPs we can look up the hostname the certificate was
// issued for and set TLS ServerName accordingly.
func buildIPToHostMap(nodes []string) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, addr := range nodes {
		host, _, err := net.SplitHostPort(addr)
		if err != nil || host == "" {
			continue
		}
		// If the configured value is already an IP there is nothing to map.
		if net.ParseIP(host) != nil {
			continue
		}
		ips, err := net.LookupHost(host)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			m[ip] = host
		}
	}
	return m
}

// tlsDialer returns a dial function that behaves like the default TLS
// dialer but, when the destination address is an IP that appears in
// ipToHost, clones the tls.Config and overrides ServerName with the
// original hostname.  This makes cluster-discovered IPs verify against
// the DNS SANs in the node's certificate.
func tlsDialer(baseCfg *tls.Config, ipToHost map[string]string) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		cfg := baseCfg

		host, _, _ := net.SplitHostPort(addr)
		if hostname, ok := ipToHost[host]; ok {
			cfg = baseCfg.Clone()
			cfg.ServerName = hostname
		}

		d := tls.Dialer{NetDialer: &net.Dialer{}, Config: cfg}
		return d.DialContext(ctx, network, addr)
	}
}
