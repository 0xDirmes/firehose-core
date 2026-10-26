package relayer

import (
	"errors"
	"testing"
	"time"

	"github.com/streamingfast/bstream"
	"github.com/streamingfast/dstore"
	"github.com/streamingfast/shutter"
)

// blockingSource never emits blocks and never terminates on its own.
type blockingSource struct{ *shutter.Shutter }

func (s *blockingSource) Run() { <-s.Terminating() }

// Reproduces the 2026-10-26 Sui outage: the relayer hub shut down before it
// ever became ready, and Relayer.Run() stayed blocked on <-r.hub.Ready
// forever, leaving the process alive with no :10014 listener.
func TestRelayerTerminatesWhenHubShutsDownBeforeReady(t *testing.T) {
	src := func(h bstream.Handler) bstream.Source {
		return &blockingSource{Shutter: shutter.New()}
	}
	r := NewRelayer(src, "localhost:0", dstore.NewMockStore(nil))

	go r.Run()
	time.Sleep(200 * time.Millisecond)

	r.hub.Shutdown(errors.New("cannot link block after reconnection, restart required"))

	select {
	case <-r.Terminating():
		// expected
	case <-time.After(3 * time.Second):
		t.Fatal("relayer did not terminate after hub shutdown before ready")
	}
}
