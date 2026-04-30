<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * MCP Tool Version - tracks versioned tool schemas for backwards compatibility.
 *
 * Enables running agents to continue using older tool versions while
 * newer versions are deployed. Supports deprecation lifecycle with
 * warnings and eventual sunset blocking.
 *
 * @property int $id
 * @property string $server_id
 * @property string $tool_name
 * @property string $version
 * @property array|null $input_schema
 * @property array|null $output_schema
 * @property string|null $description
 * @property string|null $changelog
 * @property string|null $migration_notes
 * @property bool $is_latest
 * @property \Carbon\Carbon|null $deprecated_at
 * @property \Carbon\Carbon|null $sunset_at
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 * @property-read bool $is_deprecated
 * @property-read bool $is_sunset
 * @property-read string $status
 * @property-read string $full_name
 */
class McpToolVersion extends Model
{
    protected $table = 'mcp_tool_versions';

    protected $fillable = [
        'server_id',
        'tool_name',
        'version',
        'input_schema',
        'output_schema',
        'description',
        'changelog',
        'migration_notes',
        'is_latest',
        'deprecated_at',
        'sunset_at',
    ];

    protected $casts = [
        'input_schema' => 'array',
        'output_schema' => 'array',
        'is_latest' => 'boolean',
        'deprecated_at' => 'datetime',
        'sunset_at' => 'datetime',
    ];

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
     * Filter by specific version.
     */
    public function scopeForVersion(Builder $query, string $version): Builder
    {
        return $query->where('version', $version);
    }

    /**
     * Get only latest versions.
     */
    public function scopeLatest(Builder $query): Builder
    {
        return $query->where('is_latest', true);
    }

    /**
     * Get deprecated versions.
     */
    public function scopeDeprecated(Builder $query): Builder
    {
        return $query->whereNotNull('deprecated_at')
            ->where('deprecated_at', '<=', now());
    }

    /**
     * Get sunset versions (blocked).
     */
    public function scopeSunset(Builder $query): Builder
    {
        return $query->whereNotNull('sunset_at')
            ->where('sunset_at', '<=', now());
    }

    /**
     * Get active versions (not sunset).
     */
    public function scopeActive(Builder $query): Builder
    {
        return $query->where(function ($q) {
            $q->whereNull('sunset_at')
                ->orWhere('sunset_at', '>', now());
        });
    }

    /**
     * Order by version (newest first using semver sort).
     */
    public function scopeOrderByVersion(Builder $query, string $direction = 'desc'): Builder
    {
        // Basic version ordering - splits on dots and orders numerically
        // For production use, consider a more robust semver sorting approach
        return $query->orderByRaw(
            "CAST(SUBSTRING_INDEX(version, '.', 1) AS UNSIGNED) {$direction}, ".
            "CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(version, '.', 2), '.', -1) AS UNSIGNED) {$direction}, ".
            "CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(version, '.', 3), '.', -1) AS UNSIGNED) {$direction}"
        );
    }

    // -------------------------------------------------------------------------
    // Accessors
    // -------------------------------------------------------------------------

    /**
     * Check if this version is deprecated.
     */
    public function getIsDeprecatedAttribute(): bool
    {
        return $this->deprecated_at !== null && $this->deprecated_at->isPast();
    }

    /**
     * Check if this version is sunset (blocked).
     */
    public function getIsSunsetAttribute(): bool
    {
        return $this->sunset_at !== null && $this->sunset_at->isPast();
    }

    /**
     * Get the lifecycle status of this version.
     */
    public function getStatusAttribute(): string
    {
        if ($this->is_sunset) {
            return 'sunset';
        }

        if ($this->is_deprecated) {
            return 'deprecated';
        }

        if ($this->is_latest) {
            return 'latest';
        }

        return 'active';
    }

    /**
     * Get full tool identifier (server:tool).
     */
    public function getFullNameAttribute(): string
    {
        return "{$this->server_id}:{$this->tool_name}";
    }

    /**
     * Get full versioned identifier (server:tool@version).
     */
    public function getVersionedNameAttribute(): string
    {
        return "{$this->server_id}:{$this->tool_name}@{$this->version}";
    }

    // -------------------------------------------------------------------------
    // Methods
    // -------------------------------------------------------------------------

