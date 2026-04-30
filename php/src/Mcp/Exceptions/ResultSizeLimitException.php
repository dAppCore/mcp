<?php

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Exception thrown when query results exceed the allowed size limit.
 *
 * This indicates the result set was truncated due to:
 * - Exceeding the configured maximum rows per tier
 * - Data exfiltration prevention measures
 */
class ResultSizeLimitException extends RuntimeException
{
    public function __construct(
        public readonly int $actualRows,
        public readonly int $maxRows,
        public readonly string $tier,
        string $message = '',
    ) {
        $message = $message ?: sprintf(
            'Result set truncated: returned %d rows (limit: %d for tier "%s")',
            min($actualRows, $maxRows),
            $maxRows,
            $tier
        );

        parent::__construct($message);
    }

    /**
     * Create exception for result truncation.
     */
    public static function truncated(int $actualRows, int $maxRows, string $tier): self
    {
        return new self(
            $actualRows,
            $maxRows,
            $tier,
            sprintf(
                'Query returned more rows than allowed. Results truncated to %d rows (your tier "%s" limit). Consider adding more specific filters.',
                $maxRows,
                $tier
            )
        );
    }
}
