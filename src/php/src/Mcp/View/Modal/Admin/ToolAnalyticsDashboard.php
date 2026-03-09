<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mcp\DTO\ToolStats;
use Core\Mcp\Services\ToolAnalyticsService;
use Illuminate\Support\Collection;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Url;
use Livewire\Component;

/**
 * Tool Analytics Dashboard - admin dashboard for MCP tool usage analytics.
 *
 * Displays overview cards, charts, and tables for monitoring tool usage patterns.
 */
#[Layout('hub::admin.layouts.app')]
class ToolAnalyticsDashboard extends Component
{
    /**
     * Number of days to show in analytics.
     */
    #[Url]
    public int $days = 30;

    /**
     * Currently selected tab.
     */
    #[Url]
    public string $tab = 'overview';

    /**
     * Workspace filter (null = all workspaces).
     */
    #[Url]
    public ?string $workspaceId = null;

    /**
     * Sort column for the tools table.
     */
    public string $sortColumn = 'totalCalls';

    /**
     * Sort direction for the tools table.
     */
    public string $sortDirection = 'desc';

    /**
     * The analytics service.
     */
    protected ToolAnalyticsService $analyticsService;

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
     * Set the active tab.
     */
    public function setTab(string $tab): void
    {
        $this->tab = $tab;
    }

    /**
     * Set the sort column and direction.
     */
    public function sort(string $column): void
    {
        if ($this->sortColumn === $column) {
            $this->sortDirection = $this->sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            $this->sortColumn = $column;
            $this->sortDirection = 'desc';
        }
    }

    /**
     * Set the workspace filter.
     */
    public function setWorkspace(?string $workspaceId): void
    {
        $this->workspaceId = $workspaceId;
    }

    /**
     * Get the date range.
     */
    protected function getDateRange(): array
    {
        return [
            'from' => now()->subDays($this->days - 1)->startOfDay(),
            'to' => now()->endOfDay(),
        ];
    }

    /**
     * Get overview statistics.
     */
    public function getOverviewProperty(): array
    {
        $range = $this->getDateRange();
        $stats = $this->getAllToolsProperty();

        $totalCalls = $stats->sum(fn (ToolStats $s) => $s->totalCalls);
        $totalErrors = $stats->sum(fn (ToolStats $s) => $s->errorCount);
        $avgDuration = $totalCalls > 0
            ? $stats->sum(fn (ToolStats $s) => $s->avgDurationMs * $s->totalCalls) / $totalCalls
            : 0;

        return [
            'total_calls' => $totalCalls,
            'total_errors' => $totalErrors,
            'error_rate' => $totalCalls > 0 ? round(($totalErrors / $totalCalls) * 100, 2) : 0,
            'avg_duration_ms' => round($avgDuration, 2),
            'unique_tools' => $stats->count(),
        ];
    }

    /**
     * Get all tool statistics.
     */
    public function getAllToolsProperty(): Collection
    {
        $range = $this->getDateRange();

        return app(ToolAnalyticsService::class)->getAllToolStats($range['from'], $range['to']);
    }

    /**
     * Get sorted tool statistics for the table.
     */
    public function getSortedToolsProperty(): Collection
    {
        $tools = $this->getAllToolsProperty();

        return $tools->sortBy(
            fn (ToolStats $s) => match ($this->sortColumn) {
                'toolName' => $s->toolName,
                'totalCalls' => $s->totalCalls,
                'errorCount' => $s->errorCount,
                'errorRate' => $s->errorRate,
                'avgDurationMs' => $s->avgDurationMs,
                default => $s->totalCalls,
            },
            SORT_REGULAR,
            $this->sortDirection === 'desc'
        )->values();
    }

    /**
     * Get the most popular tools.
     */
    public function getPopularToolsProperty(): Collection
    {
        $range = $this->getDateRange();

        return app(ToolAnalyticsService::class)->getPopularTools(10, $range['from'], $range['to']);
    }

    /**
     * Get tools with high error rates.
     */
    public function getErrorProneToolsProperty(): Collection
    {
        $range = $this->getDateRange();

        return app(ToolAnalyticsService::class)->getErrorProneTools(10, $range['from'], $range['to']);
    }

    /**
     * Get tool combinations.
     */
    public function getToolCombinationsProperty(): Collection
    {
        $range = $this->getDateRange();

        return app(ToolAnalyticsService::class)->getToolCombinations(10, $range['from'], $range['to']);
    }

    /**
     * Get daily trends for charting.
     */
    public function getDailyTrendsProperty(): array
    {
        $range = $this->getDateRange();
        $allStats = $this->getAllToolsProperty();

        // Aggregate daily data
        $dailyData = [];
        for ($i = $this->days - 1; $i >= 0; $i--) {
            $date = now()->subDays($i);
            $dailyData[] = [
                'date' => $date->toDateString(),
                'date_formatted' => $date->format('M j'),
                'calls' => 0, // Would need per-day aggregation
                'errors' => 0,
            ];
        }

        return $dailyData;
    }

    /**
     * Get chart data for the top tools bar chart.
     */
    public function getTopToolsChartDataProperty(): array
    {
        $tools = $this->getPopularToolsProperty()->take(10);

        return [
            'labels' => $tools->pluck('toolName')->toArray(),
            'data' => $tools->pluck('totalCalls')->toArray(),
            'colors' => $tools->map(fn (ToolStats $t) => $t->errorRate > 10 ? '#ef4444' : '#3b82f6')->toArray(),
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
        return view('mcp::admin.analytics.dashboard');
    }
}
