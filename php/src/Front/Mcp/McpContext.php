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
use Illuminate\Http\Request;

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
        private array|object|null $scopeSource = null,
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

    // ---------------------------------------------------------------------
    // Scope resolution
    //
    // Adopted from dappcore/agent, which declared this same FQCN in its own
    // tree and carried these two methods plus their helpers. Two packages
    // owning one class name is decided by autoload order, and this copy wins in
    // a consumer — so agent's scope support was unreachable wherever it
    // mattered, and a caller reaching hasScope() on the loaded class would have
    // hit an undefined method. The methods come here, agent's file goes, and
    // the FQCN has one owner.
    // ---------------------------------------------------------------------

    /**
     * @return array<int, string>
     */
    public function getScopes(): array
    {
        foreach ($this->scopeCandidates() as $candidate) {
            $scopes = $this->resolveScopes($candidate);

            if ($scopes !== []) {
                return $scopes;
            }
        }

        return [];
    }

    public function hasScope(string $scope): bool
    {
        $wanted = $this->canonicalScope($scope);
        if ($wanted === null) {
            return false;
        }

        foreach ($this->getScopes() as $grantedScope) {
            if ($this->canonicalScope($grantedScope) === $wanted) {
                return true;
            }
        }

        return false;
    }

    /**
     * @return array<int, array|object>
     */
    private function scopeCandidates(): array
    {
        $candidates = [];

        if (is_array($this->scopeSource) || is_object($this->scopeSource)) {
            $candidates[] = $this->scopeSource;
        }

        if ($this->currentPlan !== null) {
            $candidates[] = $this->currentPlan;
        }

        $request = $this->currentRequest();
        if (! $request instanceof Request) {
            return $candidates;
        }

        $requestContext = $request->attributes->get('mcp_workspace_context');
        if (is_array($requestContext) || is_object($requestContext)) {
            $candidates[] = $requestContext;
        }

        foreach (['agent_api_key', 'api_key'] as $attribute) {
            $value = $request->attributes->get($attribute);

            if (is_array($value) || is_object($value)) {
                $candidates[] = $value;
            }
        }

        return $candidates;
    }

    /**
     * @return array<int, string>
     */
    private function resolveScopes(mixed $source, int $depth = 0): array
    {
        if ($depth > 3 || $source === null || $source === $this) {
            return [];
        }

        if (is_string($source)) {
            return $this->normaliseScopes([$source]);
        }

        if (is_array($source)) {
            if (array_is_list($source)) {
                return $this->normaliseScopes($source);
            }

            foreach (['scopes', 'permissions', 'authorised_scopes', 'authorized_scopes'] as $key) {
                if (! array_key_exists($key, $source)) {
                    continue;
                }

                $scopes = $this->resolveScopes($source[$key], $depth + 1);
                if ($scopes !== []) {
                    return $scopes;
                }
            }

            foreach (['api_key', 'agent_api_key', 'apiKey', 'agentApiKey', 'session', 'auth', 'authorisation', 'authorization'] as $key) {
                if (! array_key_exists($key, $source)) {
                    continue;
                }

                $scopes = $this->resolveScopes($source[$key], $depth + 1);
                if ($scopes !== []) {
                    return $scopes;
                }
            }

            return [];
        }

        if (! is_object($source)) {
            return [];
        }

        foreach (['getScopes', 'getPermissions'] as $method) {
            if (! method_exists($source, $method)) {
                continue;
            }

            $scopes = $this->resolveScopes($source->{$method}(), $depth + 1);
            if ($scopes !== []) {
                return $scopes;
            }
        }

        foreach (['scopes', 'permissions', 'authorised_scopes', 'authorized_scopes'] as $key) {
            $scopes = $this->resolveScopes($this->extractObjectValue($source, $key), $depth + 1);
            if ($scopes !== []) {
                return $scopes;
            }
        }

        foreach (['apiKey', 'agentApiKey', 'api_key', 'agent_api_key', 'session', 'auth', 'authorisation', 'authorization'] as $key) {
            $scopes = $this->resolveScopes($this->extractObjectValue($source, $key), $depth + 1);
            if ($scopes !== []) {
                return $scopes;
            }
        }

        return [];
    }

    private function extractObjectValue(object $source, string $key): mixed
    {
        if (method_exists($source, 'getAttribute')) {
            $value = $source->getAttribute($key);

            if ($value !== null) {
                return $value;
            }
        }

        foreach ($this->getterNames($key) as $getter) {
            if (! method_exists($source, $getter)) {
                continue;
            }

            $value = $source->{$getter}();
            if ($value !== null) {
                return $value;
            }
        }

        // get_object_vars() from out here returns only what is actually
        // reachable, so it is asked first. property_exists() answers true for
        // private and protected properties too, and reading one of those from
        // outside the class raises "Cannot access private property" — which is
        // what this did to any scope source holding its data privately, the
        // commonest shape there is.
        $vars = get_object_vars($source);
        if (array_key_exists($key, $vars)) {
            return $vars[$key];
        }

        if (isset($source->{$key})) {
            return $source->{$key};
        }

        return null;
    }

    /**
     * @return array<int, string>
     */
    private function getterNames(string $key): array
    {
        $segments = array_filter(explode('_', $key), fn (string $segment): bool => $segment !== '');
        $studly = implode('', array_map(static fn (string $segment): string => ucfirst($segment), $segments));

        $names = ['get'.ucfirst($key)];
        if ($studly !== '') {
            $names[] = 'get'.$studly;
        }

        return array_values(array_unique($names));
    }

    private function currentRequest(): ?Request
    {
        if (! function_exists('app') || ! class_exists(Request::class) || ! app()->bound('request')) {
            return null;
        }

        $request = app('request');

        return $request instanceof Request ? $request : null;
    }

    /**
     * @param  array<int, mixed>  $scopes
     * @return array<int, string>
     */
    private function normaliseScopes(array $scopes): array
    {
        $normalised = [];
        $seen = [];

        foreach ($scopes as $scope) {
            if (! is_string($scope)) {
                continue;
            }

            $cleanScope = trim($scope);
            $canonicalScope = $this->canonicalScope($cleanScope);

            if ($canonicalScope === null || isset($seen[$canonicalScope])) {
                continue;
            }

            $seen[$canonicalScope] = true;
            $normalised[] = $cleanScope;
        }

        return $normalised;
    }

    private function canonicalScope(string $scope): ?string
    {
        $cleanScope = trim($scope);

        if ($cleanScope === '') {
            return null;
        }

        return str_replace(':', '.', $cleanScope);
    }
}
