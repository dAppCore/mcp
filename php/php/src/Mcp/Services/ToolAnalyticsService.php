<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\DTO\ToolStats;
use Core\Mcp\Models\ToolMetric;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;

/**
 * Tool Analytics Service - analytics and reporting for MCP tool usage.
 *
 * Provides methods for recording tool executions and querying analytics data
 * including usage statistics, trends, and tool combinations.
 */
class ToolAnalyticsService
{
    /**
     * Batch of pending metrics to be flushed.
     *
     * @var array<string, array{calls: int, errors: int, duration: int, min: int|null, max: int|null}>
     */
    protected array $pendingMetrics = [];

    /**
     * Track tools used in current session for combination tracking.
     *
     * @var array<string, array<string>>
     */
    protected array $sessionTools = [];

    /**
     * Record a tool execution.
     */
    public function recordExecution(
        string $tool,
        int $durationMs,
        bool $success,
        ?string $workspaceId = null,
        ?string $sessionId = null
    ): void {
        if (! config('mcp.analytics.enabled', true)) {
            return;
        }

        $key = $this->getMetricKey($tool, $workspaceId);

        if (! isset($this->pendingMetrics[$key])) {
            $this->pendingMetrics[$key] = [
                'tool_name' => $tool,
                'workspace_id' => $workspaceId,
                'calls' => 0,
                'errors' => 0,
                'duration' => 0,
                'min' => null,
                'max' => null,
            ];
        }

        $this->pendingMetrics[$key]['calls']++;
        $this->pendingMetrics[$key]['duration'] += $durationMs;

        if (! $success) {
            $this->pendingMetrics[$key]['errors']++;
        }

        if ($this->pendingMetrics[$key]['min'] === null || $durationMs < $this->pendingMetrics[$key]['min']) {
            $this->pendingMetrics[$key]['min'] = $durationMs;
        }

        if ($this->pendingMetrics[$key]['max'] === null || $durationMs > $this->pendingMetrics[$key]['max']) {
            $this->pendingMetrics[$key]['max'] = $durationMs;
        }

        // Track tool combinations if session ID provided
        if ($sessionId !== null) {
            $this->trackToolInSession($sessionId, $tool, $workspaceId);
        }

        // Flush if batch size reached
        $batchSize = config('mcp.analytics.batch_size', 100);
        if ($this->getTotalPendingCalls() >= $batchSize) {
            $this->flush();
        }
    }

    /**
     * Get statistics for a specific tool.
     */
    public function getToolStats(string $tool, ?Carbon $from = null, ?Carbon $to = null): ToolStats
    {
        $from = $from ?? now()->subDays(30);
        $to = $to ?? now();

        $stats = ToolMetric::getAggregatedStats($tool, $from, $to);

        return ToolStats::fromArray($stats);
    }

    /**
     * Get statistics for all tools.
     */
    public function getAllToolStats(?Carbon $from = null, ?Carbon $to = null): Collection
    {
        $from = $from ?? now()->subDays(30);
        $to = $to ?? now();

        $results = ToolMetric::query()
            ->select('tool_name')
            ->selectRaw('SUM(call_count) as total_calls')
            ->selectRaw('SUM(error_count) as error_count')
            ->selectRaw('SUM(total_duration_ms) as total_duration')
            ->selectRaw('MIN(min_duration_ms) as min_duration_ms')
            ->selectRaw('MAX(max_duration_ms) as max_duration_ms')
            ->forDateRange($from, $to)
            ->groupBy('tool_name')
            ->orderByDesc('total_calls')
            ->get();

        return $results->map(function ($row) {
            $totalCalls = (int) $row->total_calls;
            $errorCount = (int) $row->error_count;
            $totalDuration = (int) $row->total_duration;

            return new ToolStats(
                toolName: $row->tool_name,
                totalCalls: $totalCalls,
                errorCount: $errorCount,
                errorRate: $totalCalls > 0 ? round(($errorCount / $totalCalls) * 100, 2) : 0.0,
                avgDurationMs: $totalCalls > 0 ? round($totalDuration / $totalCalls, 2) : 0.0,
                minDurationMs: (int) ($row->min_duration_ms ?? 0),
                maxDurationMs: (int) ($row->max_duration_ms ?? 0),
            );
        });
    }

