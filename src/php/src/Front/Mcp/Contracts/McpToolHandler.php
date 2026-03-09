<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Front\Mcp\Contracts;

use Core\Front\Mcp\McpContext;

/**
 * Interface for MCP tool handlers.
 *
 * Each MCP tool is implemented as a handler class that provides:
 * - A JSON schema describing the tool for Claude
 * - A handle method that processes tool invocations
 *
 * Tool handlers are registered via the McpToolsRegistering event
 * and can be used by both stdio and HTTP MCP transports.
 */
interface McpToolHandler
{
    /**
     * Get the JSON schema describing this tool.
     *
     * The schema follows the MCP tool specification:
     * - name: Tool identifier (snake_case)
     * - description: What the tool does (for Claude)
     * - inputSchema: JSON Schema for parameters
     *
     * @return array{name: string, description: string, inputSchema: array}
     */
    public static function schema(): array;

    /**
     * Handle a tool invocation.
     *
     * @param  array  $args  Arguments from the tool call
     * @param  McpContext  $context  Server context (session, notifications, etc.)
     * @return array Result to return to Claude
     */
    public function handle(array $args, McpContext $context): array;
}
