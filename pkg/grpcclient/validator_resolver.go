package grpcclient

import (
	"google.golang.org/grpc/resolver"
)

func (c *Validator) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	c.client.log.Debug("validator build", "dns", c.DNS)
	r := &validatorResolver{
		target:     target,
		cc:         cc,
		addrsStore: c.DNS,
	}
	r.start()
	return r, nil
}
func (c *Validator) Scheme() string { return c.scheme }

type validatorResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *validatorResolver) start() {
	addrStrs := r.addrsStore[r.target.Endpoint()]
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}
	r.cc.UpdateState(resolver.State{Addresses: addrs})
}
func (*validatorResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*validatorResolver) Close()                                  {}
