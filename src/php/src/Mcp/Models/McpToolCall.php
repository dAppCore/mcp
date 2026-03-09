<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * MCP Tool Call - logs individual MCP tool invocations.
 *
 * Tracks tool usage for analytics and debugging.
 * Updates daily aggregates automatically.
 *
 * @property int $id
 * @property int|null $workspace_id
 * @property string $server_id
 * @property string $tool_name
 * @property string|null $session_id
 * @property array|null $input_params
 * @property bool $success
 * @property int|null $duration_ms
 * @property string|null $error_message
 * @property string|null $error_code
 * @property array|null $result_summary
 * @property string|null $agent_type
 * @property string|null $plan_slug
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class McpToolCall extends Model
{
    use BelongsToWorkspace;

    protected $fillable = [
        'workspace_id',
        'server_id',
        'tool_name',
        'session_id',
        'input_params',
        'success',
        'duration_ms',
        'error_message',
        'error_code',
        'result_summary',
        'agent_type',
        'plan_slug',
    ];

    protected $casts = [
        'input_params' => 'array',
        'result_summary' => 'array',
        'success' => 'boolean',
        'duration_ms' => 'integer',
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

    public function scopeSuccessful(Builder $query): Builder
    {
        return $query->where('success', true);
    }

    public function scopeFailed(Builder $query): Builder
    {
        return $query->where('success', false);
    }

    public function scopeRecent(Builder $query, int $hours = 24): Builder
    {
        return $query->where('created_at', '>=', now()->subHours($hours));
    }

    public function scopeToday(Builder $query): Builder
    {
        return $query->whereDate('created_at', today());
    }

    public function scopeThisWeek(Builder $query): Builder
    {
        return $query->where('created_at', '>=', now()->startOfWeek());
    }

    /**
     * Log a tool call and update daily stats.
     */
    public static function log(
        string $serverId,
        string $toolName,
        array $params = [],
        bool $success = true,
        ?int $durationMs = null,
        ?string $errorMessage = null,
        ?string $errorCode = null,
        ?array $resultSummary = null,
        ?string $sessionId = null,
        ?string $agentType = null,
        ?string $planSlug = null,
        ?int $workspaceId = null
    ): self {
        $call = static::create([
            'workspace_id' => $workspaceId,
            'server_id' => $serverId,
            'tool_name' => $toolName,
            'input_params' => $params,
            'success' => $success,
            'duration_ms' => $durationMs,
            'error_message' => $errorMessage,
            'error_code' => $errorCode,
            'result_summary' => $resultSummary,
            'session_id' => $sessionId,
            'agent_type' => $agentType,
            'plan_slug' => $planSlug,
        ]);

        // Update daily stats
        McpToolCallStat::incrementForCall($call);

        return $call;
    }

    // Helpers
    public function getDurationForHumans(): string
    {
        if (! $this->duration_ms) {
            return '-';
        }

        if ($this->duration_ms < 1000) {
            return $this->duration_ms.'ms';
        }

        return round($this->duration_ms / 1000, 2).'s';
    }

    public function getStatusBadge(): string
    {
        return $this->success
            ? '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">Success</span>'
            : '<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">Failed</span>';
    }
}
