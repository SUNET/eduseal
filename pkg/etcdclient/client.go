package etcdclient

import (
	"context"
	"eduseal/pkg/logger"
	"eduseal/pkg/model"
	"eduseal/pkg/trace"
	"fmt"
	"net/url"
	"strings"
	"time"

	"eduseal/internal/gen/status/v1_status"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client holds the etcd object
type Client struct {
	cfg          *model.Cfg
	EtcdClient   *clientv3.Client
	Probe        *v1_status.StatusProbe
	statusTick   *time.Ticker
	errChan      chan error
	downErrChan  chan string
	log          *logger.Log
	tp           *trace.Tracer
	statusResult statusResults

	Doc *Doc
}
type statusResults map[string]statusResult

func (s statusResults) mkProbe() *v1_status.StatusProbe {
	p := &v1_status.StatusProbe{
		Name:          "etcd",
		Healthy:       true,
		LastCheckedTS: timestamppb.Now(),
	}
	for k, v := range s {
		if !v.healthy {
			p.Healthy = false
		}
		p.Message += fmt.Sprintf("%s:%v:%v,", k, v.leader, v.healthy)
	}

	// remove trailing comma
	p.Message = strings.TrimRight(p.Message, ",")
	return p
}

type statusResult struct {
	healthy bool
	leader  bool
}

// New creates a new instance of etcd
func New(ctx context.Context, cfg *model.Cfg, tp *trace.Tracer, log *logger.Log) (*Client, error) {
	c := &Client{
		cfg:          cfg,
		log:          log,
		Probe:        &v1_status.StatusProbe{},
		statusTick:   time.NewTicker(10 * time.Second),
		errChan:      make(chan error, 100),
		downErrChan:  make(chan string, 100),
		tp:           tp,
		statusResult: statusResults{},
	}

	var err error
	c.EtcdClient, err = clientv3.New(clientv3.Config{
		Endpoints:   c.cfg.Common.ETCD.Addresses,
		DialTimeout: 2 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	c.Doc = &Doc{
		client: c,
		key:    "doc:%s:%s",
	}

	go func() {
		for {
			select {
			case hostname := <-c.downErrChan:
				c.statusResult[hostname] = statusResult{
					healthy: false,
					leader:  false,
				}
			case err := <-c.errChan:
				if err != nil {
					c.log.Error(err, "etcd client error")
					fmt.Println(err)
				}
			case <-ctx.Done():
				c.log.Debug("etcd client done")
			case <-c.statusTick.C:
				for _, etcdHost := range c.cfg.Common.ETCD.Addresses {
					c.errChan <- c.status(ctx, etcdHost)
				}
				c.Probe = c.statusResult.mkProbe()
			}
		}
	}()

	return c, nil
}

// status returns the status of the database
func (c *Client) status(ctx context.Context, etcdHost string) error {
	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*2))
	defer cancel()

	u, err := url.Parse(etcdHost)
	if err != nil {
		return err
	}

	memberStatus, err := c.EtcdClient.Status(ctx, etcdHost)
	if err != nil {
		c.downErrChan <- u.Hostname()
		return nil
	}

	c.statusResult[u.Hostname()] = statusResult{
		healthy: true,
		leader:  memberStatus.Header.MemberId == memberStatus.Leader,
	}

	return nil
}

// Close closes the connection to etcd
func (c *Client) Close(ctx context.Context) error {
	if err := c.EtcdClient.Close(); err != nil {
		return err
	}
	return nil
}
