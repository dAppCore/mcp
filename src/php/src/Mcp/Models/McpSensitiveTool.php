<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;

/**
 * MCP Sensitive Tool - defines tools requiring stricter auditing.
 *
 * Used by AuditLogService to determine which tools need:
 * - Enhanced logging with is_sensitive flag
 * - Field redaction for privacy
 * - Explicit consent requirements
 *
 * @property int $id
 * @property string $tool_name
 * @property string $reason
 * @property array|null $redact_fields
 * @property bool $require_explicit_consent
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class McpSensitiveTool extends Model
{
    protected $table = 'mcp_sensitive_tools';

    protected $fillable = [
        'tool_name',
        'reason',
        'redact_fields',
        'require_explicit_consent',
    ];

    protected $casts = [
        'redact_fields' => 'array',
        'require_explicit_consent' => 'boolean',
    ];

    // -------------------------------------------------------------------------
    // Scopes
    // -------------------------------------------------------------------------

    /**
     * Find by tool name.
     */
    public function scopeForTool(Builder $query, string $toolName): Builder
    {
        return $query->where('tool_name', $toolName);
    }

    /**
     * Filter tools requiring explicit consent.
     */
    public function scopeRequiringConsent(Builder $query): Builder
    {
        return $query->where('require_explicit_consent', true);
    }

    // -------------------------------------------------------------------------
    // Static Methods
    // -------------------------------------------------------------------------

    /**
     * Check if a tool is marked as sensitive.
     */
    public static function isSensitive(string $toolName): bool
    {
        return static::where('tool_name', $toolName)->exists();
    }

    /**
     * Get sensitivity info for a tool.
     */
    public static function getSensitivityInfo(string $toolName): ?array
    {
        $tool = static::where('tool_name', $toolName)->first();

        if (! $tool) {
            return null;
        }

        return [
            'is_sensitive' => true,
            'reason' => $tool->reason,
            'redact_fields' => $tool->redact_fields ?? [],
            'require_explicit_consent' => $tool->require_explicit_consent,
        ];
    }

    /**
     * Register a sensitive tool.
     */
    public static function register(
        string $toolName,
        string $reason,
        array $redactFields = [],
        bool $requireConsent = false
    ): self {
        return static::updateOrCreate(
            ['tool_name' => $toolName],
            [
                'reason' => $reason,
                'redact_fields' => $redactFields,
                'require_explicit_consent' => $requireConsent,
            ]
        );
    }

    /**
     * Unregister a sensitive tool.
     */
    public static function unregister(string $toolName): bool
    {
        return static::where('tool_name', $toolName)->delete() > 0;
    }

    /**
     * Get all sensitive tool names.
     */
    public static function getAllToolNames(): array
    {
        return static::pluck('tool_name')->toArray();
    }
}
