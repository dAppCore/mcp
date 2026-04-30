<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;
use Symfony\Component\Process\Process;
use Symfony\Component\Yaml\Yaml;

/**
 * MCP Server Health Check Service
 *
 * Pings MCP servers via stdio to check their health status.
 * Results are cached for 60 seconds to avoid excessive subprocess spawning.
 */
class McpHealthService
{
    public const STATUS_ONLINE = 'online';

    public const STATUS_OFFLINE = 'offline';

    public const STATUS_DEGRADED = 'degraded';

    public const STATUS_UNKNOWN = 'unknown';

    /**
     * Cache TTL in seconds for health check results.
     */
    protected int $cacheTtl = 60;

    /**
     * Timeout in seconds for health check ping.
     */
    protected int $timeout = 5;

    /**
     * Check health of a specific MCP server.
     */
    public function check(string $serverId, bool $forceRefresh = false): array
    {
        $cacheKey = "mcp:health:{$serverId}";

        if (! $forceRefresh && Cache::has($cacheKey)) {
            return Cache::get($cacheKey);
        }

        $server = $this->loadServerConfig($serverId);

        if (! $server) {
            $result = $this->buildResult(self::STATUS_UNKNOWN, 'Server not found');
            Cache::put($cacheKey, $result, $this->cacheTtl);

            return $result;
        }

        $result = $this->pingServer($server);
        Cache::put($cacheKey, $result, $this->cacheTtl);

        return $result;
    }

    /**
     * Check health of all registered MCP servers.
     */
    public function checkAll(bool $forceRefresh = false): array
    {
        $servers = $this->getRegisteredServers();
        $results = [];

        foreach ($servers as $serverId) {
            $results[$serverId] = $this->check($serverId, $forceRefresh);
        }

        return $results;
    }

    /**
     * Get cached health status without triggering a check.
     */
    public function getCachedStatus(string $serverId): ?array
    {
        return Cache::get("mcp:health:{$serverId}");
    }

    /**
     * Clear cached health status for a server.
     */
    public function clearCache(string $serverId): void
    {
        Cache::forget("mcp:health:{$serverId}");
    }

    /**
     * Clear all cached health statuses.
     */
    public function clearAllCache(): void
    {
        foreach ($this->getRegisteredServers() as $serverId) {
            Cache::forget("mcp:health:{$serverId}");
        }
    }

    /**
     * Ping a server by sending a minimal MCP request.
     */
    protected function pingServer(array $server): array
    {
        $connection = $server['connection'] ?? [];
        $type = $connection['type'] ?? 'stdio';

        // Only support stdio for now
        if ($type !== 'stdio') {
            return $this->buildResult(
                self::STATUS_UNKNOWN,
                "Connection type '{$type}' health check not supported"
            );
        }

        $command = $connection['command'] ?? null;
        $args = $connection['args'] ?? [];
        $cwd = $this->resolveEnvVars($connection['cwd'] ?? getcwd());

        if (! $command) {
            return $this->buildResult(self::STATUS_OFFLINE, 'No command configured');
        }

        // Build the MCP initialize request
        $initRequest = json_encode([
            'jsonrpc' => '2.0',
            'method' => 'initialize',
            'params' => [
                'protocolVersion' => '2024-11-05',
                'capabilities' => [],
                'clientInfo' => [
                    'name' => 'mcp-health-check',
                    'version' => '1.0.0',
                ],
            ],
            'id' => 1,
        ]);

        try {
            $startTime = microtime(true);

            // Build full command
            $fullCommand = array_merge([$command], $args);
            $process = new Process($fullCommand, $cwd);
            $process->setInput($initRequest);
            $process->setTimeout($this->timeout);

            $process->run();

            $duration = round((microtime(true) - $startTime) * 1000);
            $output = $process->getOutput();

            // Check for valid JSON-RPC response
            if ($process->isSuccessful() && ! empty($output)) {
                // Try to parse the response
                $lines = explode("\n", trim($output));
                foreach ($lines as $line) {
                    $response = json_decode($line, true);
                    if ($response && isset($response['result'])) {
                        return $this->buildResult(
                            self::STATUS_ONLINE,
                            'Server responding',
                            [
                                'response_time_ms' => $duration,
                                'server_info' => $response['result']['serverInfo'] ?? null,
                                'protocol_version' => $response['result']['protocolVersion'] ?? null,
                            ]
                        );
                    }
                }
            }

            // Process ran but didn't return expected response
            if ($process->isSuccessful()) {
                return $this->buildResult(
                    self::STATUS_DEGRADED,
                    'Server started but returned unexpected response',
                    [
                        'response_time_ms' => $duration,
                        'output' => substr($output, 0, 500),
                    ]
                );
            }

            // Process failed
            return $this->buildResult(
                self::STATUS_OFFLINE,
                'Server failed to start',
                [
                    'exit_code' => $process->getExitCode(),
                    'error' => substr($process->getErrorOutput(), 0, 500),
                ]
            );

        } catch (\Exception $e) {
            Log::warning("MCP health check failed for {$server['id']}", [
                'error' => $e->getMessage(),
            ]);

            return $this->buildResult(
                self::STATUS_OFFLINE,
                'Health check failed: '.$e->getMessage()
            );
        }
    }

    /**
     * Build a health check result array.
     */
    protected function buildResult(string $status, string $message, array $extra = []): array
    {
        return array_merge([
            'status' => $status,
            'message' => $message,
            'checked_at' => now()->toIso8601String(),
        ], $extra);
    }

    /**
     * Get list of registered server IDs.
     */
    protected function getRegisteredServers(): array
    {
        $registry = $this->loadRegistry();

        return collect($registry['servers'] ?? [])
            ->pluck('id')
            ->all();
    }

    /**
     * Load the main registry file.
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
     * Load a server's YAML config.
     */
    protected function loadServerConfig(string $id): ?array
    {
        $path = resource_path("mcp/servers/{$id}.yaml");

        if (! file_exists($path)) {
            return null;
        }

        return Yaml::parseFile($path);
    }

    /**
     * Resolve environment variables in a string.
     */
    protected function resolveEnvVars(string $value): string
    {
        return preg_replace_callback('/\$\{([^}]+)\}/', function ($matches) {
            $parts = explode(':-', $matches[1], 2);
            $var = $parts[0];
            $default = $parts[1] ?? '';

            return env($var, $default);
        }, $value);
    }

    /**
     * Get status badge HTML.
     */
    public function getStatusBadge(string $status): string
    {
        return match ($status) {
            self::STATUS_ONLINE => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">Online</span>',
            self::STATUS_OFFLINE => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">Offline</span>',
            self::STATUS_DEGRADED => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">Degraded</span>',
            default => '<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">Unknown</span>',
        };
    }

    /**
     * Get status colour class for Tailwind.
     */
    public function getStatusColour(string $status): string
    {
        return match ($status) {
            self::STATUS_ONLINE => 'green',
            self::STATUS_OFFLINE => 'red',
            self::STATUS_DEGRADED => 'yellow',
            default => 'gray',
        };
    }
}
