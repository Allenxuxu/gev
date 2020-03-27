package poller

import (
	"testing"
	"time"
)

func TestPoller_Poll(t *testing.T) {
	s, err := Create()
	if err != nil {
		t.Fatal(err)
	}

	go s.Poll(func(fd int, event Event) {
		if fd != -1 {
			t.Fatal()
		}
	})
	time.Sleep(time.Millisecond * 500)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPoller_Close(t *testing.T) {
	s, err := Create()
	if err != nil {
		t.Fatal(err)
	}

	if err = s.Close(); err == nil {
		t.Fatal("poller should be closed")
	}
}
