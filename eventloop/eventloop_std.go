// +build windows

package eventloop

import "github.com/Allenxuxu/toolkit/sync/atomic"

type EventLoop struct {
	ConnCunt atomic.Int64
}

func (l *EventLoop) ConnectionCount() int64 {
	return l.ConnCunt.Get()
}
