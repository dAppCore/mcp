<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Models\McpToolVersion;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;
use Symfony\Component\Yaml\Yaml;

/**
 * Registry for MCP Tools with schema and example management.
 *
 * Provides tool discovery, schema extraction, example inputs,
 * and category-based organisation for the MCP Playground UI.
 */
class ToolRegistry
{
    /**
     * Cache TTL for registry data (5 minutes).
     */
    protected const CACHE_TTL = 300;

    /**
     * Example inputs for specific tools.
     * These provide sensible defaults for testing tools.
     *
     * @var array<string, array<string, mixed>>
     */
    protected array $examples = [
        'query_database' => [
            'query' => 'SELECT id, name FROM users LIMIT 10',
        ],
        'list_tables' => [],
        'list_routes' => [],
        'list_sites' => [],
        'get_stats' => [],
        'create_coupon' => [
            'code' => 'SUMMER25',
            'discount_type' => 'percentage',
            'discount_value' => 25,
            'expires_at' => '2025-12-31',
        ],
        'list_invoices' => [
            'status' => 'paid',
            'limit' => 10,
        ],
        'get_billing_status' => [],
        'upgrade_plan' => [
            'plan_slug' => 'professional',
        ],
    ];

    /**
     * Get all available MCP servers.
     *
     * @return Collection<int, array{id: string, name: string, tagline: string, tool_count: int}>
     */
    public function getServers(): Collection
    {
        return Cache::remember('mcp:playground:servers', self::CACHE_TTL, function () {
            $registry = $this->loadRegistry();

            return collect($registry['servers'] ?? [])
                ->map(fn ($ref) => $this->loadServerSummary($ref['id']))
                ->filter()
                ->values();
        });
    }

    /**
     * Get all tools for a specific server.
     *
     * @return Collection<int, array{name: string, description: string, category: string, inputSchema: array, examples: array, version: string|null}>
     */
    public function getToolsForServer(string $serverId, bool $includeVersionInfo = false): Collection
    {
        $cacheKey = $includeVersionInfo
            ? "mcp:playground:tools:{$serverId}:versioned"
            : "mcp:playground:tools:{$serverId}";

        return Cache::remember($cacheKey, self::CACHE_TTL, function () use ($serverId, $includeVersionInfo) {
            $server = $this->loadServerFull($serverId);

            if (! $server) {
                return collect();
            }

            return collect($server['tools'] ?? [])
                ->map(function ($tool) use ($serverId, $includeVersionInfo) {
                    $name = $tool['name'];
                    $baseVersion = $tool['version'] ?? ToolVersionService::DEFAULT_VERSION;

                    $result = [
                        'name' => $name,
                        'description' => $tool['description'] ?? $tool['purpose'] ?? '',
                        'category' => $this->extractCategory($tool),
                        'inputSchema' => $tool['inputSchema'] ?? ['type' => 'object', 'properties' => $tool['parameters'] ?? []],
                        'examples' => $this->examples[$name] ?? $this->generateExampleFromSchema($tool['inputSchema'] ?? []),
                        'version' => $baseVersion,
                    ];

                    // Optionally enrich with database version info
                    if ($includeVersionInfo) {
                        $latestVersion = McpToolVersion::forServer($serverId)
                            ->forTool($name)
                            ->latest()
                            ->first();

                        if ($latestVersion) {
                            $result['version'] = $latestVersion->version;
                            $result['version_status'] = $latestVersion->status;
                            $result['is_deprecated'] = $latestVersion->is_deprecated;
                            $result['sunset_at'] = $latestVersion->sunset_at?->toIso8601String();

                            // Use versioned schema if available
                            if ($latestVersion->input_schema) {
                                $result['inputSchema'] = $latestVersion->input_schema;
                            }
                        }
                    }

                    return $result;
                })
                ->values();
        });
    }

    /**
     * Get all tools grouped by category.
     *
     * @return Collection<string, Collection<int, array>>
     */
    public function getToolsByCategory(string $serverId): Collection
    {
        return $this->getToolsForServer($serverId)
            ->groupBy('category')
            ->sortKeys();
    }

    /**
     * Search tools by name or description.
     *
     * @return Collection<int, array>
     */
    public function searchTools(string $serverId, string $query): Collection
    {
        $query = strtolower(trim($query));

        if (empty($query)) {
            return $this->getToolsForServer($serverId);
        }

        return $this->getToolsForServer($serverId)
            ->filter(function ($tool) use ($query) {
                return str_contains(strtolower($tool['name']), $query)
                    || str_contains(strtolower($tool['description']), $query)
                    || str_contains(strtolower($tool['category']), $query);
            })
            ->values();
    }

