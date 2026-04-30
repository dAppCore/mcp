<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Support\Facades\Cache;

/**
 * Rate limiter for MCP tool calls.
 *
 * Provides rate limiting for both HTTP and STDIO server tool invocations.
 * Uses cache-based rate limiting that works with any cache driver.
 */
class ToolRateLimiter
{
    /**
     * Cache key prefix for rate limit tracking.
     */
    protected const CACHE_PREFIX = 'mcp_rate_limit:';

    /**
     * Check if a tool call should be rate limited.
     *
     * @param  string  $identifier  Session ID, API key, or other unique identifier
     * @param  string  $toolName  The tool being called
     * @return array{limited: bool, remaining: int, retry_after: int|null}
     */
    public function check(string $identifier, string $toolName): array
    {
        if (! config('mcp.rate_limiting.enabled', true)) {
            return ['limited' => false, 'remaining' => PHP_INT_MAX, 'retry_after' => null];
        }

        $limit = $this->getLimitForTool($toolName);
        $decaySeconds = config('mcp.rate_limiting.decay_seconds', 60);
        $cacheKey = $this->getCacheKey($identifier, $toolName);

        $current = (int) Cache::get($cacheKey, 0);

        if ($current >= $limit) {
            $ttl = Cache::ttl($cacheKey);

            return [
                'limited' => true,
                'remaining' => 0,
                'retry_after' => $ttl > 0 ? $ttl : $decaySeconds,
            ];
        }

        return [
            'limited' => false,
            'remaining' => $limit - $current - 1,
            'retry_after' => null,
        ];
    }

    /**
     * Record a tool call against the rate limit.
     *
     * @param  string  $identifier  Session ID, API key, or other unique identifier
     * @param  string  $toolName  The tool being called
     */
    public function hit(string $identifier, string $toolName): void
    {
        if (! config('mcp.rate_limiting.enabled', true)) {
            return;
        }

        $decaySeconds = config('mcp.rate_limiting.decay_seconds', 60);
        $cacheKey = $this->getCacheKey($identifier, $toolName);

        $current = (int) Cache::get($cacheKey, 0);

        if ($current === 0) {
            // First call - set with expiration
            Cache::put($cacheKey, 1, $decaySeconds);
        } else {
            // Increment without resetting TTL
            Cache::increment($cacheKey);
        }
    }

    /**
     * Clear rate limit for an identifier.
     *
     * @param  string  $identifier  Session ID, API key, or other unique identifier
     * @param  string|null  $toolName  Specific tool, or null to clear all
     */
    public function clear(string $identifier, ?string $toolName = null): void
    {
        if ($toolName !== null) {
            Cache::forget($this->getCacheKey($identifier, $toolName));
        } else {
            // Clear all tool rate limits for this identifier (requires knowing tools)
            // For now, just clear the specific key pattern
            Cache::forget($this->getCacheKey($identifier, '*'));
        }
    }

    /**
     * Get the rate limit for a specific tool.
     */
    protected function getLimitForTool(string $toolName): int
    {
        // Check for tool-specific limit
        $perToolLimits = config('mcp.rate_limiting.per_tool', []);

        if (isset($perToolLimits[$toolName])) {
            return (int) $perToolLimits[$toolName];
        }

        // Use default limit
        return (int) config('mcp.rate_limiting.calls_per_minute', 60);
    }

    /**
     * Generate cache key for rate limiting.
     */
    protected function getCacheKey(string $identifier, string $toolName): string
    {
        // Use general key for overall rate limiting
        return self::CACHE_PREFIX.$identifier.':'.$toolName;
    }

    /**
     * Get rate limit status for reporting.
     *
     * @return array{limit: int, remaining: int, reset_at: string|null}
     */
    public function getStatus(string $identifier, string $toolName): array
    {
        $limit = $this->getLimitForTool($toolName);
        $cacheKey = $this->getCacheKey($identifier, $toolName);
        $current = (int) Cache::get($cacheKey, 0);
        $ttl = Cache::ttl($cacheKey);

        return [
            'limit' => $limit,
            'remaining' => max(0, $limit - $current),
            'reset_at' => $ttl > 0 ? now()->addSeconds($ttl)->toIso8601String() : null,
        ];
    }
}
