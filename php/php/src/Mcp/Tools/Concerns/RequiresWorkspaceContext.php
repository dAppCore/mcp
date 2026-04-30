<?php

declare(strict_types=1);

namespace Core\Mcp\Tools\Concerns;

use Core\Tenant\Models\Workspace;
use Core\Mcp\Context\WorkspaceContext;
use Core\Mcp\Exceptions\MissingWorkspaceContextException;

/**
 * Trait for MCP tools that require workspace context.
 *
 * This trait provides methods for validating and retrieving workspace context
 * from the MCP request. Tools using this trait will throw
 * MissingWorkspaceContextException if called without proper context.
 *
 * SECURITY: Workspace context must come from authentication (API key or session),
 * never from user-supplied request parameters.
 */
trait RequiresWorkspaceContext
{
    /**
     * The current workspace context.
     */
    protected ?WorkspaceContext $workspaceContext = null;

    /**
     * Get the tool name for error messages.
     */
    protected function getToolName(): string
    {
        return property_exists($this, 'name') && $this->name
            ? $this->name
            : class_basename(static::class);
    }

    /**
     * Get the workspace context, throwing if not available.
     *
     * @throws MissingWorkspaceContextException
     */
    protected function getWorkspaceContext(): WorkspaceContext
    {
        if ($this->workspaceContext) {
            return $this->workspaceContext;
        }

        throw new MissingWorkspaceContextException($this->getToolName());
    }

    /**
     * Get the workspace ID from context.
     *
     * @throws MissingWorkspaceContextException
     */
    protected function getWorkspaceId(): int
    {
        return $this->getWorkspaceContext()->workspaceId;
    }

    /**
     * Get the workspace model from context.
     *
     * @throws MissingWorkspaceContextException
     */
    protected function getWorkspace(): Workspace
    {
        return $this->getWorkspaceContext()->getWorkspace();
    }

    /**
     * Set the workspace context for this tool execution.
     */
    public function setWorkspaceContext(WorkspaceContext $context): void
    {
        $this->workspaceContext = $context;
    }

    /**
     * Set workspace context from a workspace model.
     */
    public function setWorkspace(Workspace $workspace): void
    {
        $this->workspaceContext = WorkspaceContext::fromWorkspace($workspace);
    }

    /**
     * Set workspace context from a workspace ID.
     */
    public function setWorkspaceId(int $workspaceId): void
    {
        $this->workspaceContext = WorkspaceContext::fromId($workspaceId);
    }

    /**
     * Check if workspace context is available.
     */
    protected function hasWorkspaceContext(): bool
    {
        return $this->workspaceContext !== null;
    }

    /**
     * Validate that a resource belongs to the current workspace.
     *
     * @throws \RuntimeException If the resource doesn't belong to this workspace
     * @throws MissingWorkspaceContextException If no workspace context
     */
    protected function validateResourceOwnership(int $resourceWorkspaceId, string $resourceType = 'resource'): void
    {
        $this->getWorkspaceContext()->validateOwnership($resourceWorkspaceId, $resourceType);
    }

    /**
     * Require workspace context, throwing with a custom message if not available.
     *
     * @throws MissingWorkspaceContextException
     */
    protected function requireWorkspaceContext(string $operation = 'this operation'): WorkspaceContext
    {
        if (! $this->workspaceContext) {
            throw new MissingWorkspaceContextException(
                $this->getToolName(),
                sprintf(
                    "Workspace context is required for %s in tool '%s'. Authenticate with an API key or user session.",
                    $operation,
                    $this->getToolName()
                )
            );
        }

        return $this->workspaceContext;
    }
}
