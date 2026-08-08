<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Closure;
use Core\Mcp\Exceptions\CircuitOpenException;
use Illuminate\Support\Facades\Cache;
use Throwable;

/**
 * Cache-backed circuit breaker for downstream service calls. Counts
 * consecutive failures per service and trips OPEN when the threshold
 * is exceeded; HALF_OPEN allows a probe call after a cooldown to test
 * recovery; CLOSED is the normal pass-through state.
 *
 * Example:
 *
 *   $breaker = new CircuitBreaker();
 *   $result = $breaker->call('openbrain', fn () => $client->dispatch(...));
 */
final class CircuitBreaker
{
    protected const CACHE_PREFIX = 'circuit_breaker:';

    protected const COUNTER_TTL = 300;

    public const STATE_CLOSED = 'closed';

    public const STATE_OPEN = 'open';

    public const STATE_HALF_OPEN = 'half_open';

    /**
     * Execute $operation under circuit-breaker protection. When the
     * circuit is OPEN, $fallback is invoked (or CircuitOpenException
     * is thrown if no fallback is supplied). Failures increment the
     * service's failure counter and may trip the breaker.
     *
     * Example:
     *
     *   $breaker->call('openbrain', $op, fn () => ['cached' => true]);
     */
    public function call(string $service, Closure $operation, ?Closure $fallback = null): mixed
    {
        $state = $this->getState($service);

        if ($state === self::STATE_OPEN) {
            if ($fallback !== null) {
                return $fallback();
            }

            throw new CircuitOpenException($service);
        }

        $hasTrialLock = false;

        if ($state === self::STATE_HALF_OPEN) {
            $hasTrialLock = $this->acquireTrialLock($service);

            if (! $hasTrialLock) {
                if ($fallback !== null) {
                    return $fallback();
                }

                throw new CircuitOpenException(
                    $service,
                    sprintf("Service '%s' is being tested. Please try again shortly.", $service),
                );
            }
        }

        try {
            $result = $operation();

            $this->recordSuccess($service);

            return $result;
        } catch (Throwable $throwable) {
            $this->recordFailure($service, $throwable);

            if ($this->shouldTrip($service)) {
                $this->tripCircuit($service);
            }

            if ($fallback !== null && $this->isRecoverableError($throwable)) {
                return $fallback();
            }

            throw $throwable;
        } finally {
            if ($hasTrialLock) {
                $this->releaseTrialLock($service);
            }
        }
    }

    public function getState(string $service): string
    {
        $state = Cache::get($this->stateKey($service));

        if (! is_string($state) || $state === '') {
            return self::STATE_CLOSED;
        }

        if ($state === self::STATE_OPEN) {
            $openedAt = (int) Cache::get($this->openedAtKey($service), 0);

            if ($openedAt > 0 && (time() - $openedAt) >= $this->resetTimeout($service)) {
                $this->setState($service, self::STATE_HALF_OPEN);

                return self::STATE_HALF_OPEN;
            }
        }

        return $state;
    }

    public function getStats(string $service): array
    {
        return [
            'service' => $service,
            'state' => $this->getState($service),
            'failures' => (int) Cache::get($this->failureCountKey($service), 0),
            'successes' => (int) Cache::get($this->successCountKey($service), 0),
            'last_failure' => Cache::get($this->lastFailureKey($service)),
            'opened_at' => Cache::get($this->openedAtKey($service)),
            'threshold' => $this->failureThreshold($service),
            'reset_timeout' => $this->resetTimeout($service),
        ];
    }

    public function reset(string $service): void
    {
        $this->setState($service, self::STATE_CLOSED);
        Cache::forget($this->failureCountKey($service));
        Cache::forget($this->successCountKey($service));
        Cache::forget($this->lastFailureKey($service));
        Cache::forget($this->openedAtKey($service));
        Cache::forget($this->trialLockKey($service));
    }

    public function isAvailable(string $service): bool
    {
        return $this->getState($service) !== self::STATE_OPEN;
    }

