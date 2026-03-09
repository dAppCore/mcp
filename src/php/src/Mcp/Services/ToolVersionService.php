<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Models\McpToolVersion;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;

/**
 * Tool Version Service - manages MCP tool versioning for backwards compatibility.
 *
 * Provides version registration, lookup, comparison, and migration support
 * for maintaining compatibility with running agents during tool schema changes.
 */
class ToolVersionService
{
    /**
     * Cache key prefix for version lookups.
     */
    protected const CACHE_PREFIX = 'mcp:tool_version:';

    /**
     * Cache TTL for version data (5 minutes).
     */
    protected const CACHE_TTL = 300;

    /**
     * Default version for unversioned tools.
     */
    public const DEFAULT_VERSION = '1.0.0';

    /**
     * Register a new tool version.
     *
     * @param  array  $options  Additional options (changelog, migration_notes, mark_latest)
     */
    public function registerVersion(
        string $serverId,
        string $toolName,
        string $version,
        ?array $inputSchema = null,
        ?array $outputSchema = null,
        ?string $description = null,
        array $options = []
    ): McpToolVersion {
        // Validate semver format
        if (! $this->isValidSemver($version)) {
            throw new \InvalidArgumentException("Invalid semver version: {$version}");
        }

        // Check if version already exists
        $existing = McpToolVersion::forServer($serverId)
            ->forTool($toolName)
            ->forVersion($version)
            ->first();

        if ($existing) {
            // Update existing version
            $existing->update([
                'input_schema' => $inputSchema ?? $existing->input_schema,
                'output_schema' => $outputSchema ?? $existing->output_schema,
                'description' => $description ?? $existing->description,
                'changelog' => $options['changelog'] ?? $existing->changelog,
                'migration_notes' => $options['migration_notes'] ?? $existing->migration_notes,
            ]);

            if ($options['mark_latest'] ?? false) {
                $existing->markAsLatest();
            }

            $this->clearCache($serverId, $toolName);

            return $existing->fresh();
        }

        // Create new version
        $toolVersion = McpToolVersion::create([
            'server_id' => $serverId,
            'tool_name' => $toolName,
            'version' => $version,
            'input_schema' => $inputSchema,
            'output_schema' => $outputSchema,
            'description' => $description,
            'changelog' => $options['changelog'] ?? null,
            'migration_notes' => $options['migration_notes'] ?? null,
            'is_latest' => false,
        ]);

        // Mark as latest if requested or if it's the first version
        $isFirst = McpToolVersion::forServer($serverId)->forTool($toolName)->count() === 1;

        if (($options['mark_latest'] ?? false) || $isFirst) {
            $toolVersion->markAsLatest();
        }

        $this->clearCache($serverId, $toolName);

        Log::info('MCP tool version registered', [
            'server_id' => $serverId,
            'tool_name' => $toolName,
            'version' => $version,
            'is_latest' => $toolVersion->is_latest,
        ]);

        return $toolVersion;
    }

    /**
     * Get a tool at a specific version.
     *
     * Returns null if version doesn't exist. Use getLatestVersion() for fallback.
     */
    public function getToolAtVersion(string $serverId, string $toolName, string $version): ?McpToolVersion
    {
        $cacheKey = self::CACHE_PREFIX."{$serverId}:{$toolName}:{$version}";

        return Cache::remember($cacheKey, self::CACHE_TTL, function () use ($serverId, $toolName, $version) {
            return McpToolVersion::forServer($serverId)
                ->forTool($toolName)
                ->forVersion($version)
                ->first();
        });
    }

    /**
     * Get the latest version of a tool.
     */
    public function getLatestVersion(string $serverId, string $toolName): ?McpToolVersion
    {
        $cacheKey = self::CACHE_PREFIX."{$serverId}:{$toolName}:latest";

        return Cache::remember($cacheKey, self::CACHE_TTL, function () use ($serverId, $toolName) {
            // First try to find explicitly marked latest
            $latest = McpToolVersion::forServer($serverId)
                ->forTool($toolName)
                ->latest()
                ->first();

            if ($latest) {
                return $latest;
            }

            // Fallback to newest version by semver
            return McpToolVersion::forServer($serverId)
                ->forTool($toolName)
                ->active()
                ->orderByVersion('desc')
                ->first();
        });
    }

