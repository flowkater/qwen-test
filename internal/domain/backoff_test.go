package domain

import (
	"testing"
	"time"
)

func TestBackoffDelayGrowthJitterCap(t *testing.T) {
	cfg := DefaultBackoffConfig()

	d1 := cfg.DelayForAttempt(1, 0)
	d2 := cfg.DelayForAttempt(2, 0)
	d3 := cfg.DelayForAttempt(3, 0)
	if !(d1 < d2 && d2 < d3) {
		t.Fatalf("delay must increase: %v %v %v", d1, d2, d3)
	}

	withJitter := cfg.DelayForAttempt(1, 500*time.Millisecond)
	if withJitter != 2500*time.Millisecond {
		t.Fatalf("unexpected jitter delay: %v", withJitter)
	}

	capped := cfg.DelayForAttempt(10, time.Second)
	if capped > 30*time.Second {
		t.Fatalf("must cap delay at 30s, got %v", capped)
	}
}
