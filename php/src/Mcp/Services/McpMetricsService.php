<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Carbon\CarbonImmutable;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

/**
 * Aggregate MCP tool activity into dashboard-friendly metrics.
 *
 * @example
 * $service = new McpMetricsService();
 * $overview = $service->getOverview(7);
 */
final class McpMetricsService
{
    protected string $statsTable = 'mcp_tool_call_stats';

    protected string $callsTable = 'mcp_tool_calls';

    /**
     * Summarise tool activity for the requested reporting window.
     *
     * @example
     * $overview = $service->getOverview(14);
     */
    public function getOverview(int $days = 7): array
    {
        $currentStart = CarbonImmutable::now()->subDays($days - 1)->startOfDay();
        $currentEnd = CarbonImmutable::now()->endOfDay();
        $previousStart = $currentStart->subDays($days);
        $previousEnd = $currentStart->subDay()->endOfDay();

        $current = $this->statsInRange($currentStart, $currentEnd);
        $previous = $this->statsInRange($previousStart, $previousEnd);

        $totalCalls = (int) $current->sum('call_count');
        $successCalls = (int) $current->sum('success_count');
        $errorCalls = (int) $current->sum('error_count');
        $previousCalls = (int) $previous->sum('call_count');
        $totalDuration = (float) $current->sum('total_duration_ms');

        return [
            'total_calls' => $totalCalls,
            'success_calls' => $successCalls,
            'error_calls' => $errorCalls,
            'success_rate' => $totalCalls > 0 ? round(($successCalls / $totalCalls) * 100, 1) : 0.0,
            'avg_duration_ms' => $totalCalls > 0 ? round($totalDuration / $totalCalls, 1) : 0.0,
            'calls_trend_percent' => $previousCalls > 0 ? round((($totalCalls - $previousCalls) / $previousCalls) * 100, 1) : 0.0,
            'unique_tools' => $current->pluck('tool_name')->filter()->unique()->count(),
            'unique_servers' => $current->pluck('server_id')->filter()->unique()->count(),
            'period_days' => $days,
        ];
    }

    /**
     * Return the per-day call trend for the requested reporting window.
     *
     * @example
     * $trend = $service->getDailyTrend(7);
     */
    public function getDailyTrend(int $days = 7): Collection
    {
        $result = collect();
        $start = CarbonImmutable::now()->subDays($days - 1)->startOfDay();
        $end = CarbonImmutable::now()->endOfDay();
        $rows = $this->dailyTrendRows($start, $end)->keyBy('date');

        for ($offset = 0; $offset < $days; $offset++) {
            $date = $start->addDays($offset)->toDateString();
            $row = $rows->get($date);
            $totalCalls = (int) ($row->total_calls ?? 0);
            $totalSuccess = (int) ($row->total_success ?? 0);
            $totalErrors = (int) ($row->total_errors ?? 0);

            $result->push(collect([
                'date' => $date,
                'date_formatted' => Carbon::parse($date)->format('M j'),
                'total_calls' => $totalCalls,
                'total_success' => $totalSuccess,
                'total_errors' => $totalErrors,
                'success_rate' => $totalCalls > 0 ? round(($totalSuccess / $totalCalls) * 100, 1) : 0.0,
            ]));
        }

        return $result;
    }

    /**
     * Return the most-used tools in the requested reporting window.
     *
     * @example
     * $tools = $service->getTopTools(7, 5);
     */
    public function getTopTools(int $days = 7, int $limit = 10): Collection
    {
        if (! Schema::hasTable($this->statsTable)) {
            return collect();
        }

        return DB::table($this->statsTable)
            ->select('tool_name')
            ->selectRaw('SUM(call_count) as call_count')
            ->selectRaw('SUM(success_count) as success_count')
            ->selectRaw('SUM(error_count) as error_count')
            ->selectRaw('SUM(total_duration_ms) as total_duration_ms')
            ->whereBetween('date', [
                CarbonImmutable::now()->subDays($days - 1)->toDateString(),
                CarbonImmutable::now()->toDateString(),
            ])
            ->groupBy('tool_name')
            ->orderByDesc('call_count')
            ->limit($limit)
            ->get()
            ->map(function (object $row): Collection {
                $callCount = (int) $row->call_count;
                $successCount = (int) $row->success_count;

                return collect([
                    'tool_name' => (string) $row->tool_name,
                    'call_count' => $callCount,
                    'success_count' => $successCount,
                    'error_count' => (int) $row->error_count,
                    'success_rate' => $callCount > 0 ? round(($successCount / $callCount) * 100, 1) : 0.0,
                    'avg_duration_ms' => $callCount > 0 ? round(((int) $row->total_duration_ms) / $callCount, 1) : 0.0,
                ]);
            });
    }

