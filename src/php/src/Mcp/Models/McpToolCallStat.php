<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;

/**
 * MCP Tool Call Stats - daily aggregates for MCP tool calls.
 *
 * Provides efficient querying for dashboards and reports.
 * Updated automatically when McpToolCall::log() is called.
 *
 * @property int $id
 * @property int|null $workspace_id
 * @property \Carbon\Carbon $date
 * @property string $server_id
 * @property string $tool_name
 * @property int $call_count
 * @property int $success_count
 * @property int $error_count
 * @property int $total_duration_ms
 * @property int|null $min_duration_ms
 * @property int|null $max_duration_ms
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class McpToolCallStat extends Model
{
    use BelongsToWorkspace;

    protected $fillable = [
        'workspace_id',
        'date',
        'server_id',
        'tool_name',
        'call_count',
        'success_count',
        'error_count',
        'total_duration_ms',
        'min_duration_ms',
        'max_duration_ms',
    ];

    protected $casts = [
        'date' => 'date',
        'call_count' => 'integer',
        'success_count' => 'integer',
        'error_count' => 'integer',
        'total_duration_ms' => 'integer',
        'min_duration_ms' => 'integer',
        'max_duration_ms' => 'integer',
    ];

    // Relationships
    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    // Scopes
    public function scopeForServer(Builder $query, string $serverId): Builder
    {
        return $query->where('server_id', $serverId);
    }

    public function scopeForTool(Builder $query, string $toolName): Builder
    {
        return $query->where('tool_name', $toolName);
    }

    public function scopeForDate(Builder $query, Carbon|string $date): Builder
    {
        $date = $date instanceof Carbon ? $date->toDateString() : $date;

        return $query->where('date', $date);
    }

    public function scopeForDateRange(Builder $query, Carbon|string $start, Carbon|string $end): Builder
    {
        $start = $start instanceof Carbon ? $start->toDateString() : $start;
        $end = $end instanceof Carbon ? $end->toDateString() : $end;

        return $query->whereBetween('date', [$start, $end]);
    }

    public function scopeLast7Days(Builder $query): Builder
    {
        return $query->forDateRange(now()->subDays(6), now());
    }

    public function scopeLast30Days(Builder $query): Builder
    {
        return $query->forDateRange(now()->subDays(29), now());
    }

    /**
     * Increment stats for a tool call.
     */
    public static function incrementForCall(McpToolCall $call): void
    {
        $stat = static::firstOrCreate([
            'date' => $call->created_at->toDateString(),
            'server_id' => $call->server_id,
            'tool_name' => $call->tool_name,
            'workspace_id' => $call->workspace_id,
        ], [
            'call_count' => 0,
            'success_count' => 0,
            'error_count' => 0,
            'total_duration_ms' => 0,
        ]);

        $stat->call_count++;

        if ($call->success) {
            $stat->success_count++;
        } else {
            $stat->error_count++;
        }

        if ($call->duration_ms) {
            $stat->total_duration_ms += $call->duration_ms;

            if ($stat->min_duration_ms === null || $call->duration_ms < $stat->min_duration_ms) {
                $stat->min_duration_ms = $call->duration_ms;
            }

            if ($stat->max_duration_ms === null || $call->duration_ms > $stat->max_duration_ms) {
                $stat->max_duration_ms = $call->duration_ms;
            }
        }

        $stat->save();
    }

    // Computed attributes
    public function getSuccessRateAttribute(): float
    {
        if ($this->call_count === 0) {
            return 0;
        }

        return round(($this->success_count / $this->call_count) * 100, 1);
    }

    public function getAvgDurationMsAttribute(): ?float
    {
        if ($this->call_count === 0 || $this->total_duration_ms === 0) {
            return null;
        }

        return round($this->total_duration_ms / $this->call_count, 1);
    }

    public function getAvgDurationForHumansAttribute(): string
    {
        $avg = $this->avg_duration_ms;
        if ($avg === null) {
            return '-';
        }

        if ($avg < 1000) {
            return round($avg).'ms';
        }

        return round($avg / 1000, 2).'s';
    }

    /**
     * Get top tools by call count.
     */
    public static function getTopTools(int $days = 7, int $limit = 10, ?int $workspaceId = null): Collection
    {
        $query = static::query()
            ->select('server_id', 'tool_name')
            ->selectRaw('SUM(call_count) as total_calls')
            ->selectRaw('SUM(success_count) as total_success')
            ->selectRaw('SUM(error_count) as total_errors')
            ->selectRaw('AVG(total_duration_ms / NULLIF(call_count, 0)) as avg_duration')
            ->forDateRange(now()->subDays($days - 1), now())
            ->groupBy('server_id', 'tool_name')
            ->orderByDesc('total_calls')
            ->limit($limit);

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        return $query->get()
            ->map(function ($item) {
                $item->success_rate = $item->total_calls > 0
                    ? round(($item->total_success / $item->total_calls) * 100, 1)
                    : 0;

                return $item;
            });
    }

    /**
     * Get daily trend data.
     */
    public static function getDailyTrend(int $days = 7, ?int $workspaceId = null): Collection
    {
        $query = static::query()
            ->select('date')
            ->selectRaw('SUM(call_count) as total_calls')
            ->selectRaw('SUM(success_count) as total_success')
            ->selectRaw('SUM(error_count) as total_errors')
            ->forDateRange(now()->subDays($days - 1), now())
            ->groupBy('date')
            ->orderBy('date');

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        return $query->get()
            ->map(function ($item) {
                $item->success_rate = $item->total_calls > 0
                    ? round(($item->total_success / $item->total_calls) * 100, 1)
                    : 0;

                return $item;
            });
    }

    /**
     * Get server-level statistics.
     */
    public static function getServerStats(int $days = 7, ?int $workspaceId = null): Collection
    {
        $query = static::query()
            ->select('server_id')
            ->selectRaw('SUM(call_count) as total_calls')
            ->selectRaw('SUM(success_count) as total_success')
            ->selectRaw('SUM(error_count) as total_errors')
            ->selectRaw('COUNT(DISTINCT tool_name) as unique_tools')
            ->forDateRange(now()->subDays($days - 1), now())
            ->groupBy('server_id')
            ->orderByDesc('total_calls');

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        return $query->get()
            ->map(function ($item) {
                $item->success_rate = $item->total_calls > 0
                    ? round(($item->total_success / $item->total_calls) * 100, 1)
                    : 0;

                return $item;
            });
    }
}
