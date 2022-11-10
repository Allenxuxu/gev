// +build !windows

package poller

import (
	"errors"
	"testing"
	"time"
)

func TestPoller_Poll(t *testing.T) {
	s, err := Create()
	if err != nil {
		t.Fatal(err)
	}

	errs := make(chan error, 1)
	go s.Poll(func(fd int, event Event) {
		if fd != -1 {
			errs <- errors.New("fd should be -1")
		}
	})

	time.Sleep(time.Millisecond * 500)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
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
