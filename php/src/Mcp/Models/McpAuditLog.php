<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

/**
 * MCP Audit Log - immutable audit trail for MCP tool executions.
 *
 * Implements a hash chain for tamper detection. Each entry contains
 * a hash of the previous entry, creating a verifiable chain of custody.
 *
 * @property int $id
 * @property string $server_id
 * @property string $tool_name
 * @property int|null $workspace_id
 * @property string|null $session_id
 * @property array|null $input_params
 * @property array|null $output_summary
 * @property bool $success
 * @property int|null $duration_ms
 * @property string|null $error_code
 * @property string|null $error_message
 * @property string|null $actor_type
 * @property int|null $actor_id
 * @property string|null $actor_ip
 * @property bool $is_sensitive
 * @property string|null $sensitivity_reason
 * @property string|null $previous_hash
 * @property string $entry_hash
 * @property string|null $agent_type
 * @property string|null $plan_slug
 * @property \Carbon\Carbon $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class McpAuditLog extends Model
{
    use BelongsToWorkspace;

    /**
     * Actor types.
     */
    public const ACTOR_USER = 'user';

    public const ACTOR_API_KEY = 'api_key';

    public const ACTOR_SYSTEM = 'system';

    /**
     * The table associated with the model.
     */
    protected $table = 'mcp_audit_logs';

    /**
     * Indicates if the model should be timestamped.
     * We handle timestamps manually for immutability.
     */
    public $timestamps = true;

    /**
     * The attributes that are mass assignable.
     */
    protected $fillable = [
        'server_id',
        'tool_name',
        'workspace_id',
        'session_id',
        'input_params',
        'output_summary',
        'success',
        'duration_ms',
        'error_code',
        'error_message',
        'actor_type',
        'actor_id',
        'actor_ip',
        'is_sensitive',
        'sensitivity_reason',
        'previous_hash',
        'entry_hash',
        'agent_type',
        'plan_slug',
    ];

    /**
     * The attributes that should be cast.
     */
    protected $casts = [
        'input_params' => 'array',
        'output_summary' => 'array',
        'success' => 'boolean',
        'duration_ms' => 'integer',
        'actor_id' => 'integer',
        'is_sensitive' => 'boolean',
        'created_at' => 'datetime',
    ];

    /**
     * Boot the model.
     */
    protected static function boot(): void
    {
        parent::boot();

        // Prevent updates to maintain immutability
        static::updating(function (self $model) {
            // Allow only specific fields to be updated (for soft operations)
            $allowedChanges = ['updated_at'];
            $changes = array_keys($model->getDirty());

            foreach ($changes as $change) {
                if (! in_array($change, $allowedChanges)) {
                    throw new \RuntimeException(
                        'Audit log entries are immutable. Cannot modify: '.$change
                    );
                }
            }
        });

        // Prevent deletion
        static::deleting(function () {
            throw new \RuntimeException(
                'Audit log entries cannot be deleted. They are immutable for compliance purposes.'
            );
        });
    }

    // -------------------------------------------------------------------------
    // Relationships
    // -------------------------------------------------------------------------

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    // -------------------------------------------------------------------------
    // Scopes
    // -------------------------------------------------------------------------

    /**
     * Filter by server.
     */
    public function scopeForServer(Builder $query, string $serverId): Builder
    {
        return $query->where('server_id', $serverId);
    }

    /**
     * Filter by tool name.
     */
    public function scopeForTool(Builder $query, string $toolName): Builder
    {
        return $query->where('tool_name', $toolName);
    }

    /**
     * Filter by session.
     */
    public function scopeForSession(Builder $query, string $sessionId): Builder
    {
        return $query->where('session_id', $sessionId);
    }

    /**
     * Filter successful calls.
     */
    public function scopeSuccessful(Builder $query): Builder
    {
        return $query->where('success', true);
    }

    /**
     * Filter failed calls.
     */
    public function scopeFailed(Builder $query): Builder
    {
        return $query->where('success', false);
    }

    /**
     * Filter sensitive tool calls.
     */
    public function scopeSensitive(Builder $query): Builder
    {
        return $query->where('is_sensitive', true);
    }

    /**
     * Filter by actor type.
     */
    public function scopeByActorType(Builder $query, string $actorType): Builder
    {
        return $query->where('actor_type', $actorType);
    }

    /**
     * Filter by actor.
     */
    public function scopeByActor(Builder $query, string $actorType, int $actorId): Builder
    {
        return $query->where('actor_type', $actorType)
            ->where('actor_id', $actorId);
    }

    /**
     * Filter by date range.
     */
    public function scopeInDateRange(Builder $query, string|\DateTimeInterface $start, string|\DateTimeInterface $end): Builder
    {
        return $query->whereBetween('created_at', [$start, $end]);
    }

    /**
     * Filter for today.
     */
    public function scopeToday(Builder $query): Builder
    {
        return $query->whereDate('created_at', today());
    }

    /**
     * Filter for last N days.
     */
    public function scopeLastDays(Builder $query, int $days): Builder
    {
        return $query->where('created_at', '>=', now()->subDays($days));
    }

    // -------------------------------------------------------------------------
    // Hash Chain Methods
    // -------------------------------------------------------------------------

    /**
     * Compute the hash for this entry.
     * Uses SHA-256 to create a deterministic hash of the entry data.
     */
    public function computeHash(): string
    {
        $data = [
            'id' => $this->id,
            'server_id' => $this->server_id,
            'tool_name' => $this->tool_name,
            'workspace_id' => $this->workspace_id,
            'session_id' => $this->session_id,
            'input_params' => $this->input_params,
            'output_summary' => $this->output_summary,
            'success' => $this->success,
            'duration_ms' => $this->duration_ms,
            'error_code' => $this->error_code,
            'actor_type' => $this->actor_type,
            'actor_id' => $this->actor_id,
            'actor_ip' => $this->actor_ip,
            'is_sensitive' => $this->is_sensitive,
            'previous_hash' => $this->previous_hash,
            'created_at' => $this->created_at?->toIso8601String(),
        ];

        return hash('sha256', json_encode($data, JSON_THROW_ON_ERROR));
    }

    /**
     * Verify this entry's hash is valid.
     */
    public function verifyHash(): bool
    {
        return $this->entry_hash === $this->computeHash();
    }

    /**
     * Verify the chain link to the previous entry.
     */
    public function verifyChainLink(): bool
    {
        if ($this->previous_hash === null) {
            // First entry in chain - check there's no earlier entry
            return ! static::where('id', '<', $this->id)->exists();
        }

        $previous = static::where('id', '<', $this->id)
            ->orderByDesc('id')
            ->first();

        if (! $previous) {
            return false; // Previous entry missing
        }

        return $this->previous_hash === $previous->entry_hash;
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    /**
     * Get duration formatted for humans.
     */
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

    /**
     * Get actor display name.
     */
    public function getActorDisplay(): string
    {
        return match ($this->actor_type) {
            self::ACTOR_USER => "User #{$this->actor_id}",
            self::ACTOR_API_KEY => "API Key #{$this->actor_id}",
            self::ACTOR_SYSTEM => 'System',
            default => 'Unknown',
        };
    }

    /**
     * Check if this entry has integrity issues.
     */
    public function hasIntegrityIssues(): bool
    {
        return ! $this->verifyHash() || ! $this->verifyChainLink();
    }

    /**
     * Get integrity status.
     */
    public function getIntegrityStatus(): array
    {
        $hashValid = $this->verifyHash();
        $chainValid = $this->verifyChainLink();

        return [
            'valid' => $hashValid && $chainValid,
            'hash_valid' => $hashValid,
            'chain_valid' => $chainValid,
            'issues' => array_filter([
                ! $hashValid ? 'Entry hash mismatch - data may have been tampered' : null,
                ! $chainValid ? 'Chain link broken - previous entry missing or modified' : null,
            ]),
        ];
    }

    /**
     * Convert to array for export.
     */
    public function toExportArray(): array
    {
        return [
            'id' => $this->id,
            'timestamp' => $this->created_at->toIso8601String(),
            'server_id' => $this->server_id,
            'tool_name' => $this->tool_name,
            'workspace_id' => $this->workspace_id,
            'session_id' => $this->session_id,
            'success' => $this->success,
            'duration_ms' => $this->duration_ms,
            'error_code' => $this->error_code,
            'actor_type' => $this->actor_type,
            'actor_id' => $this->actor_id,
            'actor_ip' => $this->actor_ip,
            'is_sensitive' => $this->is_sensitive,
            'sensitivity_reason' => $this->sensitivity_reason,
            'entry_hash' => $this->entry_hash,
            'previous_hash' => $this->previous_hash,
            'agent_type' => $this->agent_type,
            'plan_slug' => $this->plan_slug,
        ];
    }
}
