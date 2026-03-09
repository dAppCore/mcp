<?php

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Exception thrown when a SQL query is forbidden by security policies.
 *
 * This indicates the query failed validation due to:
 * - Containing disallowed SQL keywords (UNION, INSERT, UPDATE, DELETE, etc.)
 * - Not matching any whitelisted query pattern
 * - Containing potentially malicious constructs (stacked queries, comments)
 */
class ForbiddenQueryException extends RuntimeException
{
    public function __construct(
        public readonly string $query,
        public readonly string $reason,
        string $message = '',
    ) {
        $message = $message ?: sprintf(
            'Query rejected: %s',
            $reason
        );

        parent::__construct($message);
    }

    /**
     * Create exception for disallowed keyword.
     */
    public static function disallowedKeyword(string $query, string $keyword): self
    {
        return new self(
            $query,
            sprintf("Disallowed SQL keyword '%s' detected", strtoupper($keyword))
        );
    }

    /**
     * Create exception for query not matching whitelist.
     */
    public static function notWhitelisted(string $query): self
    {
        return new self(
            $query,
            'Query does not match any allowed pattern'
        );
    }

    /**
     * Create exception for invalid query structure.
     */
    public static function invalidStructure(string $query, string $detail): self
    {
        return new self(
            $query,
            sprintf('Invalid query structure: %s', $detail)
        );
    }
}
