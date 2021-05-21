package gev

import (
	"time"

	"github.com/Allenxuxu/gev/connection"
)

// ConnectorOptions 服务配置
type ConnectorOptions struct {
	NumLoops int
	IdleTime time.Duration
	Protocol connection.Protocol
	Strategy LoadBalanceStrategy

	tick                        time.Duration
	wheelSize                   int64
	metricsPath, metricsAddress string
}

// Option ...
type ConnectorOption func(*ConnectorOptions)

func newConnectorOptions(opt ...ConnectorOption) *ConnectorOptions {
	opts := ConnectorOptions{}

	for _, o := range opt {
		o(&opts)
	}

	if opts.tick == 0 {
		opts.tick = 1 * time.Millisecond
	}
	if opts.wheelSize == 0 {
		opts.wheelSize = 1000
	}
	if opts.Protocol == nil {
		opts.Protocol = &connection.DefaultProtocol{}
	}
	if opts.Strategy == nil {
		opts.Strategy = RoundRobin()
	}

	return &opts
}

// NumLoops work eventloop 的数量
func ConnectorNumLoops(n int) ConnectorOption {
	return func(o *ConnectorOptions) {
		o.NumLoops = n
	}
}

// Protocol 数据包处理
func ConnectorProtocol(p connection.Protocol) ConnectorOption {
	return func(o *ConnectorOptions) {
		o.Protocol = p
	}
}

// IdleTime 最大空闲时间（秒）
func ConnectorIdleTime(t time.Duration) ConnectorOption {
	return func(o *ConnectorOptions) {
		o.IdleTime = t
	}
}

func ConnectorLoadBalance(strategy LoadBalanceStrategy) ConnectorOption {
	return func(o *ConnectorOptions) {
		o.Strategy = strategy
	}
}
