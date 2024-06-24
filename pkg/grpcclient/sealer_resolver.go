package grpcclient

import (
	"google.golang.org/grpc/resolver"
)

func (c *Sealer) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	c.client.log.Debug("sealer build", "dns", c.DNS)
	r := &sealerResolver{
		target:     target,
		cc:         cc,
		addrsStore: c.DNS,
	}

	r.start()
	return r, nil
}
func (c *Sealer) Scheme() string { return c.scheme }

type sealerResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *sealerResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint()]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*sealerResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*sealerResolver) Close()                                  {}
