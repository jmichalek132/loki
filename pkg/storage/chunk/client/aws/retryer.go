package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/grafana/dskit/backoff"
	attribute "go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Map Cortex Backoff into AWS Retryer interface
type retryer struct {
	*backoff.Backoff
	maxRetries int
}

var _ request.Retryer = &retryer{}

func newRetryer(ctx context.Context, cfg backoff.Config) *retryer {
	return &retryer{
		Backoff:    backoff.New(ctx, cfg),
		maxRetries: cfg.MaxRetries,
	}
}

func (r *retryer) withRetries(req *request.Request) {
	req.Retryer = r
}

// RetryRules return the retry delay that should be used by the SDK before
// making another request attempt for the failed request.
func (r *retryer) RetryRules(req *request.Request) time.Duration {
	duration := r.Backoff.NextDelay()
	trace.SpanFromContext(req.Context()).SetAttributes(attribute.Int("retry", r.NumRetries()))
	return duration
}

// ShouldRetry returns if the failed request is retryable.
func (r *retryer) ShouldRetry(req *request.Request) bool {
	return r.Ongoing() && (req.IsErrorRetryable() || req.IsErrorThrottle())
}

// MaxRetries is the number of times a request may be retried before
// failing.
func (r *retryer) MaxRetries() int {
	return r.maxRetries
}
