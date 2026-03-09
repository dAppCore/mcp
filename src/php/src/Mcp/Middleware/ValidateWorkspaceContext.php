<?php

declare(strict_types=1);

namespace Core\Mcp\Middleware;

use Closure;
use Illuminate\Http\Request;
use Core\Mcp\Context\WorkspaceContext;
use Core\Mcp\Exceptions\MissingWorkspaceContextException;
use Symfony\Component\HttpFoundation\Response;

/**
 * Middleware that validates workspace context for MCP API requests.
 *
 * This middleware ensures that workspace-scoped MCP tools have proper
 * authentication context. It creates a WorkspaceContext object from
 * the authenticated workspace and stores it for downstream use.
 *
 * SECURITY: This prevents cross-tenant data leakage by ensuring
 * workspace context comes from authentication, not user-supplied parameters.
 */
class ValidateWorkspaceContext
{
    /**
     * Handle an incoming request.
     *
     * @param  string  $mode  'required' or 'optional'
     */
    public function handle(Request $request, Closure $next, string $mode = 'required'): Response
    {
        $workspace = $request->attributes->get('mcp_workspace');

        if ($workspace) {
            // Create workspace context and store it
            $context = WorkspaceContext::fromWorkspace($workspace);
            $request->attributes->set('mcp_workspace_context', $context);

            return $next($request);
        }

        // Try to get workspace from API key
        $apiKey = $request->attributes->get('api_key');
        if ($apiKey?->workspace_id) {
            $context = new WorkspaceContext(
                workspaceId: $apiKey->workspace_id,
                workspace: $apiKey->workspace,
            );
            $request->attributes->set('mcp_workspace_context', $context);

            return $next($request);
        }

        // Try authenticated user's default workspace
        $user = $request->user();
        if ($user && method_exists($user, 'defaultHostWorkspace')) {
            $workspace = $user->defaultHostWorkspace();
            if ($workspace) {
                $context = WorkspaceContext::fromWorkspace($workspace);
                $request->attributes->set('mcp_workspace_context', $context);

                return $next($request);
            }
        }

        // If mode is 'required', reject the request
        if ($mode === 'required') {
            return $this->missingContextResponse($request);
        }

        // Mode is 'optional', continue without context
        return $next($request);
    }

    /**
     * Return response for missing workspace context.
     */
    protected function missingContextResponse(Request $request): Response
    {
        $exception = new MissingWorkspaceContextException('MCP API');

        if ($request->expectsJson() || $request->is('api/*')) {
            return response()->json([
                'error' => $exception->getErrorType(),
                'message' => $exception->getMessage(),
            ], $exception->getStatusCode());
        }

        return response($exception->getMessage(), $exception->getStatusCode());
    }
}
