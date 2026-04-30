<?php

declare(strict_types=1);

namespace Core\Mcp\Middleware;

use Closure;
use Core\Mcp\Exceptions\MissingDependencyException;
use Core\Mcp\Services\ToolDependencyService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

/**
 * Middleware to validate tool dependencies before execution.
 *
 * Checks that all prerequisites are met for an MCP tool call
 * and returns a helpful error response if not.
 */
class ValidateToolDependencies
{
    public function __construct(
        protected ToolDependencyService $dependencyService,
    ) {}

    /**
     * Handle an incoming request.
     */
    public function handle(Request $request, Closure $next): mixed
    {
        // Only validate tool call endpoints
        if (! $this->isToolCallRequest($request)) {
            return $next($request);
        }

        $toolName = $this->extractToolName($request);
        $sessionId = $this->extractSessionId($request);
        $context = $this->extractContext($request);
        $args = $this->extractArguments($request);

        if (! $toolName || ! $sessionId) {
            return $next($request);
        }

        try {
            $this->dependencyService->validateDependencies($sessionId, $toolName, $context, $args);
        } catch (MissingDependencyException $e) {
            return $this->buildErrorResponse($e);
        }

        // Record the tool call after successful execution
        $response = $next($request);

        // Only record on success
        if ($response instanceof JsonResponse && $this->isSuccessResponse($response)) {
            $this->dependencyService->recordToolCall($sessionId, $toolName, $args);
        }

        return $response;
    }

    /**
     * Check if this is a tool call request.
     */
    protected function isToolCallRequest(Request $request): bool
    {
        return $request->is('*/tools/call') || $request->is('api/*/mcp/tools/call');
    }

    /**
     * Extract the tool name from the request.
     */
    protected function extractToolName(Request $request): ?string
    {
        return $request->input('tool') ?? $request->input('name');
    }

    /**
     * Extract the session ID from the request.
     */
    protected function extractSessionId(Request $request): ?string
    {
        // Try various locations where session ID might be
        return $request->input('session_id')
            ?? $request->input('arguments.session_id')
            ?? $request->header('X-MCP-Session-ID')
            ?? $request->attributes->get('session_id');
    }

    /**
     * Extract context from the request.
     */
    protected function extractContext(Request $request): array
    {
        $context = [];

        // Get API key context
        $apiKey = $request->attributes->get('api_key');
        if ($apiKey) {
            $context['workspace_id'] = $apiKey->workspace_id;
        }

        // Get explicit context from request
        $requestContext = $request->input('context', []);
        if (is_array($requestContext)) {
            $context = array_merge($context, $requestContext);
        }

        // Get session ID
        $sessionId = $this->extractSessionId($request);
        if ($sessionId) {
            $context['session_id'] = $sessionId;
        }

        return $context;
    }

    /**
     * Extract tool arguments from the request.
     */
    protected function extractArguments(Request $request): array
    {
        return $request->input('arguments', []) ?? [];
    }

    /**
     * Check if response indicates success.
     */
    protected function isSuccessResponse(JsonResponse $response): bool
    {
        if ($response->getStatusCode() >= 400) {
            return false;
        }

        $data = $response->getData(true);

        return ($data['success'] ?? true) !== false;
    }

    /**
     * Build error response for missing dependencies.
     */
    protected function buildErrorResponse(MissingDependencyException $e): JsonResponse
    {
        return response()->json($e->toApiResponse(), 422);
    }
}
