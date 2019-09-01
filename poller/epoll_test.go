// +build linux

package poller

import (
	"testing"
)

func TestPoller_Poll(t *testing.T) {
	s, err := Create()
	if err != nil {
		t.Fatal(err)
	}

	go s.Poll(func(fd int, event uint32) {
		t.Log(fd)
	})

	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPoller_Close(t *testing.T) {
	s, err := Create()
	if err != nil {
		t.Fatal(err)
	}

	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
}
