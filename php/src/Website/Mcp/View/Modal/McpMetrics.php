<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Livewire\Attributes\Layout;
use Livewire\Component;
use Core\Mcp\Services\McpMetricsService;

/**
 * MCP Metrics Dashboard
 *
 * Displays analytics and metrics for MCP tool usage.
 */
#[Layout('layouts::mcp')]
class McpMetrics extends Component
{
    public int $days = 7;

    public string $activeTab = 'overview';

    protected McpMetricsService $metricsService;

    public function boot(McpMetricsService $metricsService): void
    {
        $this->metricsService = $metricsService;
    }

    public function setDays(int $days): void
    {
        // Bound days to a reasonable range (1-90)
        $this->days = min(max($days, 1), 90);
    }

    public function setTab(string $tab): void
    {
        $this->activeTab = $tab;
    }

    public function getOverviewProperty(): array
    {
        return app(McpMetricsService::class)->getOverview($this->days);
    }

    public function getDailyTrendProperty(): array
    {
        return app(McpMetricsService::class)->getDailyTrend($this->days);
    }

    public function getTopToolsProperty(): array
    {
        return app(McpMetricsService::class)->getTopTools($this->days, 10);
    }

    public function getServerStatsProperty(): array
    {
        return app(McpMetricsService::class)->getServerStats($this->days);
    }

    public function getRecentCallsProperty(): array
    {
        return app(McpMetricsService::class)->getRecentCalls(20);
    }

    public function getErrorBreakdownProperty(): array
    {
        return app(McpMetricsService::class)->getErrorBreakdown($this->days);
    }

    public function getToolPerformanceProperty(): array
    {
        return app(McpMetricsService::class)->getToolPerformance($this->days, 10);
    }

    public function getHourlyDistributionProperty(): array
    {
        return app(McpMetricsService::class)->getHourlyDistribution();
    }

    public function getPlanActivityProperty(): array
    {
        return app(McpMetricsService::class)->getPlanActivity($this->days, 10);
    }

    public function render()
    {
        return view('mcp::web.mcp-metrics');
    }
}
