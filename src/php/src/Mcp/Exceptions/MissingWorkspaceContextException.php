<?php

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Exception thrown when an MCP tool requires workspace context but none is provided.
 *
 * This is a security measure to prevent cross-tenant data leakage.
 * Workspace-scoped tools must have explicit workspace context from authentication,
 * not from user-supplied parameters.
 */
class MissingWorkspaceContextException extends RuntimeException
{
    public function __construct(
        public readonly string $tool,
        string $message = '',
    ) {
        $message = $message ?: sprintf(
            "MCP tool '%s' requires workspace context. Authenticate with an API key or user session.",
            $tool
        );

        parent::__construct($message);
    }

    /**
     * Get the HTTP status code for this exception.
     */
    public function getStatusCode(): int
    {
        return 403;
    }

    /**
     * Get the error type for JSON responses.
     */
    public function getErrorType(): string
    {
        return 'missing_workspace_context';
    }
}