    /**
     * Get the most popular tools by call count.
     */
    public function getPopularTools(int $limit = 10, ?Carbon $from = null, ?Carbon $to = null): Collection
    {
        return $this->getAllToolStats($from, $to)
            ->sortByDesc(fn (ToolStats $stats) => $stats->totalCalls)
            ->take($limit)
            ->values();
    }

    /**
     * Get tools with the highest error rates.
     */
    public function getErrorProneTools(int $limit = 10, ?Carbon $from = null, ?Carbon $to = null): Collection
    {
        $minCalls = 10; // Require minimum calls to be considered

        return $this->getAllToolStats($from, $to)
            ->filter(fn (ToolStats $stats) => $stats->totalCalls >= $minCalls)
            ->sortByDesc(fn (ToolStats $stats) => $stats->errorRate)
            ->take($limit)
            ->values();
    }

    /**
     * Get tool combinations - tools frequently used together.
     */
    public function getToolCombinations(int $limit = 10, ?Carbon $from = null, ?Carbon $to = null): Collection
    {
        $from = $from ?? now()->subDays(30);
        $to = $to ?? now();

        return DB::table('mcp_tool_combinations')
            ->select('tool_a', 'tool_b')
            ->selectRaw('SUM(occurrence_count) as total_occurrences')
            ->whereBetween('date', [$from->toDateString(), $to->toDateString()])
            ->groupBy('tool_a', 'tool_b')
            ->orderByDesc('total_occurrences')
            ->limit($limit)
            ->get()
            ->map(fn ($row) => [
                'tool_a' => $row->tool_a,
                'tool_b' => $row->tool_b,
                'occurrences' => (int) $row->total_occurrences,
            ]);
    }

    /**
     * Get usage trends for a specific tool.
     */
    public function getUsageTrends(string $tool, int $days = 30): array
    {
        $startDate = now()->subDays($days - 1)->startOfDay();
        $endDate = now()->endOfDay();

        $metrics = ToolMetric::forTool($tool)
            ->forDateRange($startDate, $endDate)
            ->orderBy('date')
            ->get()
            ->keyBy(fn ($m) => $m->date->toDateString());

        $trends = [];

        for ($i = $days - 1; $i >= 0; $i--) {
            $date = now()->subDays($i)->toDateString();
            $metric = $metrics->get($date);

            $trends[] = [
                'date' => $date,
                'date_formatted' => Carbon::parse($date)->format('M j'),
                'calls' => $metric?->call_count ?? 0,
                'errors' => $metric?->error_count ?? 0,
                'avg_duration_ms' => $metric?->average_duration ?? 0,
                'error_rate' => $metric?->error_rate ?? 0,
            ];
        }

        return $trends;
    }

    /**
     * Get workspace-specific statistics.
     */
    public function getWorkspaceStats(string $workspaceId, ?Carbon $from = null, ?Carbon $to = null): array
    {
        $from = $from ?? now()->subDays(30);
        $to = $to ?? now();

        $results = ToolMetric::query()
            ->forWorkspace($workspaceId)
            ->forDateRange($from, $to)
            ->get();

        $totalCalls = $results->sum('call_count');
        $errorCount = $results->sum('error_count');
        $totalDuration = $results->sum('total_duration_ms');
        $uniqueTools = $results->pluck('tool_name')->unique()->count();

        return [
            'workspace_id' => $workspaceId,
            'total_calls' => $totalCalls,
            'error_count' => $errorCount,
            'error_rate' => $totalCalls > 0 ? round(($errorCount / $totalCalls) * 100, 2) : 0.0,
            'avg_duration_ms' => $totalCalls > 0 ? round($totalDuration / $totalCalls, 2) : 0.0,
            'unique_tools' => $uniqueTools,
        ];
    }

