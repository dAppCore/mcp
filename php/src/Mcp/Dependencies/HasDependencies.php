<?php

declare(strict_types=1);

namespace Core\Mcp\Dependencies;

/**
 * Interface for tools that declare dependencies.
 *
 * Tools implementing this interface can specify prerequisites
 * that must be satisfied before execution.
 */
interface HasDependencies
{
    /**
     * Get the dependencies for this tool.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array;
}
