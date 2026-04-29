package agentic

import core "dappco.re/go"

func ExampleReviewQueueInput() {
	var _ ReviewQueueInput
	core.Println("ReviewQueueInput")
	// Output: ReviewQueueInput
}

func ExampleReviewQueueOutput() {
	var _ ReviewQueueOutput
	core.Println("ReviewQueueOutput")
	// Output: ReviewQueueOutput
}

func ExampleReviewResult() {
	var _ ReviewResult
	core.Println("ReviewResult")
	// Output: ReviewResult
}

func ExampleRateLimitInfo() {
	var _ RateLimitInfo
	core.Println("RateLimitInfo")
	// Output: RateLimitInfo
}
