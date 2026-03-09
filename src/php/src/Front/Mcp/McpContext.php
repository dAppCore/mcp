<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Front\Mcp;

use Closure;

/**
 * Context object passed to MCP tool handlers.
 *
 * Abstracts the transport layer (stdio vs HTTP) so tool handlers
 * can work with either transport without modification.
 *
 * Provides access to:
 * - Current session tracking
 * - Current plan context
 * - Notification sending
 * - Session logging
 */
class McpContext
{
    /**
     * @param  object|null  $currentPlan  AgentPlan model instance when Agentic module installed
     */
    public function __construct(
        private ?string $sessionId = null,
        private ?object $currentPlan = null,
        private ?Closure $notificationCallback = null,
        private ?Closure $logCallback = null,
    ) {}

    /**
     * Get the current session ID if one is active.
     */
    public function getSessionId(): ?string
    {
        return $this->sessionId;
    }

    /**
     * Set the current session ID.
     */
    public function setSessionId(?string $sessionId): void
    {
        $this->sessionId = $sessionId;
    }

    /**
     * Get the current plan if one is active.
     *
     * @return object|null AgentPlan model instance when Agentic module installed
     */
    public function getCurrentPlan(): ?object
    {
        return $this->currentPlan;
    }

    /**
     * Set the current plan.
     *
     * @param  object|null  $plan  AgentPlan model instance
     */
    public function setCurrentPlan(?object $plan): void
    {
        $this->currentPlan = $plan;
    }

    /**
     * Send an MCP notification to the client.
     *
     * Notifications are one-way messages that don't expect a response.
     * Common notifications include progress updates, log messages, etc.
     */
    public function sendNotification(string $method, array $params = []): void
    {
        if ($this->notificationCallback) {
            ($this->notificationCallback)($method, $params);
        }
    }

    /**
     * Log a message to the current session.
     *
     * Messages are recorded in the session log for handoff context
     * and audit trail purposes.
     */
    public function logToSession(string $message, string $type = 'info', array $data = []): void
    {
        if ($this->logCallback) {
            ($this->logCallback)($message, $type, $data);
        }
    }

    /**
     * Set the notification callback.
     */
    public function setNotificationCallback(?Closure $callback): void
    {
        $this->notificationCallback = $callback;
    }

    /**
     * Set the log callback.
     */
    public function setLogCallback(?Closure $callback): void
    {
        $this->logCallback = $callback;
    }

    /**
     * Check if a session is currently active.
     */
    public function hasSession(): bool
    {
        return $this->sessionId !== null;
    }

    /**
     * Check if a plan is currently active.
     */
    public function hasPlan(): bool
    {
        return $this->currentPlan !== null;
    }
}