    /**
     * Return aggregated call statistics grouped by MCP server.
     *
     * @example
     * $servers = $service->getServerStats(30);
     */
    public function getServerStats(int $days = 7): Collection
    {
        if (! Schema::hasTable($this->statsTable)) {
            return collect();
        }

        return DB::table($this->statsTable)
            ->select('server_id')
            ->selectRaw('SUM(call_count) as call_count')
            ->selectRaw('SUM(success_count) as success_count')
            ->selectRaw('SUM(error_count) as error_count')
            ->selectRaw('COUNT(DISTINCT tool_name) as unique_tools')
            ->whereBetween('date', [
                CarbonImmutable::now()->subDays($days - 1)->toDateString(),
                CarbonImmutable::now()->toDateString(),
            ])
            ->groupBy('server_id')
            ->orderByDesc('call_count')
            ->get()
            ->map(function (object $row): Collection {
                $callCount = (int) $row->call_count;
                $successCount = (int) $row->success_count;

                return collect([
                    'server_id' => (string) $row->server_id,
                    'call_count' => $callCount,
                    'success_count' => $successCount,
                    'error_count' => (int) $row->error_count,
                    'unique_tools' => (int) $row->unique_tools,
                    'success_rate' => $callCount > 0 ? round(($successCount / $callCount) * 100, 1) : 0.0,
                ]);
            });
    }

    /**
     * Return the latest recorded MCP calls for dashboard activity feeds.
     *
     * @example
     * $calls = $service->getRecentCalls(20);
     */
    public function getRecentCalls(int $limit = 20): Collection
    {
        if (! Schema::hasTable($this->callsTable)) {
            return collect();
        }

        return DB::table($this->callsTable)
            ->orderByDesc('created_at')
            ->limit($limit)
            ->get()
            ->map(function (object $row): Collection {
                $createdAt = isset($row->created_at) ? Carbon::parse((string) $row->created_at) : null;
                $durationMs = isset($row->duration_ms) ? (int) $row->duration_ms : null;

                return collect([
                    'id' => $row->id,
                    'server_id' => (string) ($row->server_id ?? ''),
                    'tool_name' => (string) ($row->tool_name ?? ''),
                    'success' => (bool) ($row->success ?? false),
                    'duration' => $this->humanDuration($durationMs),
                    'duration_ms' => $durationMs,
                    'error_message' => $row->error_message,
                    'session_id' => $row->session_id,
                    'plan_slug' => $row->plan_slug,
                    'created_at' => $createdAt?->diffForHumans(),
                    'created_at_full' => $createdAt?->toIso8601String(),
                ]);
            });
    }

    /**
     * Group recent tool failures by tool name and error code.
     *
     * @example
     * $errors = $service->getErrorBreakdown(7);
     */
    public function getErrorBreakdown(int $days = 7): Collection
    {
        if (! Schema::hasTable($this->callsTable)) {
            return collect();
        }

        return DB::table($this->callsTable)
            ->select('tool_name', 'error_code')
            ->selectRaw('COUNT(*) as error_count')
            ->where('success', false)
            ->where('created_at', '>=', CarbonImmutable::now()->subDays($days - 1)->startOfDay()->toDateTimeString())
            ->groupBy('tool_name', 'error_code')
            ->orderByDesc('error_count')
            ->get()
            ->map(fn (object $row): Collection => collect([
                'tool_name' => (string) ($row->tool_name ?? ''),
                'error_code' => $row->error_code,
                'error_count' => (int) $row->error_count,
            ]));
    }

    /**
     * Calculate duration percentiles for successful tool calls.
     *
     * @example
     * $performance = $service->getToolPerformance(7, 10);
     */
    public function getToolPerformance(int $days = 7, int $limit = 10): Collection
    {
        if (! Schema::hasTable($this->callsTable)) {
            return collect();
        }

        $rows = DB::table($this->callsTable)
            ->select('tool_name', 'duration_ms')
            ->whereNotNull('duration_ms')
            ->where('success', true)
            ->where('created_at', '>=', CarbonImmutable::now()->subDays($days - 1)->startOfDay()->toDateTimeString())
            ->get()
            ->groupBy('tool_name');

        return $rows->map(function (Collection $items, string $toolName): Collection {
            $durations = $items->pluck('duration_ms')->map(static fn (mixed $value): int => (int) $value)->sort()->values();
            $count = $durations->count();

            return collect([
                'tool_name' => $toolName,
                'call_count' => $count,
                'min_ms' => $count > 0 ? (int) $durations->first() : 0,
                'max_ms' => $count > 0 ? (int) $durations->last() : 0,
                'avg_ms' => $count > 0 ? round($durations->avg(), 1) : 0.0,
                'p50_ms' => $this->percentile($durations, 50),
                'p95_ms' => $this->percentile($durations, 95),
                'p99_ms' => $this->percentile($durations, 99),
            ]);
        })->sortByDesc('call_count')->take($limit)->values();
    }