    /**
     * Get deprecation warning message if deprecated but not sunset.
     */
    public function getDeprecationWarning(): ?array
    {
        if (! $this->is_deprecated || $this->is_sunset) {
            return null;
        }

        $warning = [
            'code' => 'TOOL_VERSION_DEPRECATED',
            'message' => "Tool version {$this->version} is deprecated.",
            'current_version' => $this->version,
        ];

        // Find the latest version to suggest
        $latest = static::forServer($this->server_id)
            ->forTool($this->tool_name)
            ->latest()
            ->first();

        if ($latest && $latest->version !== $this->version) {
            $warning['latest_version'] = $latest->version;
            $warning['message'] .= " Please upgrade to version {$latest->version}.";
        }

        if ($this->sunset_at) {
            $warning['sunset_at'] = $this->sunset_at->toIso8601String();
            $warning['message'] .= " This version will be blocked after {$this->sunset_at->format('Y-m-d')}.";
        }

        if ($this->migration_notes) {
            $warning['migration_notes'] = $this->migration_notes;
        }

        return $warning;
    }

    /**
     * Get sunset error if this version is blocked.
     */
    public function getSunsetError(): ?array
    {
        if (! $this->is_sunset) {
            return null;
        }

        $error = [
            'code' => 'TOOL_VERSION_SUNSET',
            'message' => "Tool version {$this->version} is no longer available as of {$this->sunset_at->format('Y-m-d')}.",
            'sunset_version' => $this->version,
            'sunset_at' => $this->sunset_at->toIso8601String(),
        ];

        // Find the latest version to suggest
        $latest = static::forServer($this->server_id)
            ->forTool($this->tool_name)
            ->latest()
            ->first();

        if ($latest && $latest->version !== $this->version) {
            $error['latest_version'] = $latest->version;
            $error['message'] .= " Please use version {$latest->version} instead.";
        }

        if ($this->migration_notes) {
            $error['migration_notes'] = $this->migration_notes;
        }

        return $error;
    }

    /**
     * Compare schemas between this version and another.
     *
     * @return array{added: array, removed: array, changed: array}
     */
    public function compareSchemaWith(self $other): array
    {
        $thisProps = $this->input_schema['properties'] ?? [];
        $otherProps = $other->input_schema['properties'] ?? [];

        $added = array_diff_key($otherProps, $thisProps);
        $removed = array_diff_key($thisProps, $otherProps);

        $changed = [];
        foreach (array_intersect_key($thisProps, $otherProps) as $key => $thisProp) {
            $otherProp = $otherProps[$key];
            if (json_encode($thisProp) !== json_encode($otherProp)) {
                $changed[$key] = [
                    'from' => $thisProp,
                    'to' => $otherProp,
                ];
            }
        }

        return [
            'added' => array_keys($added),
            'removed' => array_keys($removed),
            'changed' => $changed,
        ];
    }

    /**
     * Mark this version as deprecated.
     */
    public function deprecate(?Carbon $sunsetAt = null): self
    {
        $this->deprecated_at = now();

        if ($sunsetAt) {
            $this->sunset_at = $sunsetAt;
        }

        $this->save();

        return $this;
    }

    /**
     * Mark this version as the latest (and unmark others).
     */
    public function markAsLatest(): self
    {
        // Unmark all other versions for this tool
        static::forServer($this->server_id)
            ->forTool($this->tool_name)
            ->where('id', '!=', $this->id)
            ->update(['is_latest' => false]);

        $this->is_latest = true;
        $this->save();

        return $this;
    }

    /**
     * Export version info for API responses.
     */
    public function toApiArray(): array
    {
        return [
            'server_id' => $this->server_id,
            'tool_name' => $this->tool_name,
            'version' => $this->version,
            'is_latest' => $this->is_latest,
            'status' => $this->status,
            'description' => $this->description,
            'input_schema' => $this->input_schema,
            'output_schema' => $this->output_schema,
            'deprecated_at' => $this->deprecated_at?->toIso8601String(),
            'sunset_at' => $this->sunset_at?->toIso8601String(),
            'migration_notes' => $this->migration_notes,
            'changelog' => $this->changelog,
            'created_at' => $this->created_at?->toIso8601String(),
        ];
    }
}
