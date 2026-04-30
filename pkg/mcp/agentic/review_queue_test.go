package agentic

import (
	"testing"
	"time"
)

func TestReviewQueue_PublicTypes(t *testing.T) {
	result := ReviewResult{Repo: "go-mcp", Verdict: "clean", Action: "waiting"}
	rate := &RateLimitInfo{Limited: true, RetryAt: time.Now().Add(time.Minute), Message: "retry"}
	out := ReviewQueueOutput{Success: true, Processed: []ReviewResult{result}, RateLimit: rate}
	if out.Processed[0].Verdict != "clean" || out.RateLimit.Message != "retry" {
		t.Fatalf("unexpected review queue output: %+v", out)
	}
}
