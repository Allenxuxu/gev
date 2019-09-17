package gev

// Options 服务配置
type Options struct {
	Network  string
	Address  string
	NumLoops int
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

	return &opts
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
