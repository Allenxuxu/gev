package gev

type SendInLoopFunc func(interface{})

type ConnectionOptions struct {
	sendInLoopFinish SendInLoopFunc
}

type ConnectionOption func(*ConnectionOptions)

func SendInLoop(f SendInLoopFunc) ConnectionOption {
	return func(o *ConnectionOptions) {
		o.sendInLoopFinish = f
	}
}
