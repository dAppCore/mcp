<?php

declare(strict_types=1);

namespace Core\Mcp\Tools\Concerns;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mcp\Exceptions\MissingDependencyException;
use Core\Mcp\Services\ToolDependencyService;

/**
 * Trait for tools that validate dependencies before execution.
 *
 * Provides methods to declare and check dependencies inline.
 */
trait ValidatesDependencies
{
    /**
     * Get the dependencies for this tool.
     *
     * Override this method in your tool to declare dependencies.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        return [];
    }

    /**
     * Validate that all dependencies are met.
     *
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     *
     * @throws MissingDependencyException If dependencies are not met
     */
    protected function validateDependencies(array $context = [], array $args = []): void
    {
        $sessionId = $context['session_id'] ?? 'anonymous';

        app(ToolDependencyService::class)->validateDependencies(
            sessionId: $sessionId,
            toolName: $this->name(),
            context: $context,
            args: $args,
        );
    }

    /**
     * Check if all dependencies are met without throwing.
     *
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     */
    protected function dependenciesMet(array $context = [], array $args = []): bool
    {
        $sessionId = $context['session_id'] ?? 'anonymous';

        return app(ToolDependencyService::class)->checkDependencies(
            sessionId: $sessionId,
            toolName: $this->name(),
            context: $context,
            args: $args,
        );
    }

    /**
     * Get list of unmet dependencies.
     *
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     * @return array<ToolDependency>
     */
    protected function getMissingDependencies(array $context = [], array $args = []): array
    {
        $sessionId = $context['session_id'] ?? 'anonymous';

        return app(ToolDependencyService::class)->getMissingDependencies(
            sessionId: $sessionId,
            toolName: $this->name(),
            context: $context,
            args: $args,
        );
    }

    /**
     * Record this tool call for dependency tracking.
     *
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     */
    protected function recordToolCall(array $context = [], array $args = []): void
    {
        $sessionId = $context['session_id'] ?? 'anonymous';

        app(ToolDependencyService::class)->recordToolCall(
            sessionId: $sessionId,
            toolName: $this->name(),
            args: $args,
        );
    }

    /**
     * Create a dependency error response.
     */
    protected function dependencyError(MissingDependencyException $e): array
    {
        return [
            'error' => 'dependency_not_met',
            'message' => $e->getMessage(),
            'missing' => array_map(
                fn (ToolDependency $dep) => [
                    'type' => $dep->type->value,
                    'key' => $dep->key,
                    'description' => $dep->description,
                ],
                $e->missingDependencies
            ),
            'suggested_order' => $e->suggestedOrder,
        ];
    }
}
