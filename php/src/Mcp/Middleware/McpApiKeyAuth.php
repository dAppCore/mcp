<?php

declare(strict_types=1);

namespace Core\Mcp\Middleware;

use Core\Api\Models\ApiKey;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * API Key authentication for MCP HTTP API.
 *
 * Supports:
 * - Authorization: Bearer hk_xxx_yyy
 * - X-API-Key: hk_xxx_yyy
 *
 * Also enforces per-server access scopes on tool execution.
 */
class McpApiKeyAuth
{
    public function handle(Request $request, Closure $next): Response
    {
        $key = $this->extractKey($request);

        if (! $key) {
            return response()->json([
                'error' => 'Missing API key',
                'hint' => 'Provide via Authorization: Bearer <key> or X-API-Key header',
            ], 401);
        }

        $apiKey = ApiKey::findByPlainKey($key);

        if (! $apiKey) {
            return response()->json([
                'error' => 'Invalid API key',
            ], 401);
        }

        if ($apiKey->isExpired()) {
            return response()->json([
                'error' => 'API key has expired',
            ], 401);
        }

        // Check server-level access for tool calls
        if ($request->is('*/tools/call') && $request->isMethod('POST')) {
            $serverId = $request->input('server');
            if ($serverId && ! $apiKey->hasServerAccess($serverId)) {
                return response()->json([
                    'error' => 'Access denied to server: '.$serverId,
                    'allowed_servers' => $apiKey->getAllowedServers(),
                ], 403);
            }
        }

        // Record usage
        $apiKey->recordUsage();

        // Attach to request for controller access
        $request->attributes->set('api_key', $apiKey);
        $request->attributes->set('workspace', $apiKey->workspace);

        return $next($request);
    }

    protected function extractKey(Request $request): ?string
    {
        // Try Authorization: Bearer
        $authHeader = $request->header('Authorization');
        if ($authHeader && str_starts_with($authHeader, 'Bearer ')) {
            return substr($authHeader, 7);
        }

        // Try X-API-Key
        $apiKeyHeader = $request->header('X-API-Key');
        if ($apiKeyHeader) {
            return $apiKeyHeader;
        }

        return null;
    }
}
