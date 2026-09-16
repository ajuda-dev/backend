package controller

import (
	"net"
	"testing"
	"time"
)

func TestWaitClientDisconnect_ClosesWhenPeerCloses(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() {
		_ = server.Close()
		_ = client.Close()
	})

	closed := make(chan struct{})
	go waitClientDisconnect(server, closed)

	select {
	case <-closed:
		t.Fatal("did not expect disconnect before the peer closed")
	case <-time.After(50 * time.Millisecond):
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("expected disconnect when the peer closed the connection")
	}
}

func TestWaitClientDisconnect_NilConnDoesNotClose(t *testing.T) {
	closed := make(chan struct{})
	go waitClientDisconnect(nil, closed)
	select {
	case <-closed:
		t.Fatal("nil conn must not signal disconnect")
	case <-time.After(50 * time.Millisecond):
	}
}