    /**
     * Flush pending metrics to the database.
     */
    public function flush(): void
    {
        if (empty($this->pendingMetrics)) {
            return;
        }

        $date = now()->toDateString();

        foreach ($this->pendingMetrics as $data) {
            $metric = ToolMetric::firstOrCreate([
                'tool_name' => $data['tool_name'],
                'workspace_id' => $data['workspace_id'],
                'date' => $date,
            ], [
                'call_count' => 0,
                'error_count' => 0,
                'total_duration_ms' => 0,
            ]);

            $metric->call_count += $data['calls'];
            $metric->error_count += $data['errors'];
            $metric->total_duration_ms += $data['duration'];

            if ($data['min'] !== null) {
                if ($metric->min_duration_ms === null || $data['min'] < $metric->min_duration_ms) {
                    $metric->min_duration_ms = $data['min'];
                }
            }

            if ($data['max'] !== null) {
                if ($metric->max_duration_ms === null || $data['max'] > $metric->max_duration_ms) {
                    $metric->max_duration_ms = $data['max'];
                }
            }

            $metric->save();
        }

        // Flush session tool combinations
        $this->flushToolCombinations();

        $this->pendingMetrics = [];
    }

    /**
     * Track a tool being used in a session.
     */
    protected function trackToolInSession(string $sessionId, string $tool, ?string $workspaceId): void
    {
        $key = $sessionId.':'.($workspaceId ?? 'global');

        if (! isset($this->sessionTools[$key])) {
            $this->sessionTools[$key] = [
                'workspace_id' => $workspaceId,
                'tools' => [],
            ];
        }

        if (! in_array($tool, $this->sessionTools[$key]['tools'], true)) {
            $this->sessionTools[$key]['tools'][] = $tool;
        }
    }

    /**
     * Flush tool combinations to the database.
     */
    protected function flushToolCombinations(): void
    {
        $date = now()->toDateString();

        foreach ($this->sessionTools as $sessionData) {
            $tools = $sessionData['tools'];
            $workspaceId = $sessionData['workspace_id'];

            // Generate all unique pairs
            $count = count($tools);
            for ($i = 0; $i < $count; $i++) {
                for ($j = $i + 1; $j < $count; $j++) {
                    // Ensure consistent ordering (alphabetical)
                    $pair = [$tools[$i], $tools[$j]];
                    sort($pair);

                    DB::table('mcp_tool_combinations')
                        ->updateOrInsert(
                            [
                                'tool_a' => $pair[0],
                                'tool_b' => $pair[1],
                                'workspace_id' => $workspaceId,
                                'date' => $date,
                            ],
                            [
                                'occurrence_count' => DB::raw('occurrence_count + 1'),
                                'updated_at' => now(),
                            ]
                        );

                    // Handle insert case where occurrence_count wasn't set
                    DB::table('mcp_tool_combinations')
                        ->where('tool_a', $pair[0])
                        ->where('tool_b', $pair[1])
                        ->where('workspace_id', $workspaceId)
                        ->where('date', $date)
                        ->whereNull('created_at')
                        ->update([
                            'created_at' => now(),
                            'occurrence_count' => 1,
                        ]);
                }
            }
        }

        $this->sessionTools = [];
    }

    /**
     * Get the metric key for batching.
     */
    protected function getMetricKey(string $tool, ?string $workspaceId): string
    {
        return $tool.':'.($workspaceId ?? 'global');
    }

    /**
     * Get total pending calls across all batches.
     */
    protected function getTotalPendingCalls(): int
    {
        $total = 0;
        foreach ($this->pendingMetrics as $data) {
            $total += $data['calls'];
        }

        return $total;
    }
}
