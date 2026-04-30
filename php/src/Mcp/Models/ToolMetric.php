<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * Tool Metric - daily aggregates for MCP tool usage analytics.
 *
 * Tracks per-tool call counts, error rates, and response times.
 * Updated automatically via ToolAnalyticsService.
 *
 * @property int $id
 * @property string $tool_name
 * @property string|null $workspace_id
 * @property int $call_count
 * @property int $error_count
 * @property int $total_duration_ms
 * @property int|null $min_duration_ms
 * @property int|null $max_duration_ms
 * @property \Carbon\Carbon $date
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 * @property-read float $average_duration
 * @property-read float $error_rate
 */
class ToolMetric extends Model
{
    protected $table = 'mcp_tool_metrics';

    protected $fillable = [
        'tool_name',
        'workspace_id',
        'call_count',
        'error_count',
        'total_duration_ms',
        'min_duration_ms',
        'max_duration_ms',
        'date',
    ];

    protected $casts = [
        'date' => 'date',
        'call_count' => 'integer',
        'error_count' => 'integer',
        'total_duration_ms' => 'integer',
        'min_duration_ms' => 'integer',
        'max_duration_ms' => 'integer',
    ];

    // -------------------------------------------------------------------------
    // Scopes
    // -------------------------------------------------------------------------

    /**
     * Filter metrics for a specific tool.
     */
    public function scopeForTool(Builder $query, string $toolName): Builder
    {
        return $query->where('tool_name', $toolName);
    }

    /**
     * Filter metrics for a specific workspace.
     */
    public function scopeForWorkspace(Builder $query, ?string $workspaceId): Builder
    {
        if ($workspaceId === null) {
            return $query->whereNull('workspace_id');
        }

        return $query->where('workspace_id', $workspaceId);
    }

    /**
     * Filter metrics within a date range.
     */
    public function scopeForDateRange(Builder $query, Carbon|string $start, Carbon|string $end): Builder
    {
        $start = $start instanceof Carbon ? $start->toDateString() : $start;
        $end = $end instanceof Carbon ? $end->toDateString() : $end;

        return $query->whereBetween('date', [$start, $end]);
    }

    /**
     * Filter metrics for today.
     */
    public function scopeToday(Builder $query): Builder
    {
        return $query->where('date', today()->toDateString());
    }

    /**
     * Filter metrics for the last N days.
     */
    public function scopeLastDays(Builder $query, int $days): Builder
    {
        return $query->forDateRange(now()->subDays($days - 1), now());
    }

    // -------------------------------------------------------------------------
    // Accessors
    // -------------------------------------------------------------------------

    /**
     * Get the average duration in milliseconds.
     */
    public function getAverageDurationAttribute(): float
    {
        if ($this->call_count === 0 || $this->total_duration_ms === 0) {
            return 0.0;
        }

        return round($this->total_duration_ms / $this->call_count, 2);
    }

    /**
     * Get the error rate as a percentage (0-100).
     */
    public function getErrorRateAttribute(): float
    {
        if ($this->call_count === 0) {
            return 0.0;
        }

        return round(($this->error_count / $this->call_count) * 100, 2);
    }

    /**
     * Get average duration formatted for display.
     */
    public function getAverageDurationForHumansAttribute(): string
    {
        $avg = $this->average_duration;

        if ($avg === 0.0) {
            return '-';
        }

        if ($avg < 1000) {
            return round($avg).'ms';
        }

        return round($avg / 1000, 2).'s';
    }

    // -------------------------------------------------------------------------
    // Methods
    // -------------------------------------------------------------------------

    /**
     * Record a successful tool call.
     */
    public static function recordCall(
        string $toolName,
        int $durationMs,
        ?string $workspaceId = null,
        ?Carbon $date = null
    ): self {
        $date = $date ?? now();

        $metric = static::firstOrCreate([
            'tool_name' => $toolName,
            'workspace_id' => $workspaceId,
            'date' => $date->toDateString(),
        ], [
            'call_count' => 0,
            'error_count' => 0,
            'total_duration_ms' => 0,
        ]);

        $metric->call_count++;
        $metric->total_duration_ms += $durationMs;

        if ($metric->min_duration_ms === null || $durationMs < $metric->min_duration_ms) {
            $metric->min_duration_ms = $durationMs;
        }

        if ($metric->max_duration_ms === null || $durationMs > $metric->max_duration_ms) {
            $metric->max_duration_ms = $durationMs;
        }

        $metric->save();

        return $metric;
    }

    /**
     * Record a failed tool call.
     */
    public static function recordError(
        string $toolName,
        int $durationMs,
        ?string $workspaceId = null,
        ?Carbon $date = null
    ): self {
        $date = $date ?? now();

        $metric = static::firstOrCreate([
            'tool_name' => $toolName,
            'workspace_id' => $workspaceId,
            'date' => $date->toDateString(),
        ], [
            'call_count' => 0,
            'error_count' => 0,
            'total_duration_ms' => 0,
        ]);

        $metric->call_count++;
        $metric->error_count++;
        $metric->total_duration_ms += $durationMs;

        if ($metric->min_duration_ms === null || $durationMs < $metric->min_duration_ms) {
            $metric->min_duration_ms = $durationMs;
        }

        if ($metric->max_duration_ms === null || $durationMs > $metric->max_duration_ms) {
            $metric->max_duration_ms = $durationMs;
        }

        $metric->save();

        return $metric;
    }

    /**
     * Get aggregated stats for a tool across all dates.
     */
    public static function getAggregatedStats(
        string $toolName,
        ?Carbon $from = null,
        ?Carbon $to = null,
        ?string $workspaceId = null
    ): array {
        $query = static::forTool($toolName);

        if ($from && $to) {
            $query->forDateRange($from, $to);
        }

        if ($workspaceId !== null) {
            $query->forWorkspace($workspaceId);
        }

        $metrics = $query->get();

        if ($metrics->isEmpty()) {
            return [
                'tool_name' => $toolName,
                'total_calls' => 0,
                'error_count' => 0,
                'error_rate' => 0.0,
                'avg_duration_ms' => 0.0,
                'min_duration_ms' => 0,
                'max_duration_ms' => 0,
            ];
        }

        $totalCalls = $metrics->sum('call_count');
        $errorCount = $metrics->sum('error_count');
        $totalDuration = $metrics->sum('total_duration_ms');

        return [
            'tool_name' => $toolName,
            'total_calls' => $totalCalls,
            'error_count' => $errorCount,
            'error_rate' => $totalCalls > 0 ? round(($errorCount / $totalCalls) * 100, 2) : 0.0,
            'avg_duration_ms' => $totalCalls > 0 ? round($totalDuration / $totalCalls, 2) : 0.0,
            'min_duration_ms' => $metrics->min('min_duration_ms') ?? 0,
            'max_duration_ms' => $metrics->max('max_duration_ms') ?? 0,
        ];
    }
}
