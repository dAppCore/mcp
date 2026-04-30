<?php

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Exception thrown when a SQL query exceeds the allowed execution time.
 *
 * This indicates the query was terminated due to:
 * - Exceeding the configured timeout limit
 * - Potentially expensive or malicious query patterns
 */
class QueryTimeoutException extends RuntimeException
{
    public function __construct(
        public readonly string $query,
        public readonly int $timeoutSeconds,
        string $message = '',
    ) {
        $message = $message ?: sprintf(
            'Query execution exceeded timeout of %d seconds',
            $timeoutSeconds
        );

        parent::__construct($message);
    }

    /**
     * Create exception for timeout.
     */
    public static function exceeded(string $query, int $timeoutSeconds): self
    {
        return new self(
            $query,
            $timeoutSeconds,
            sprintf(
                'Query execution timed out after %d seconds. Consider optimising the query or adding appropriate indexes.',
                $timeoutSeconds
            )
        );
    }
}
