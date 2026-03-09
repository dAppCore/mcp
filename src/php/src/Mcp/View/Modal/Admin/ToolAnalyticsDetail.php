<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mcp\DTO\ToolStats;
use Core\Mcp\Services\ToolAnalyticsService;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Url;
use Livewire\Component;

/**
 * Tool Analytics Detail - detailed view for a single MCP tool.
 *
 * Shows usage trends, performance metrics, and error details for a specific tool.
 */
#[Layout('hub::admin.layouts.app')]
class ToolAnalyticsDetail extends Component
{
    /**
     * The tool name to display.
     */
    public string $toolName;

    /**
     * Number of days to show in analytics.
     */
    #[Url]
    public int $days = 30;

    /**
     * The analytics service.
     */
    protected ToolAnalyticsService $analyticsService;

    public function mount(string $name): void
    {
        $this->toolName = $name;
    }

    public function boot(ToolAnalyticsService $analyticsService): void
    {
        $this->analyticsService = $analyticsService;
    }

    /**
     * Set the number of days to display.
     */
    public function setDays(int $days): void
    {
        $this->days = max(1, min(90, $days));
    }

    /**
     * Get the tool statistics.
     */
    public function getStatsProperty(): ToolStats
    {
        $from = now()->subDays($this->days - 1)->startOfDay();
        $to = now()->endOfDay();

        return app(ToolAnalyticsService::class)->getToolStats($this->toolName, $from, $to);
    }

    /**
     * Get usage trends for the tool.
     */
    public function getTrendsProperty(): array
    {
        return app(ToolAnalyticsService::class)->getUsageTrends($this->toolName, $this->days);
    }

    /**
     * Get chart data for the usage trend line chart.
     */
    public function getTrendChartDataProperty(): array
    {
        $trends = $this->getTrendsProperty();

        return [
            'labels' => array_column($trends, 'date_formatted'),
            'calls' => array_column($trends, 'calls'),
            'errors' => array_column($trends, 'errors'),
            'avgDuration' => array_column($trends, 'avg_duration_ms'),
        ];
    }

    /**
     * Format duration for display.
     */
    public function formatDuration(float $ms): string
    {
        if ($ms === 0.0) {
            return '-';
        }

        if ($ms < 1000) {
            return round($ms).'ms';
        }

        return round($ms / 1000, 2).'s';
    }

    public function render()
    {
        return view('mcp::admin.analytics.tool-detail');
    }
}
