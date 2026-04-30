<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Closure;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;
use Mod\Mcp\Exceptions\CircuitOpenException;
use Throwable;

/**
 * Circuit Breaker for external module dependencies.
 *
 * Provides fault tolerance when dependent services (like the Agentic module)
 * are unavailable. Implements the circuit breaker pattern with three states:
 * - Closed: Normal operation, requests pass through
 * - Open: Service is down, requests fail fast
 * - Half-Open: Testing if service has recovered
 *
 * @see https://martinfowler.com/bliki/CircuitBreaker.html
 */
class CircuitBreaker
{
    /**
     * Cache key prefix for circuit state.
     */
    protected const CACHE_PREFIX = 'circuit_breaker:';

    /**
     * Circuit states.
     */
    public const STATE_CLOSED = 'closed';

    public const STATE_OPEN = 'open';

    public const STATE_HALF_OPEN = 'half_open';

    /**
     * Default TTL for success/failure counters (seconds).
     */
    protected const COUNTER_TTL = 300;

    /**
     * Execute a callable with circuit breaker protection.
     *
     * @param  string  $service  Service identifier (e.g., 'agentic', 'content')
     * @param  Closure  $operation  The operation to execute
     * @param  Closure|null  $fallback  Optional fallback when circuit is open
     * @return mixed The operation result or fallback value
     *
     * @throws CircuitOpenException When circuit is open and no fallback provided
     * @throws Throwable When operation fails and circuit records the failure
     */
    public function call(string $service, Closure $operation, ?Closure $fallback = null): mixed
    {
        $state = $this->getState($service);

        // Fast fail when circuit is open
        if ($state === self::STATE_OPEN) {
            Log::debug("Circuit breaker open for {$service}, failing fast");

            if ($fallback !== null) {
                return $fallback();
            }

            throw new CircuitOpenException($service);
        }

        // Handle half-open state with trial lock to prevent concurrent trial requests
        $hasTrialLock = false;
        if ($state === self::STATE_HALF_OPEN) {
            $hasTrialLock = $this->acquireTrialLock($service);

            if (! $hasTrialLock) {
                // Another request is already testing the service, fail fast
                Log::debug("Circuit breaker half-open for {$service}, trial in progress, failing fast");

                if ($fallback !== null) {
                    return $fallback();
                }

                throw new CircuitOpenException($service, "Service '{$service}' is being tested. Please try again shortly.");
            }
        }

        // Try the operation
        try {
            $result = $operation();

            // Record success and release trial lock if held
            $this->recordSuccess($service);

            if ($hasTrialLock) {
                $this->releaseTrialLock($service);
            }

            return $result;
        } catch (Throwable $e) {
            // Release trial lock if held
            if ($hasTrialLock) {
                $this->releaseTrialLock($service);
            }

            // Record failure
            $this->recordFailure($service, $e);

            // Check if we should trip the circuit
            if ($this->shouldTrip($service)) {
                $this->tripCircuit($service);
            }

            // If fallback provided and this is a recoverable error, use it
            if ($fallback !== null && $this->isRecoverableError($e)) {
                Log::warning("Circuit breaker using fallback for {$service}", [
                    'error' => $e->getMessage(),
                ]);

                return $fallback();
            }

            throw $e;
        }
    }

    /**
     * Get the current state of a circuit.
     */
    public function getState(string $service): string
    {
        $cacheKey = $this->getStateKey($service);

        $state = Cache::get($cacheKey);

        if ($state === null) {
            return self::STATE_CLOSED;
        }

        // Check if open circuit should transition to half-open
        if ($state === self::STATE_OPEN) {
            $openedAt = Cache::get($this->getOpenedAtKey($service));
            $resetTimeout = $this->getResetTimeout($service);

            if ($openedAt && (time() - $openedAt) >= $resetTimeout) {
                $this->setState($service, self::STATE_HALF_OPEN);

                return self::STATE_HALF_OPEN;
            }
        }

        return $state;
    }

    /**
     * Get circuit statistics for monitoring.
     */
    public function getStats(string $service): array
    {
        return [
            'service' => $service,
            'state' => $this->getState($service),
            'failures' => (int) Cache::get($this->getFailureCountKey($service), 0),
            'successes' => (int) Cache::get($this->getSuccessCountKey($service), 0),
            'last_failure' => Cache::get($this->getLastFailureKey($service)),
            'opened_at' => Cache::get($this->getOpenedAtKey($service)),
            'threshold' => $this->getFailureThreshold($service),
            'reset_timeout' => $this->getResetTimeout($service),
        ];
    }

    /**
     * Manually reset a circuit to closed state.
     */
    public function reset(string $service): void
    {
        $this->setState($service, self::STATE_CLOSED);
        Cache::forget($this->getFailureCountKey($service));
        Cache::forget($this->getSuccessCountKey($service));
        Cache::forget($this->getLastFailureKey($service));
        Cache::forget($this->getOpenedAtKey($service));

        Log::info("Circuit breaker manually reset for {$service}");
    }

    /**
     * Check if a service is available (circuit not open).
     */
    public function isAvailable(string $service): bool
    {
        return $this->getState($service) !== self::STATE_OPEN;
    }

    /**
     * Record a successful operation.
     */
    protected function recordSuccess(string $service): void
    {
        $state = $this->getState($service);

        // Increment success counter with TTL
        $this->atomicIncrement($this->getSuccessCountKey($service), self::COUNTER_TTL);

        // If half-open and we got a success, close the circuit
        if ($state === self::STATE_HALF_OPEN) {
            $this->closeCircuit($service);
        }

        // Decay failures over time (successful calls reduce failure count)
        $this->atomicDecrement($this->getFailureCountKey($service));
    }