    /**
     * Bucket recent tool calls by hour for the last 24 hours.
     *
     * @example
     * $hours = $service->getHourlyDistribution();
     */
    public function getHourlyDistribution(): Collection
    {
        $distribution = collect();
        $hours = collect();

        if (Schema::hasTable($this->callsTable)) {
            $hours = DB::table($this->callsTable)
                ->select('success', 'created_at')
                ->where('created_at', '>=', CarbonImmutable::now()->subHours(24)->toDateTimeString())
                ->get()
                ->groupBy(static function (object $row): string {
                    return Carbon::parse((string) $row->created_at)->format('H');
                });
        }

        for ($hour = 0; $hour < 24; $hour++) {
            $key = str_pad((string) $hour, 2, '0', STR_PAD_LEFT);
            $rows = $hours->get($key, collect());

            $distribution->push(collect([
                'hour' => $key,
                'hour_formatted' => Carbon::createFromTime($hour)->format('ga'),
                'call_count' => $rows->count(),
                'success_count' => $rows->filter(static fn (object $row): bool => (bool) ($row->success ?? false))->count(),
            ]));
        }

        return $distribution;
    }

    /**
     * Return activity totals grouped by plan slug for the requested window.
     *
     * @example
     * $plans = $service->getPlanActivity(7, 10);
     */
    public function getPlanActivity(int $days = 7, int $limit = 10): Collection
    {
        if (! Schema::hasTable($this->callsTable)) {
            return collect();
        }

        return DB::table($this->callsTable)
            ->select('plan_slug')
            ->selectRaw('COUNT(*) as call_count')
            ->selectRaw('COUNT(DISTINCT tool_name) as unique_tools')
            ->selectRaw('SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count')
            ->whereNotNull('plan_slug')
            ->where('created_at', '>=', CarbonImmutable::now()->subDays($days - 1)->startOfDay()->toDateTimeString())
            ->groupBy('plan_slug')
            ->orderByDesc('call_count')
            ->limit($limit)
            ->get()
            ->map(function (object $row): Collection {
                $callCount = (int) $row->call_count;
                $successCount = (int) $row->success_count;

                return collect([
                    'plan_slug' => (string) $row->plan_slug,
                    'call_count' => $callCount,
                    'unique_tools' => (int) $row->unique_tools,
                    'success_count' => $successCount,
                    'success_rate' => $callCount > 0 ? round(($successCount / $callCount) * 100, 1) : 0.0,
                ]);
            });
    }

    /**
     * Load raw stats rows for the supplied date range.
     *
     * @example
     * $rows = $this->statsInRange(CarbonImmutable::now()->subDays(6)->startOfDay(), CarbonImmutable::now()->endOfDay());
     */
    protected function statsInRange(CarbonImmutable $start, CarbonImmutable $end): Collection
    {
        if (! Schema::hasTable($this->statsTable)) {
            return collect();
        }

        return DB::table($this->statsTable)
            ->whereBetween('date', [$start->toDateString(), $end->toDateString()])
            ->get()
            ->map(static fn (object $row): Collection => collect((array) $row));
    }

    /**
     * Load grouped daily totals for the supplied date range.
     *
     * @example
     * $rows = $this->dailyTrendRows(CarbonImmutable::now()->subDays(6)->startOfDay(), CarbonImmutable::now()->endOfDay());
     */
    protected function dailyTrendRows(CarbonImmutable $start, CarbonImmutable $end): Collection
    {
        if (! Schema::hasTable($this->statsTable)) {
            return collect();
        }

        return DB::table($this->statsTable)
            ->select('date')
            ->selectRaw('SUM(call_count) as total_calls')
            ->selectRaw('SUM(success_count) as total_success')
            ->selectRaw('SUM(error_count) as total_errors')
            ->whereBetween('date', [$start->toDateString(), $end->toDateString()])
            ->groupBy('date')
            ->orderBy('date')
            ->get();
    }

    /**
     * Interpolate a percentile value from an ascending list of durations.
     *
     * @example
     * $p95 = $this->percentile(collect([10, 25, 40, 90]), 95);
     */
    protected function percentile(Collection $sortedValues, int $percentile): float
    {
        $count = $sortedValues->count();

        if ($count === 0) {
            return 0.0;
        }

        $index = ($percentile / 100) * ($count - 1);
        $lower = (int) floor($index);
        $upper = (int) ceil($index);

        if ($lower === $upper) {
            return (float) $sortedValues[$lower];
        }

        $fraction = $index - $lower;

        return round(((float) $sortedValues[$lower]) + (((float) $sortedValues[$upper] - (float) $sortedValues[$lower]) * $fraction), 1);
    }

    /**
     * Render a millisecond duration using the dashboard display format.
     *
     * @example
     * $label = $this->humanDuration(1250);
     */
    protected function humanDuration(?int $durationMs): string
    {
        if ($durationMs === null || $durationMs <= 0) {
            return '-';
        }

        if ($durationMs < 1000) {
            return $durationMs.'ms';
        }

        return round($durationMs / 1000, 2).'s';
    }
}
