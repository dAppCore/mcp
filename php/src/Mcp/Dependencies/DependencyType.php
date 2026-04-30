<?php

declare(strict_types=1);

namespace Core\Mcp\Dependencies;

/**
 * Types of tool dependencies.
 *
 * Defines how a prerequisite must be satisfied before a tool can execute.
 */
enum DependencyType: string
{
    /**
     * Another tool must have been called in the current session.
     * Example: task_update requires plan_create to have been called.
     */
    case TOOL_CALLED = 'tool_called';

    /**
     * A specific state key must exist in the session context.
     * Example: session_log requires session_id to be set.
     */
    case SESSION_STATE = 'session_state';

    /**
     * A specific context value must be present.
     * Example: workspace_id must exist for workspace-scoped tools.
     */
    case CONTEXT_EXISTS = 'context_exists';

    /**
     * A database entity must exist (checked by ID or slug).
     * Example: task_update requires the plan_slug to reference an existing plan.
     */
    case ENTITY_EXISTS = 'entity_exists';

    /**
     * A custom condition evaluated at runtime.
     * Example: Complex business rules that don't fit other types.
     */
    case CUSTOM = 'custom';

    /**
     * Get a human-readable label for this dependency type.
     */
    public function label(): string
    {
        return match ($this) {
            self::TOOL_CALLED => 'Tool must be called first',
            self::SESSION_STATE => 'Session state required',
            self::CONTEXT_EXISTS => 'Context value required',
            self::ENTITY_EXISTS => 'Entity must exist',
            self::CUSTOM => 'Custom condition',
        };
    }
}
