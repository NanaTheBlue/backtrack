package retry

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Do executes an operation with exponential backoff until it succeeds,
// maxAttempts is reached, or ctx is cancelled.
func Do(ctx context.Context, maxAttempts int, initialBackoff time.Duration, op func() error) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	backoff := initialBackoff
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = op()
		if lastErr == nil {
			return nil
		}

		slog.Warn("operation attempt failed, retrying",
			"attempt", attempt,
			"max_attempts", maxAttempts,
			"backoff", backoff.String(),
			"error", lastErr,
		)

		if attempt == maxAttempts {
			break
		}

		timer := time.NewTimer(backoff)
		select {
		case <-timer.C:
			backoff *= 2
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("retry aborted by context: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, lastErr)
}
