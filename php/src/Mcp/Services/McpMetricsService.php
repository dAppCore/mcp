<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Models\McpToolCall;
use Core\Mcp\Models\McpToolCallStat;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;

/**
 * MCP Metrics Service - dashboard metrics for MCP tool usage.
 *
 * Provides overview stats, trends, performance percentiles, and activity feeds.
 */
class McpMetricsService
{
    /**
     * Get overview metrics for the dashboard.
     */
    public function getOverview(int $days = 7): array
    {
        $startDate = now()->subDays($days - 1)->startOfDay();

        $stats = McpToolCallStat::forDateRange($startDate, now())->get();

        $totalCalls = $stats->sum('call_count');
        $successCalls = $stats->sum('success_count');
        $errorCalls = $stats->sum('error_count');

        $successRate = $totalCalls > 0
            ? round(($successCalls / $totalCalls) * 100, 1)
            : 0;

        $avgDuration = $totalCalls > 0
            ? round($stats->sum('total_duration_ms') / $totalCalls, 1)
            : 0;

        // Compare to previous period
        $previousStart = $startDate->copy()->subDays($days);
        $previousStats = McpToolCallStat::forDateRange($previousStart, $startDate->copy()->subDay())->get();
        $previousCalls = $previousStats->sum('call_count');

        $callsTrend = $previousCalls > 0
            ? round((($totalCalls - $previousCalls) / $previousCalls) * 100, 1)
            : 0;

        return [
            'total_calls' => $totalCalls,
            'success_calls' => $successCalls,
            'error_calls' => $errorCalls,
            'success_rate' => $successRate,
            'avg_duration_ms' => $avgDuration,
            'calls_trend_percent' => $callsTrend,
            'unique_tools' => $stats->pluck('tool_name')->unique()->count(),
            'unique_servers' => $stats->pluck('server_id')->unique()->count(),
            'period_days' => $days,
        ];
    }

    /**
     * Get daily call trend data for charting.
     */
    public function getDailyTrend(int $days = 7): Collection
    {
        $trend = McpToolCallStat::getDailyTrend($days);

        // Fill in missing dates with zeros
        $dates = collect();
        for ($i = $days - 1; $i >= 0; $i--) {
            $date = now()->subDays($i)->toDateString();
            $existing = $trend->firstWhere('date', $date);

            $dates->push([
                'date' => $date,
                'date_formatted' => Carbon::parse($date)->format('M j'),
                'total_calls' => $existing->total_calls ?? 0,
                'total_success' => $existing->total_success ?? 0,
                'total_errors' => $existing->total_errors ?? 0,
                'success_rate' => $existing->success_rate ?? 0,
            ]);
        }

        return $dates;
    }

    /**
     * Get top tools by call count.
     */
    public function getTopTools(int $days = 7, int $limit = 10): Collection
    {
        return McpToolCallStat::getTopTools($days, $limit);
    }

    /**
     * Get server breakdown.
     */
    public function getServerStats(int $days = 7): Collection
    {
        return McpToolCallStat::getServerStats($days);
    }

    /**
     * Get recent tool calls for activity feed.
     */
    public function getRecentCalls(int $limit = 20): Collection
    {
        return McpToolCall::query()
            ->orderByDesc('created_at')
            ->limit($limit)
            ->get()
            ->map(function ($call) {
                return [
                    'id' => $call->id,
                    'server_id' => $call->server_id,
                    'tool_name' => $call->tool_name,
                    'success' => $call->success,
                    'duration' => $call->getDurationForHumans(),
                    'duration_ms' => $call->duration_ms,
                    'error_message' => $call->error_message,
                    'session_id' => $call->session_id,
                    'plan_slug' => $call->plan_slug,
                    'created_at' => $call->created_at->diffForHumans(),
                    'created_at_full' => $call->created_at->toIso8601String(),
                ];
            });
    }