    protected function recordSuccess(string $service): void
    {
        $state = $this->getState($service);

        $this->atomicIncrement($this->successCountKey($service), self::COUNTER_TTL);

        if ($state === self::STATE_HALF_OPEN) {
            $this->closeCircuit($service);
        }

        $this->atomicDecrement($this->failureCountKey($service));
    }

    protected function recordFailure(string $service, Throwable $throwable): void
    {
        $window = $this->failureWindow($service);
        $failures = $this->atomicIncrement($this->failureCountKey($service), $window);

        Cache::put($this->lastFailureKey($service), [
            'message' => $throwable->getMessage(),
            'class' => $throwable::class,
            'time' => now()->toIso8601String(),
            'failures' => $failures,
        ], $window);
    }

    protected function shouldTrip(string $service): bool
    {
        return (int) Cache::get($this->failureCountKey($service), 0) >= $this->failureThreshold($service);
    }

    protected function tripCircuit(string $service): void
    {
        $this->setState($service, self::STATE_OPEN);
        Cache::put($this->openedAtKey($service), time(), 86400);
    }

    protected function closeCircuit(string $service): void
    {
        $this->setState($service, self::STATE_CLOSED);
        Cache::forget($this->failureCountKey($service));
        Cache::forget($this->openedAtKey($service));
        Cache::forget($this->trialLockKey($service));
    }

    protected function setState(string $service, string $state): void
    {
        Cache::put($this->stateKey($service), $state, 86400);
    }

    protected function isRecoverableError(Throwable $throwable): bool
    {
        $patterns = [
            'SQLSTATE',
            'Connection refused',
            'Table .* doesn\'t exist',
            'Base table or view not found',
            'Connection timed out',
            'Too many connections',
        ];

        foreach ($patterns as $pattern) {
            if (preg_match('/'.$pattern.'/i', $throwable->getMessage()) === 1) {
                return true;
            }
        }

        return false;
    }

    protected function failureThreshold(string $service): int
    {
        return (int) config(
            sprintf('mcp.circuit_breaker.%s.threshold', $service),
            config('mcp.circuit_breaker.default_threshold', 5),
        );
    }

    protected function resetTimeout(string $service): int
    {
        return (int) config(
            sprintf('mcp.circuit_breaker.%s.reset_timeout', $service),
            config('mcp.circuit_breaker.default_reset_timeout', 60),
        );
    }

    protected function failureWindow(string $service): int
    {
        return (int) config(
            sprintf('mcp.circuit_breaker.%s.failure_window', $service),
            config('mcp.circuit_breaker.default_failure_window', 120),
        );
    }

    protected function atomicIncrement(string $key, int $ttl): int
    {
        $lock = Cache::lock($key.':lock', 5);

        try {
            $lock->block(3);

            $value = (int) Cache::get($key, 0) + 1;
            Cache::put($key, $value, $ttl);

            return $value;
        } finally {
            rescue(static fn (): mixed => $lock->release(), report: false);
        }
    }

    protected function atomicDecrement(string $key): int
    {
        $lock = Cache::lock($key.':lock', 5);

        try {
            $lock->block(3);

            $value = max((int) Cache::get($key, 0) - 1, 0);
            Cache::put($key, $value, self::COUNTER_TTL);

            return $value;
        } finally {
            rescue(static fn (): mixed => $lock->release(), report: false);
        }
    }

    protected function acquireTrialLock(string $service): bool
    {
        return Cache::add($this->trialLockKey($service), true, 30);
    }

    protected function releaseTrialLock(string $service): void
    {
        Cache::forget($this->trialLockKey($service));
    }

    protected function stateKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':state';
    }

    protected function failureCountKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':failures';
    }

    protected function successCountKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':successes';
    }

    protected function lastFailureKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':last_failure';
    }

    protected function openedAtKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':opened_at';
    }

    protected function trialLockKey(string $service): string
    {
        return self::CACHE_PREFIX.$service.':trial_lock';
    }
}
