<?php

declare(strict_types=1);

namespace Core\Mcp\Listeners;

use Core\Mcp\Services\ToolAnalyticsService;

/**
 * Listener to record MCP tool executions for analytics.
 *
 * Hooks into the MCP tool execution pipeline to track timing,
 * success/failure, and other metrics.
 */
class RecordToolExecution
{
    public function __construct(
        protected ToolAnalyticsService $analyticsService
    ) {}

    /**
     * Handle the tool execution event.
     *
     * @param  object  $event  The tool execution event
     */
    public function handle(object $event): void
    {
        if (! config('mcp.analytics.enabled', true)) {
            return;
        }

        // Extract data from the event
        $toolName = $this->getToolName($event);
        $durationMs = $this->getDuration($event);
        $success = $this->wasSuccessful($event);
        $workspaceId = $this->getWorkspaceId($event);
        $sessionId = $this->getSessionId($event);

        if ($toolName === null || $durationMs === null) {
            return;
        }

        $this->analyticsService->recordExecution(
            tool: $toolName,
            durationMs: $durationMs,
            success: $success,
            workspaceId: $workspaceId,
            sessionId: $sessionId
        );
    }

    /**
     * Extract tool name from the event.
     */
    protected function getToolName(object $event): ?string
    {
        // Support multiple event structures
        if (property_exists($event, 'toolName')) {
            return $event->toolName;
        }

        if (property_exists($event, 'tool_name')) {
            return $event->tool_name;
        }

        if (property_exists($event, 'tool')) {
            return is_string($event->tool) ? $event->tool : $event->tool->getName();
        }

        if (method_exists($event, 'getToolName')) {
            return $event->getToolName();
        }

        return null;
    }

    /**
     * Extract duration from the event.
     */
    protected function getDuration(object $event): ?int
    {
        if (property_exists($event, 'durationMs')) {
            return (int) $event->durationMs;
        }

        if (property_exists($event, 'duration_ms')) {
            return (int) $event->duration_ms;
        }

        if (property_exists($event, 'duration')) {
            return (int) $event->duration;
        }

        if (method_exists($event, 'getDurationMs')) {
            return $event->getDurationMs();
        }

        return null;
    }

    /**
     * Determine if the execution was successful.
     */
    protected function wasSuccessful(object $event): bool
    {
        if (property_exists($event, 'success')) {
            return (bool) $event->success;
        }

        if (property_exists($event, 'error')) {
            return $event->error === null;
        }

        if (property_exists($event, 'exception')) {
            return $event->exception === null;
        }

        if (method_exists($event, 'wasSuccessful')) {
            return $event->wasSuccessful();
        }

        return true; // Assume success if no indicator
    }

    /**
     * Extract workspace ID from the event.
     */
    protected function getWorkspaceId(object $event): ?string
    {
        if (property_exists($event, 'workspaceId')) {
            return $event->workspaceId;
        }

        if (property_exists($event, 'workspace_id')) {
            return $event->workspace_id;
        }

        if (method_exists($event, 'getWorkspaceId')) {
            return $event->getWorkspaceId();
        }

        return null;
    }

    /**
     * Extract session ID from the event.
     */
    protected function getSessionId(object $event): ?string
    {
        if (property_exists($event, 'sessionId')) {
            return $event->sessionId;
        }

        if (property_exists($event, 'session_id')) {
            return $event->session_id;
        }

        if (method_exists($event, 'getSessionId')) {
            return $event->getSessionId();
        }

        return null;
    }
}