    /**
     * Resolve a tool version, falling back to latest if not specified.
     *
     * @return array{version: McpToolVersion|null, warning: array|null, error: array|null}
     */
    public function resolveVersion(string $serverId, string $toolName, ?string $requestedVersion = null): array
    {
        // If no version requested, use latest
        if ($requestedVersion === null) {
            $version = $this->getLatestVersion($serverId, $toolName);

            return [
                'version' => $version,
                'warning' => null,
                'error' => $version === null ? [
                    'code' => 'TOOL_NOT_FOUND',
                    'message' => "No versions found for tool {$serverId}:{$toolName}",
                ] : null,
            ];
        }

        // Look up specific version
        $version = $this->getToolAtVersion($serverId, $toolName, $requestedVersion);

        if (! $version) {
            return [
                'version' => null,
                'warning' => null,
                'error' => [
                    'code' => 'VERSION_NOT_FOUND',
                    'message' => "Version {$requestedVersion} not found for tool {$serverId}:{$toolName}",
                ],
            ];
        }

        // Check if sunset
        if ($version->is_sunset) {
            return [
                'version' => null,
                'warning' => null,
                'error' => $version->getSunsetError(),
            ];
        }

        // Check if deprecated (warning, not error)
        $warning = $version->getDeprecationWarning();

        return [
            'version' => $version,
            'warning' => $warning,
            'error' => null,
        ];
    }

    /**
     * Check if a version is deprecated.
     */
    public function isDeprecated(string $serverId, string $toolName, string $version): bool
    {
        $toolVersion = $this->getToolAtVersion($serverId, $toolName, $version);

        return $toolVersion?->is_deprecated ?? false;
    }

    /**
     * Check if a version is sunset (blocked).
     */
    public function isSunset(string $serverId, string $toolName, string $version): bool
    {
        $toolVersion = $this->getToolAtVersion($serverId, $toolName, $version);

        return $toolVersion?->is_sunset ?? false;
    }

    /**
     * Compare two semver versions.
     *
     * @return int -1 if $a < $b, 0 if equal, 1 if $a > $b
     */
    public function compareVersions(string $a, string $b): int
    {
        return version_compare(
            $this->normalizeSemver($a),
            $this->normalizeSemver($b)
        );
    }

    /**
     * Get version history for a tool.
     *
     * @return Collection<int, McpToolVersion>
     */
    public function getVersionHistory(string $serverId, string $toolName): Collection
    {
        return McpToolVersion::forServer($serverId)
            ->forTool($toolName)
            ->orderByVersion('desc')
            ->get();
    }

    /**
     * Attempt to migrate a tool call from an old version schema to a new one.
     *
     * This is a best-effort migration that:
     * - Preserves arguments that exist in both schemas
     * - Applies defaults for new required arguments where possible
     * - Returns warnings for arguments that couldn't be migrated
     *
     * @return array{arguments: array, warnings: array, success: bool}
     */
    public function migrateToolCall(
        string $serverId,
        string $toolName,
        string $fromVersion,
        string $toVersion,
        array $arguments
    ): array {
        $fromTool = $this->getToolAtVersion($serverId, $toolName, $fromVersion);
        $toTool = $this->getToolAtVersion($serverId, $toolName, $toVersion);

        if (! $fromTool || ! $toTool) {
            return [
                'arguments' => $arguments,
                'warnings' => ['Could not load version schemas for migration'],
                'success' => false,
            ];
        }

        $toSchema = $toTool->input_schema ?? [];
        $toProperties = $toSchema['properties'] ?? [];
        $toRequired = $toSchema['required'] ?? [];

        $migratedArgs = [];
        $warnings = [];

        // Copy over arguments that exist in the new schema
        foreach ($arguments as $key => $value) {
            if (isset($toProperties[$key])) {
                $migratedArgs[$key] = $value;
            } else {
                $warnings[] = "Argument '{$key}' removed in version {$toVersion}";
            }
        }

        // Check for new required arguments without defaults
        foreach ($toRequired as $requiredKey) {
            if (! isset($migratedArgs[$requiredKey])) {
                // Try to apply default from schema
                if (isset($toProperties[$requiredKey]['default'])) {
                    $migratedArgs[$requiredKey] = $toProperties[$requiredKey]['default'];
                    $warnings[] = "Applied default value for new required argument '{$requiredKey}'";
                } else {
                    $warnings[] = "Missing required argument '{$requiredKey}' added in version {$toVersion}";
                }
            }
        }

        return [
            'arguments' => $migratedArgs,
            'warnings' => $warnings,
            'success' => empty(array_filter($warnings, fn ($w) => str_starts_with($w, 'Missing required'))),
        ];
    }

