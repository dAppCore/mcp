package mcp

import core "dappco.re/go"

func ExampleMetricsRecordInput() {
	var _ MetricsRecordInput
	core.Println("MetricsRecordInput")
	// Output: MetricsRecordInput
}

func ExampleMetricsRecordOutput() {
	var _ MetricsRecordOutput
	core.Println("MetricsRecordOutput")
	// Output: MetricsRecordOutput
}

func ExampleMetricsQueryInput() {
	var _ MetricsQueryInput
	core.Println("MetricsQueryInput")
	// Output: MetricsQueryInput
}

func ExampleMetricsQueryOutput() {
	var _ MetricsQueryOutput
	core.Println("MetricsQueryOutput")
	// Output: MetricsQueryOutput
}

func ExampleMetricCount() {
	var _ MetricCount
	core.Println("MetricCount")
	// Output: MetricCount
}

func ExampleMetricEventBrief() {
	var _ MetricEventBrief
	core.Println("MetricEventBrief")
	// Output: MetricEventBrief
}
