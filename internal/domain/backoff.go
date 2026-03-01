package domain

import "time"

func (c BackoffConfig) DelayForAttempt(attempt int, jitter time.Duration) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	delay := c.InitialDelay
	for i := 1; i < attempt; i++ {
		delay = delay * time.Duration(c.Multiplier)
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > c.MaxJitter {
		jitter = c.MaxJitter
	}
	delay += jitter
	if delay > c.MaxDelay {
		return c.MaxDelay
	}
	return delay
}