    /**
     * Record a failed operation.
     */
    protected function recordFailure(string $service, Throwable $e): void
    {
        $failureKey = $this->getFailureCountKey($service);
        $lastFailureKey = $this->getLastFailureKey($service);
        $window = $this->getFailureWindow($service);

        // Atomic increment with TTL refresh using lock
        $newCount = $this->atomicIncrement($failureKey, $window);

        // Record last failure details
        Cache::put($lastFailureKey, [
            'message' => $e->getMessage(),
            'class' => get_class($e),
            'time' => now()->toIso8601String(),
        ], $window);

        Log::warning("Circuit breaker recorded failure for {$service}", [
            'error' => $e->getMessage(),
            'failures' => $newCount,
        ]);
    }

    /**
     * Check if the circuit should trip (open).
     */
    protected function shouldTrip(string $service): bool
    {
        $failures = (int) Cache::get($this->getFailureCountKey($service), 0);
        $threshold = $this->getFailureThreshold($service);

        return $failures >= $threshold;
    }

    /**
     * Trip the circuit to open state.
     */
    protected function tripCircuit(string $service): void
    {
        $this->setState($service, self::STATE_OPEN);
        Cache::put($this->getOpenedAtKey($service), time(), 86400); // 24h max

        Log::error("Circuit breaker tripped for {$service}", [
            'failures' => Cache::get($this->getFailureCountKey($service)),
        ]);
    }

    /**
     * Close the circuit after successful recovery.
     */
    protected function closeCircuit(string $service): void
    {
        $this->setState($service, self::STATE_CLOSED);
        Cache::forget($this->getFailureCountKey($service));
        Cache::forget($this->getOpenedAtKey($service));

        Log::info("Circuit breaker closed for {$service} after successful recovery");
    }

    /**
     * Set circuit state.
     */
    protected function setState(string $service, string $state): void
    {
        Cache::put($this->getStateKey($service), $state, 86400); // 24h max
    }

    /**
     * Check if an exception is recoverable (should use fallback).
     */
    protected function isRecoverableError(Throwable $e): bool
    {
        // Database connection errors, table not found, etc.
        $recoverablePatterns = [
            'SQLSTATE',
            'Connection refused',
            'Table .* doesn\'t exist',
            'Base table or view not found',
            'Connection timed out',
            'Too many connections',
        ];

        $message = $e->getMessage();

        foreach ($recoverablePatterns as $pattern) {
            if (preg_match('/'.$pattern.'/i', $message)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Get the failure threshold from config.
     */
    protected function getFailureThreshold(string $service): int
    {
        return (int) config("mcp.circuit_breaker.{$service}.threshold",
            config('mcp.circuit_breaker.default_threshold', 5)
        );
    }

    /**
     * Get the reset timeout (how long to wait before trying again).
     */
    protected function getResetTimeout(string $service): int
    {
        return (int) config("mcp.circuit_breaker.{$service}.reset_timeout",
            config('mcp.circuit_breaker.default_reset_timeout', 60)
        );
    }

    /**
     * Get the failure window (how long failures are counted).
     */
    protected function getFailureWindow(string $service): int
    {
        return (int) config("mcp.circuit_breaker.{$service}.failure_window",
            config('mcp.circuit_breaker.default_failure_window', 120)
        );
    }

    /**
     * Atomically increment a counter with TTL refresh.
     *
     * Uses a lock to ensure the increment and TTL refresh are atomic.
     */
    protected function atomicIncrement(string $key, int $ttl): int
    {
        $lock = Cache::lock($key.':lock', 5);

        try {
            $lock->block(3);

            $current = (int) Cache::get($key, 0);
            $newValue = $current + 1;
            Cache::put($key, $newValue, $ttl);

            return $newValue;
        } finally {
            $lock->release();
        }
    }

    /**
     * Atomically decrement a counter (only if positive).
     *
     * Note: We use COUNTER_TTL as a fallback since Laravel's Cache facade
     * doesn't expose remaining TTL. The counter will refresh on activity.
     */
    protected function atomicDecrement(string $key): int
    {
        $lock = Cache::lock($key.':lock', 5);

        try {
            $lock->block(3);

            $current = (int) Cache::get($key, 0);
            if ($current > 0) {
                $newValue = $current - 1;
                Cache::put($key, $newValue, self::COUNTER_TTL);

                return $newValue;
            }

            return 0;
        } finally {
            $lock->release();
        }
    }

    /**
     * Acquire a trial lock for half-open state.
     *
     * Only one request can hold the trial lock at a time, preventing
     * concurrent trial requests during half-open state.
     */
    protected function acquireTrialLock(string $service): bool
    {
        $lockKey = $this->getTrialLockKey($service);

        // Try to acquire lock with a short TTL (auto-release if request hangs)
        return Cache::add($lockKey, true, 30);
    }

    /**
     * Release the trial lock.
     */
    protected function releaseTrialLock(string $service): void
    {
        Cache::forget($this->getTrialLockKey($service));
    }

    /**
     * Get the trial lock cache key.
     */
    protected function getTrialLockKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':trial_lock';
    }

    // Cache key helpers
    protected function getStateKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':state';
    }

    protected function getFailureCountKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':failures';
    }

    protected function getSuccessCountKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':successes';
    }

    protected function getLastFailureKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':last_failure';
    }

    protected function getOpenedAtKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':opened_at';
    }
}
