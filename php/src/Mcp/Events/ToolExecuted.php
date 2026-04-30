<?php

declare(strict_types=1);

namespace Core\Mcp\Events;

use Illuminate\Foundation\Events\Dispatchable;
use Illuminate\Queue\SerializesModels;

/**
 * Event fired when an MCP tool execution completes.
 *
 * This event can be dispatched after tool execution to trigger
 * analytics recording and other side effects.
 */
class ToolExecuted
{
    use Dispatchable, SerializesModels;

    public function __construct(
        public readonly string $toolName,
        public readonly int $durationMs,
        public readonly bool $success,
        public readonly ?string $workspaceId = null,
        public readonly ?string $sessionId = null,
        public readonly ?string $errorMessage = null,
        public readonly ?string $errorCode = null,
        public readonly ?array $metadata = null,
    ) {}

    /**
     * Create event for a successful execution.
     */
    public static function success(
        string $toolName,
        int $durationMs,
        ?string $workspaceId = null,
        ?string $sessionId = null,
        ?array $metadata = null
    ): self {
        return new self(
            toolName: $toolName,
            durationMs: $durationMs,
            success: true,
            workspaceId: $workspaceId,
            sessionId: $sessionId,
            metadata: $metadata,
        );
    }

    /**
     * Create event for a failed execution.
     */
    public static function failure(
        string $toolName,
        int $durationMs,
        ?string $errorMessage = null,
        ?string $errorCode = null,
        ?string $workspaceId = null,
        ?string $sessionId = null,
        ?array $metadata = null
    ): self {
        return new self(
            toolName: $toolName,
            durationMs: $durationMs,
            success: false,
            workspaceId: $workspaceId,
            sessionId: $sessionId,
            errorMessage: $errorMessage,
            errorCode: $errorCode,
            metadata: $metadata,
        );
    }

    /**
     * Get the tool name.
     */
    public function getToolName(): string
    {
        return $this->toolName;
    }

    /**
     * Get the duration in milliseconds.
     */
    public function getDurationMs(): int
    {
        return $this->durationMs;
    }

    /**
     * Check if the execution was successful.
     */
    public function wasSuccessful(): bool
    {
        return $this->success;
    }

    /**
     * Get the workspace ID.
     */
    public function getWorkspaceId(): ?string
    {
        return $this->workspaceId;
    }

    /**
     * Get the session ID.
     */
    public function getSessionId(): ?string
    {
        return $this->sessionId;
    }
}
