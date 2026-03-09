<?php

declare(strict_types=1);

namespace Core\Mcp\Context;

use Core\Tenant\Models\Workspace;
use Core\Mcp\Exceptions\MissingWorkspaceContextException;

/**
 * Workspace context for MCP tool execution.
 *
 * Holds authenticated workspace information and provides validation.
 * This ensures workspace-scoped tools always have proper context
 * from authentication, not from user-supplied parameters.
 */
final class WorkspaceContext
{
    public function __construct(
        public readonly int $workspaceId,
        public readonly ?Workspace $workspace = null,
    ) {}

    /**
     * Create context from a workspace model.
     */
    public static function fromWorkspace(Workspace $workspace): self
    {
        return new self(
            workspaceId: $workspace->id,
            workspace: $workspace,
        );
    }

    /**
     * Create context from a workspace ID (lazy loads workspace when needed).
     */
    public static function fromId(int $workspaceId): self
    {
        return new self(workspaceId: $workspaceId);
    }

    /**
     * Create context from request attributes.
     *
     * @throws MissingWorkspaceContextException If no workspace context is available
     */
    public static function fromRequest(mixed $request, string $toolName = 'unknown'): self
    {
        // Try to get workspace from request attributes (set by middleware)
        $workspace = $request->attributes->get('mcp_workspace')
            ?? $request->attributes->get('workspace');

        if ($workspace instanceof Workspace) {
            return self::fromWorkspace($workspace);
        }

        // Try to get API key's workspace
        $apiKey = $request->attributes->get('api_key');
        if ($apiKey?->workspace_id) {
            return new self(
                workspaceId: $apiKey->workspace_id,
                workspace: $apiKey->workspace,
            );
        }

        // Try authenticated user's default workspace
        $user = $request->user();
        if ($user && method_exists($user, 'defaultHostWorkspace')) {
            $workspace = $user->defaultHostWorkspace();
            if ($workspace) {
                return self::fromWorkspace($workspace);
            }
        }

        throw new MissingWorkspaceContextException($toolName);
    }

    /**
     * Get the workspace model, loading it if necessary.
     */
    public function getWorkspace(): Workspace
    {
        if ($this->workspace) {
            return $this->workspace;
        }

        return Workspace::findOrFail($this->workspaceId);
    }

    /**
     * Check if this context has a specific workspace ID.
     */
    public function hasWorkspaceId(int $workspaceId): bool
    {
        return $this->workspaceId === $workspaceId;
    }

    /**
     * Validate that a resource belongs to this workspace.
     *
     * @throws \RuntimeException If the resource doesn't belong to this workspace
     */
    public function validateOwnership(int $resourceWorkspaceId, string $resourceType = 'resource'): void
    {
        if ($resourceWorkspaceId !== $this->workspaceId) {
            throw new \RuntimeException(
                "Access denied: {$resourceType} does not belong to the authenticated workspace."
            );
        }
    }
}
