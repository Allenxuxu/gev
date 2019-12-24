package gev

import (
	"time"

	"github.com/Allenxuxu/gev/connection"
)

// Options 服务配置
type Options struct {
	Network   string
	Address   string
	NumLoops  int
	ReusePort bool

	tick      time.Duration
	wheelSize int64
	IdleTime  time.Duration
	Protocol  connection.Protocol
}

// Option ...
type Option func(*Options)

func newOptions(opt ...Option) *Options {
	opts := Options{}

	for _, o := range opt {
		o(&opts)
	}

	if len(opts.Network) == 0 {
		opts.Network = "tcp"
	}
	if len(opts.Address) == 0 {
		opts.Address = ":1388"
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

	return &opts
}

// ReusePort 设置 SO_REUSEPORT
func ReusePort(reusePort bool) Option {
	return func(o *Options) {
		o.ReusePort = reusePort
	}
}

// Network [tcp] 暂时只支持tcp
func Network(n string) Option {
	return func(o *Options) {
		o.Network = n
	}
}

// Address server 监听地址
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}

// NumLoops work eventloop 的数量
func NumLoops(n int) Option {
	return func(o *Options) {
		o.NumLoops = n
	}
}

// Protocol 数据包处理
func Protocol(p connection.Protocol) Option {
	return func(o *Options) {
		o.Protocol = p
	}
}

// IdleTime 最大空闲时间（秒）
func IdleTime(t time.Duration) Option {
	return func(o *Options) {
		o.IdleTime = t
	}
}
