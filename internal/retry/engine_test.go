package retry

import (
	"testing"
	"time"

	"github.com/eventflow/eventflow/pkg/config"
)

func TestBackoff(t *testing.T) {
	e := &Engine{policy: config.RetryPolicy{
		MaxAttempts: 5, InitialBackoff: time.Second,
		MaxBackoff: 30 * time.Second, Multiplier: 2,
	}}
	d0 := e.backoff(0)
	if d0 != time.Second {
		t.Fatalf("backoff(0)=%v want 1s", d0)
	}
	d2 := e.backoff(2)
	if d2 != 4*time.Second {
		t.Fatalf("backoff(2)=%v want 4s", d2)
	}
	d10 := e.backoff(10)
	if d10 != 30*time.Second {
		t.Fatalf("backoff(10)=%v want capped 30s", d10)
	}
}