    /**
     * Get error breakdown.
     */
    public function getErrorBreakdown(int $days = 7): Collection
    {
        return McpToolCall::query()
            ->select('tool_name', 'error_code')
            ->selectRaw('COUNT(*) as error_count')
            ->where('success', false)
            ->where('created_at', '>=', now()->subDays($days))
            ->groupBy('tool_name', 'error_code')
            ->orderByDesc('error_count')
            ->limit(20)
            ->get();
    }

    /**
     * Get tool performance metrics (p50, p95, p99).
     */
    public function getToolPerformance(int $days = 7, int $limit = 10): Collection
    {
        // Get raw call data for percentile calculations
        $calls = McpToolCall::query()
            ->select('tool_name', 'duration_ms')
            ->whereNotNull('duration_ms')
            ->where('success', true)
            ->where('created_at', '>=', now()->subDays($days))
            ->get()
            ->groupBy('tool_name');

        $performance = collect();

        foreach ($calls as $toolName => $toolCalls) {
            $durations = $toolCalls->pluck('duration_ms')->sort()->values();
            $count = $durations->count();

            if ($count === 0) {
                continue;
            }

            $performance->push([
                'tool_name' => $toolName,
                'call_count' => $count,
                'min_ms' => $durations->first(),
                'max_ms' => $durations->last(),
                'avg_ms' => round($durations->avg(), 1),
                'p50_ms' => $this->percentile($durations, 50),
                'p95_ms' => $this->percentile($durations, 95),
                'p99_ms' => $this->percentile($durations, 99),
            ]);
        }

        return $performance
            ->sortByDesc('call_count')
            ->take($limit)
            ->values();
    }

    /**
     * Get hourly distribution for the last 24 hours.
     */
    public function getHourlyDistribution(): Collection
    {
        $hourly = McpToolCall::query()
            ->selectRaw('HOUR(created_at) as hour')
            ->selectRaw('COUNT(*) as call_count')
            ->selectRaw('SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count')
            ->where('created_at', '>=', now()->subHours(24))
            ->groupBy('hour')
            ->orderBy('hour')
            ->get()
            ->keyBy('hour');

        // Fill in missing hours
        $result = collect();
        for ($i = 0; $i < 24; $i++) {
            $hour = str_pad((string) $i, 2, '0', STR_PAD_LEFT);
            $existing = $hourly->get($i);

            $result->push([
                'hour' => $hour,
                'hour_formatted' => Carbon::createFromTime($i)->format('ga'),
                'call_count' => $existing->call_count ?? 0,
                'success_count' => $existing->success_count ?? 0,
            ]);
        }

        return $result;
    }

    /**
     * Get plan activity - which plans are using MCP tools.
     */
    public function getPlanActivity(int $days = 7, int $limit = 10): Collection
    {
        return McpToolCall::query()
            ->select('plan_slug')
            ->selectRaw('COUNT(*) as call_count')
            ->selectRaw('COUNT(DISTINCT tool_name) as unique_tools')
            ->selectRaw('SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count')
            ->whereNotNull('plan_slug')
            ->where('created_at', '>=', now()->subDays($days))
            ->groupBy('plan_slug')
            ->orderByDesc('call_count')
            ->limit($limit)
            ->get()
            ->map(function ($item) {
                $item->success_rate = $item->call_count > 0
                    ? round(($item->success_count / $item->call_count) * 100, 1)
                    : 0;

                return $item;
            });
    }

    /**
     * Calculate percentile from a sorted collection.
     */
    protected function percentile(Collection $sortedValues, int $percentile): float
    {
        $count = $sortedValues->count();
        if ($count === 0) {
            return 0;
        }

        $index = ($percentile / 100) * ($count - 1);
        $lower = (int) floor($index);
        $upper = (int) ceil($index);

        if ($lower === $upper) {
            return $sortedValues[$lower];
        }

        $fraction = $index - $lower;

        return round($sortedValues[$lower] + ($sortedValues[$upper] - $sortedValues[$lower]) * $fraction, 1);
    }
}
