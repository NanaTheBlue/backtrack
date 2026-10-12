package retry

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Do executes an operation with exponential backoff until it succeeds,
// maxAttempts is reached, or ctx is cancelled.
func Do(ctx context.Context, maxAttempts int, initialBackoff time.Duration, op func() error) error {
	backoff := initialBackoff
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = op()
		if lastErr == nil {
			return nil
		}

		log.Printf("attempt %d/%d failed: %v. Retrying in %v...", attempt, maxAttempts, lastErr, backoff)

		if attempt == maxAttempts {
			break
		}
		select {
		case <-time.After(backoff):
			backoff *= 2
		case <-ctx.Done():
			return fmt.Errorf("retry aborted by context: %w", ctx.Err())
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, lastErr)
}
