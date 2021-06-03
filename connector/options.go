package connector

import (
	"time"

	"github.com/Allenxuxu/gev/eventloop"
)

// Options 服务配置
type Options struct {
	NumLoops int
	IdleTime time.Duration
	Strategy eventloop.LoadBalanceStrategy

	tick                        time.Duration
	wheelSize                   int64
	metricsPath, metricsAddress string
}

// Option ...
type Option func(*Options)

func newOptions(opt ...Option) *Options {
	opts := Options{}

	for _, o := range opt {
		o(&opts)
	}

	if opts.tick == 0 {
		opts.tick = 1 * time.Millisecond
	}
	if opts.wheelSize == 0 {
		opts.wheelSize = 1000
	}

	if opts.Strategy == nil {
		opts.Strategy = eventloop.RoundRobin()
	}

	return &opts
}

// NumLoops work eventloop 的数量
func NumLoops(n int) Option {
	return func(o *Options) {
		o.NumLoops = n
	}
}

// IdleTime 最大空闲时间（秒）
func IdleTime(t time.Duration) Option {
	return func(o *Options) {
		o.IdleTime = t
	}
}

func LoadBalance(strategy eventloop.LoadBalanceStrategy) Option {
	return func(o *Options) {
		o.Strategy = strategy
	}
}
