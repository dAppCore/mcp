<?php

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Exception thrown when the circuit breaker is open and no fallback is provided.
 *
 * This indicates the target service is temporarily unavailable due to repeated failures.
 */
class CircuitOpenException extends RuntimeException
{
    public function __construct(
        public readonly string $service,
        string $message = '',
    ) {
        $message = $message ?: sprintf(
            "Service '%s' is temporarily unavailable. Please try again later.",
            $service
        );

        parent::__construct($message);
    }
}
