package gev

import (
	"log"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

func TestEventLoop_RunLoop(t *testing.T) {
	el, err := newEventLoop()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		go func() {
			el.queueInLoop(func() {
				log.Println("runinloop")
			})
		}()
	}

	go func() {
		time.Sleep(time.Second)
		if err := el.stop(); err != nil {
			panic(err)
		}
	}()

	el.runLoop()
}

func TestEventLoopSize(t *testing.T) {
	t.Log(unsafe.Sizeof(eventLoopLocal{}))
	t.Log(unsafe.Sizeof(eventLoop{}))

	assert.Equal(t, 128, int(unsafe.Sizeof(eventLoop{})))
}
