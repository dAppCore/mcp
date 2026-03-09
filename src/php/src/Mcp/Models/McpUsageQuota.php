<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * MCP Usage Quota - tracks monthly workspace MCP usage.
 *
 * Stores monthly aggregated usage for tool calls and token consumption
 * to enforce workspace-level quotas.
 *
 * @property int $id
 * @property int $workspace_id
 * @property string $month YYYY-MM format
 * @property int $tool_calls_count
 * @property int $input_tokens
 * @property int $output_tokens
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class McpUsageQuota extends Model
{
    use BelongsToWorkspace;

    protected $table = 'mcp_usage_quotas';

    protected $fillable = [
        'workspace_id',
        'month',
        'tool_calls_count',
        'input_tokens',
        'output_tokens',
    ];

    protected $casts = [
        'tool_calls_count' => 'integer',
        'input_tokens' => 'integer',
        'output_tokens' => 'integer',
    ];

    // ─────────────────────────────────────────────────────────────────────────
    // Relationships
    // ─────────────────────────────────────────────────────────────────────────

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Scopes
    // ─────────────────────────────────────────────────────────────────────────

    public function scopeForMonth(Builder $query, string $month): Builder
    {
        return $query->where('month', $month);
    }

    public function scopeCurrentMonth(Builder $query): Builder
    {
        return $query->where('month', now()->format('Y-m'));
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Factory Methods
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Get or create usage quota record for a workspace and month.
     */
    public static function getOrCreate(int $workspaceId, ?string $month = null): self
    {
        $month = $month ?? now()->format('Y-m');

        return static::firstOrCreate(
            [
                'workspace_id' => $workspaceId,
                'month' => $month,
            ],
            [
                'tool_calls_count' => 0,
                'input_tokens' => 0,
                'output_tokens' => 0,
            ]
        );
    }

    /**
     * Get current month's quota for a workspace.
     */
    public static function getCurrentForWorkspace(int $workspaceId): self
    {
        return static::getOrCreate($workspaceId);
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Usage Recording
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Record usage (increments counters atomically).
     */
    public function recordUsage(int $toolCalls = 1, int $inputTokens = 0, int $outputTokens = 0): self
    {
        $this->increment('tool_calls_count', $toolCalls);

        if ($inputTokens > 0) {
            $this->increment('input_tokens', $inputTokens);
        }

        if ($outputTokens > 0) {
            $this->increment('output_tokens', $outputTokens);
        }

        return $this->fresh();
    }

    /**
     * Record usage for a workspace (static convenience method).
     */
    public static function record(
        int $workspaceId,
        int $toolCalls = 1,
        int $inputTokens = 0,
        int $outputTokens = 0
    ): self {
        $quota = static::getCurrentForWorkspace($workspaceId);

        return $quota->recordUsage($toolCalls, $inputTokens, $outputTokens);
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Computed Attributes
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Get total tokens (input + output).
     */
    public function getTotalTokensAttribute(): int
    {
        return $this->input_tokens + $this->output_tokens;
    }

    /**
     * Get formatted month (e.g., "January 2026").
     */
    public function getMonthLabelAttribute(): string
    {
        return \Carbon\Carbon::createFromFormat('Y-m', $this->month)->format('F Y');
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Helpers
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Reset usage counters (for billing cycle reset).
     */
    public function reset(): self
    {
        $this->update([
            'tool_calls_count' => 0,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        return $this;
    }

    /**
     * Convert to array for API responses.
     */
    public function toArray(): array
    {
        return [
            'workspace_id' => $this->workspace_id,
            'month' => $this->month,
            'month_label' => $this->month_label,
            'tool_calls_count' => $this->tool_calls_count,
            'input_tokens' => $this->input_tokens,
            'output_tokens' => $this->output_tokens,
            'total_tokens' => $this->total_tokens,
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }
}
