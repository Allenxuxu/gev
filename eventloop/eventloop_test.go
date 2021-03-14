package eventloop

import (
	"log"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/Allenxuxu/gev/poller"
	"github.com/Allenxuxu/toolkit/sync/atomic"
	"github.com/Allenxuxu/toolkit/sync/spinlock"
)

func TestEventLoop_RunLoop(t *testing.T) {
	el, err := New()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		go func() {
			el.QueueInLoop(func() {
				log.Println("runinloop")
			})
		}()
	}

	go func() {
		time.Sleep(time.Second)
		if err := el.Stop(); err != nil {
			panic(err)
		}
	}()

	el.RunLoop()
}

func TestEventLoopSize(t *testing.T) {
	//	eventHandling atomic.Bool
	//	poll          *poller.Poller
	//	mu            spinlock.SpinLock
	//	sockets       *sync.Map
	//	packet        []byte
	//	pendingFunc   []func()

	t.Log(unsafe.Sizeof(atomic.Bool{}))
	t.Log(unsafe.Sizeof(&poller.Poller{}))
	t.Log(unsafe.Sizeof(spinlock.SpinLock{}))
	t.Log(unsafe.Sizeof(&sync.Map{}))
	t.Log(unsafe.Sizeof([]byte{}))
	t.Log(unsafe.Sizeof([]func(){}))

	t.Log(unsafe.Sizeof(eventLoopLocal{}))
	t.Log(unsafe.Sizeof(EventLoop{}))
}