    /**
     * Get a specific tool by name.
     */
    public function getTool(string $serverId, string $toolName): ?array
    {
        return $this->getToolsForServer($serverId)
            ->firstWhere('name', $toolName);
    }

    /**
     * Get example inputs for a tool.
     */
    public function getExampleInputs(string $toolName): array
    {
        return $this->examples[$toolName] ?? [];
    }

    /**
     * Set custom example inputs for a tool.
     */
    public function setExampleInputs(string $toolName, array $examples): void
    {
        $this->examples[$toolName] = $examples;
    }

    /**
     * Get all categories across all servers.
     *
     * @return Collection<string, int>
     */
    public function getAllCategories(): Collection
    {
        return $this->getServers()
            ->flatMap(fn ($server) => $this->getToolsForServer($server['id']))
            ->groupBy('category')
            ->map(fn ($tools) => $tools->count())
            ->sortKeys();
    }

    /**
     * Get full server configuration.
     */
    public function getServerFull(string $serverId): ?array
    {
        return $this->loadServerFull($serverId);
    }

    /**
     * Clear cached registry data.
     */
    public function clearCache(): void
    {
        Cache::forget('mcp:playground:servers');

        foreach ($this->getServers() as $server) {
            Cache::forget("mcp:playground:tools:{$server['id']}");
        }
    }

    /**
     * Extract category from tool definition.
     */
    protected function extractCategory(array $tool): string
    {
        // Check for explicit category
        if (isset($tool['category'])) {
            return ucfirst($tool['category']);
        }

        // Infer from tool name
        $name = $tool['name'] ?? '';

        $categoryPatterns = [
            'query' => ['query', 'search', 'find', 'get', 'list'],
            'commerce' => ['coupon', 'invoice', 'billing', 'plan', 'payment', 'subscription'],
            'content' => ['content', 'article', 'page', 'post', 'media'],
            'system' => ['table', 'route', 'stat', 'config', 'setting'],
            'user' => ['user', 'auth', 'session', 'permission'],
        ];

        foreach ($categoryPatterns as $category => $patterns) {
            foreach ($patterns as $pattern) {
                if (str_contains(strtolower($name), $pattern)) {
                    return ucfirst($category);
                }
            }
        }

        return 'General';
    }

    /**
     * Generate example inputs from JSON schema.
     */
    protected function generateExampleFromSchema(array $schema): array
    {
        $properties = $schema['properties'] ?? [];
        $examples = [];

        foreach ($properties as $name => $prop) {
            $type = is_array($prop['type'] ?? 'string') ? ($prop['type'][0] ?? 'string') : ($prop['type'] ?? 'string');

            // Use default if available
            if (isset($prop['default'])) {
                $examples[$name] = $prop['default'];

                continue;
            }

            // Use example if available
            if (isset($prop['example'])) {
                $examples[$name] = $prop['example'];

                continue;
            }

            // Use first enum value if available
            if (isset($prop['enum']) && ! empty($prop['enum'])) {
                $examples[$name] = $prop['enum'][0];

                continue;
            }

            // Generate based on type
            $examples[$name] = match ($type) {
                'integer', 'number' => $prop['minimum'] ?? 0,
                'boolean' => false,
                'array' => [],
                'object' => new \stdClass,
                default => '', // string
            };
        }

        return $examples;
    }

    /**
     * Load the MCP registry file.
     */
    protected function loadRegistry(): array
    {
        $path = resource_path('mcp/registry.yaml');

        if (! file_exists($path)) {
            return ['servers' => []];
        }

        return Yaml::parseFile($path);
    }

    /**
     * Load full server configuration.
     */
    protected function loadServerFull(string $id): ?array
    {
        // Sanitise server ID to prevent path traversal
        $id = basename($id, '.yaml');

        if (! preg_match('/^[a-z0-9-]+$/', $id)) {
            return null;
        }

        $path = resource_path("mcp/servers/{$id}.yaml");

        if (! file_exists($path)) {
            return null;
        }

        return Yaml::parseFile($path);
    }

    /**
     * Load server summary (id, name, tagline, tool count).
     */
    protected function loadServerSummary(string $id): ?array
    {
        $server = $this->loadServerFull($id);

        if (! $server) {
            return null;
        }

        return [
            'id' => $server['id'],
            'name' => $server['name'],
            'tagline' => $server['tagline'] ?? '',
            'tool_count' => count($server['tools'] ?? []),
        ];
    }
}
