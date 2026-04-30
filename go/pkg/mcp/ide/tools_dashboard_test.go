package ide

import "testing"

func TestToolsDashboard_PublicTypes(t *testing.T) {
	metrics := DashboardMetrics{BuildsTotal: 2, BuildsSuccess: 1, AgentSessions: 1}
	outMetrics := DashboardMetricsOutput{Period: "24h", Metrics: metrics}
	overview := DashboardOverview{Repos: 2, ActiveSessions: 1, BridgeOnline: true}
	out := DashboardOverviewOutput{Overview: overview}
	if out.Overview.Repos != 2 || outMetrics.Metrics.BuildsTotal != 2 {
		t.Fatalf("unexpected dashboard outputs: %+v %+v", out, outMetrics)
	}
}
