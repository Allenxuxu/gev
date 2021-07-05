package connection

type SendInLoopFunc func(interface{})

type Options struct {
	sendInLoopFinish SendInLoopFunc
}

type Option func(*Options)

func SendInLoop(f SendInLoopFunc) Option {
	return func(o *Options) {
		o.sendInLoopFinish = f
	}
}
