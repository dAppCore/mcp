<?php

declare(strict_types=1);

namespace Core\Mcp\Middleware;

use Core\Tenant\Models\Workspace;
use Core\Tenant\Services\EntitlementService;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * MCP Portal Authentication Middleware.
 *
 * Handles authentication for the MCP portal:
 * - Public routes (landing, server list) pass through
 * - Protected routes require auth or API key
 * - Checks mcp.access entitlement for workspace
 */
class McpAuthenticate
{
    public function __construct(
        protected EntitlementService $entitlementService
    ) {}

    /**
     * Handle an incoming request.
     */
    public function handle(Request $request, Closure $next, string $level = 'optional'): Response
    {
        // Try API key auth first (for programmatic access)
        $workspace = $this->authenticateByApiKey($request);

        // Fall back to session auth
        if (! $workspace && $request->user()) {
            $user = $request->user();
            if (method_exists($user, 'defaultHostWorkspace')) {
                $workspace = $user->defaultHostWorkspace();
            }
        }

        // Store workspace for downstream use
        if ($workspace) {
            $request->attributes->set('mcp_workspace', $workspace);

            // Check MCP access entitlement
            $result = $this->entitlementService->can($workspace, 'mcp.access');
            $request->attributes->set('mcp_entitlement', $result);
        }

        // For 'required' level, must have workspace
        if ($level === 'required' && ! $workspace) {
            return $this->unauthenticatedResponse($request);
        }

        return $next($request);
    }

    /**
     * Authenticate using API key from header or query.
     */
    protected function authenticateByApiKey(Request $request): ?Workspace
    {
        $apiKey = $request->header('X-API-Key')
            ?? $request->header('Authorization')
            ?? $request->query('api_key');

        if (! $apiKey) {
            return null;
        }

        // Strip 'Bearer ' prefix if present
        if (str_starts_with($apiKey, 'Bearer ')) {
            $apiKey = substr($apiKey, 7);
        }

        // Try AgentApiKey authentication (agentic module)
        try {
            $agentKeyService = app(\Core\Mod\Agentic\Services\AgentApiKeyService::class);
            $agentKey = $agentKeyService->validate($apiKey);
            if ($agentKey) {
                return Workspace::find($agentKey->workspace_id);
            }
        } catch (\Throwable) {
            // AgentApiKeyService not available, skip
        }

        // Fall back to workspace apiKeys relationship
        try {
            return Workspace::whereHas('apiKeys', function ($query) use ($apiKey) {
                $query->where('key', hash('sha256', $apiKey))
                    ->where(function ($q) {
                        $q->whereNull('expires_at')
                            ->orWhere('expires_at', '>', now());
                    });
            })->first();
        } catch (\Throwable) {
            // ApiKey model not available (package removed)
            return null;
        }
    }

    /**
     * Return unauthenticated response.
     */
    protected function unauthenticatedResponse(Request $request): Response
    {
        if ($request->expectsJson() || $request->is('api/*')) {
            return response()->json([
                'error' => 'unauthenticated',
                'message' => 'Authentication required. Provide an API key or sign in.',
            ], 401);
        }

        return redirect()->guest(route('login'));
    }
}
