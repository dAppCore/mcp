<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Support\Facades\Cache;

/**
 * Per-(caller, tool) rate limiter for MCP tool invocations. Backed by
 * Laravel's cache; respects mcp.rate_limiting.enabled / decay_seconds /
 * limit-by-tool configuration. Returns a structured result so callers
 * can produce 429 responses with retry-after headers.
 *
 * Example:
 *
 *   $rl = new ToolRateLimiter();
 *   $result = $rl->check($apiKey, 'agent.brain.recall');
 *   if ($result['limited']) {
 *       return response()->json(['error' => 'rate_limited'], 429);
 *   }
 *   $rl->hit($apiKey, 'agent.brain.recall');
 */
final class ToolRateLimiter
{
    protected const CACHE_PREFIX = 'mcp_rate_limit:';

    /**
     * Inspect the rate-limit state for a (caller, tool) pair without
     * incrementing it. Returns ['limited' => bool, 'remaining' => int,
     * 'retry_after' => ?int].
     *
     * Example:
     *
     *   ['limited' => false, 'remaining' => 99, 'retry_after' => null]
     */
    public function check(string $identifier, string $toolName): array
    {
        if (! config('mcp.rate_limiting.enabled', true)) {
            return ['limited' => false, 'remaining' => PHP_INT_MAX, 'retry_after' => null];
        }

        $limit = $this->limitForTool($toolName);
        $cacheKey = $this->cacheKey($identifier, $toolName);
        $current = (int) Cache::get($cacheKey, 0);
        $decaySeconds = (int) config('mcp.rate_limiting.decay_seconds', 60);

        if ($current >= $limit) {
            $ttl = $this->ttl($cacheKey, $decaySeconds);

            return [
                'limited' => true,
                'remaining' => 0,
                'retry_after' => $ttl > 0 ? $ttl : $decaySeconds,
            ];
        }

        return [
            'limited' => false,
            'remaining' => max($limit - $current - 1, 0),
            'retry_after' => null,
        ];
    }

    /**
     * Increment the rate-limit counter for a (caller, tool) pair.
     *
     * @example
     * $rl->hit($apiKey, 'agent.brain.recall');
     */
    public function hit(string $identifier, string $toolName): void
    {
        if (! config('mcp.rate_limiting.enabled', true)) {
            return;
        }

        $cacheKey = $this->cacheKey($identifier, $toolName);
        $decaySeconds = (int) config('mcp.rate_limiting.decay_seconds', 60);

        if (Cache::add($cacheKey, 1, $decaySeconds)) {
            return;
        }

        Cache::increment($cacheKey);
    }

    /**
     * Clear one rate-limit bucket or every configured bucket for a caller.
     *
     * @example
     * $rl->clear($apiKey, 'agent.brain.recall');
     */
    public function clear(string $identifier, ?string $toolName = null): void
    {
        if ($toolName !== null) {
            Cache::forget($this->cacheKey($identifier, $toolName));

            return;
        }

        foreach (array_keys((array) config('mcp.rate_limiting.per_tool', [])) as $configuredTool) {
            Cache::forget($this->cacheKey($identifier, (string) $configuredTool));
        }

        Cache::forget($this->cacheKey($identifier, '*'));
    }

    /**
     * Return the current limit, remaining calls, and reset timestamp.
     *
     * @example
     * $status = $rl->getStatus($apiKey, 'agent.brain.recall');
     */
    public function getStatus(string $identifier, string $toolName): array
    {
        $limit = $this->limitForTool($toolName);
        $cacheKey = $this->cacheKey($identifier, $toolName);
        $current = (int) Cache::get($cacheKey, 0);
        $ttl = $this->ttl($cacheKey, (int) config('mcp.rate_limiting.decay_seconds', 60));

        return [
            'limit' => $limit,
            'remaining' => max($limit - $current, 0),
            'reset_at' => $ttl > 0 ? now()->addSeconds($ttl)->toIso8601String() : null,
        ];
    }

    /**
     * Resolve the configured call limit for one tool name.
     *
     * @example
     * $limit = $this->limitForTool('agent.brain.recall');
     */
    protected function limitForTool(string $toolName): int
    {
        $perTool = (array) config('mcp.rate_limiting.per_tool', []);

        if (array_key_exists($toolName, $perTool)) {
            return (int) $perTool[$toolName];
        }

        return (int) config('mcp.rate_limiting.calls_per_minute', 60);
    }

    /**
     * Build the cache key used for one caller and tool bucket.
     *
     * @example
     * $key = $this->cacheKey($apiKey, 'agent.brain.recall');
     */
    protected function cacheKey(string $identifier, string $toolName): string
    {
        return self::CACHE_PREFIX.$identifier.':'.$toolName;
    }

    /**
     * Read the remaining time-to-live for a cached rate-limit bucket.
     *
     * @example
     * $ttl = $this->ttl($this->cacheKey($apiKey, 'agent.brain.recall'), 60);
     */
    protected function ttl(string $cacheKey, int $default): int
    {
        try {
            $ttl = Cache::ttl($cacheKey);

            return is_int($ttl) ? $ttl : $default;
        } catch (\Throwable) {
            return $default;
        }
    }
}
