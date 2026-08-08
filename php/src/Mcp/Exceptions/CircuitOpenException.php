<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Thrown by CircuitBreaker when a downstream service has tripped its
 * failure threshold and the circuit is open. The exception carries the
 * service identifier so callers can produce a user-readable retry hint
 * without parsing the message.
 *
 * Example:
 *
 *   try {
 *       $breaker->call('openbrain', fn () => $client->dispatch(...));
 *   } catch (CircuitOpenException $e) {
 *       return ['error' => "service '{$e->service}' temporarily unavailable"];
 *   }
 */
final class CircuitOpenException extends RuntimeException
{
    /**
     * Construct an open-circuit exception. When $message is empty, a
     * default human-readable message is generated from $service.
     *
     * Example:
     *
     *   throw new CircuitOpenException('openbrain');
     */
    public function __construct(
        public readonly string $service,
        string $message = '',
    ) {
        parent::__construct($message !== '' ? $message : sprintf(
            "Service '%s' is temporarily unavailable. Please try again later.",
            $service,
        ));
    }
}