    /**
     * Deprecate a tool version with optional sunset date.
     */
    public function deprecateVersion(
        string $serverId,
        string $toolName,
        string $version,
        ?Carbon $sunsetAt = null
    ): ?McpToolVersion {
        $toolVersion = McpToolVersion::forServer($serverId)
            ->forTool($toolName)
            ->forVersion($version)
            ->first();

        if (! $toolVersion) {
            return null;
        }

        $toolVersion->deprecate($sunsetAt);
        $this->clearCache($serverId, $toolName);

        Log::info('MCP tool version deprecated', [
            'server_id' => $serverId,
            'tool_name' => $toolName,
            'version' => $version,
            'sunset_at' => $sunsetAt?->toIso8601String(),
        ]);

        return $toolVersion;
    }

    /**
     * Get all tools with version info for a server.
     *
     * @return Collection<string, array{latest: McpToolVersion|null, versions: Collection}>
     */
    public function getToolsWithVersions(string $serverId): Collection
    {
        $versions = McpToolVersion::forServer($serverId)
            ->orderByVersion('desc')
            ->get();

        return $versions->groupBy('tool_name')
            ->map(function ($toolVersions, $toolName) {
                return [
                    'tool_name' => $toolName,
                    'latest' => $toolVersions->firstWhere('is_latest', true) ?? $toolVersions->first(),
                    'versions' => $toolVersions,
                    'version_count' => $toolVersions->count(),
                    'has_deprecated' => $toolVersions->contains(fn ($v) => $v->is_deprecated),
                    'has_sunset' => $toolVersions->contains(fn ($v) => $v->is_sunset),
                ];
            });
    }

    /**
     * Get all unique servers that have versioned tools.
     */
    public function getServersWithVersions(): Collection
    {
        return McpToolVersion::select('server_id')
            ->distinct()
            ->orderBy('server_id')
            ->pluck('server_id');
    }

    /**
     * Sync tool versions from YAML server definitions.
     *
     * Call this during deployment to register/update versions from server configs.
     *
     * @param  array  $serverConfig  Parsed YAML server configuration
     * @param  string  $version  Version to register (e.g., from deployment tag)
     */
    public function syncFromServerConfig(array $serverConfig, string $version, bool $markLatest = true): int
    {
        $serverId = $serverConfig['id'] ?? null;
        $tools = $serverConfig['tools'] ?? [];

        if (! $serverId || empty($tools)) {
            return 0;
        }

        $registered = 0;

        foreach ($tools as $tool) {
            $toolName = $tool['name'] ?? null;
            if (! $toolName) {
                continue;
            }

            $this->registerVersion(
                serverId: $serverId,
                toolName: $toolName,
                version: $version,
                inputSchema: $tool['inputSchema'] ?? null,
                outputSchema: $tool['outputSchema'] ?? null,
                description: $tool['description'] ?? $tool['purpose'] ?? null,
                options: [
                    'mark_latest' => $markLatest,
                ]
            );

            $registered++;
        }

        return $registered;
    }

    /**
     * Get statistics about tool versions.
     */
    public function getStats(): array
    {
        return [
            'total_versions' => McpToolVersion::count(),
            'total_tools' => McpToolVersion::select('server_id', 'tool_name')
                ->distinct()
                ->count(),
            'deprecated_count' => McpToolVersion::deprecated()->count(),
            'sunset_count' => McpToolVersion::sunset()->count(),
            'servers' => $this->getServersWithVersions()->count(),
        ];
    }

    // -------------------------------------------------------------------------
    // Protected Methods
    // -------------------------------------------------------------------------

    /**
     * Validate semver format.
     */
    protected function isValidSemver(string $version): bool
    {
        // Basic semver pattern: major.minor.patch with optional prerelease/build
        $pattern = '/^(\d+)\.(\d+)\.(\d+)(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$/';

        return (bool) preg_match($pattern, $version);
    }

    /**
     * Normalize semver for comparison (removes prerelease/build metadata).
     */
    protected function normalizeSemver(string $version): string
    {
        // Remove prerelease and build metadata for basic comparison
        return preg_replace('/[-+].*$/', '', $version) ?? $version;
    }

    /**
     * Clear cache for a tool's versions.
     */
    protected function clearCache(string $serverId, string $toolName): void
    {
        // Clear specific version caches would require tracking all versions
        // For simplicity, we use a short TTL and let cache naturally expire
        Cache::forget(self::CACHE_PREFIX."{$serverId}:{$toolName}:latest");
    }
}
