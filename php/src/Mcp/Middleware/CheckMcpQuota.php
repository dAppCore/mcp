<?php

declare(strict_types=1);

namespace Core\Mcp\Middleware;

use Closure;
use Core\Mcp\Services\McpQuotaService;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Middleware to check MCP workspace quota before processing requests.
 *
 * Enforces monthly tool call and token limits based on workspace entitlements.
 * Adds quota information to response headers.
 */
class CheckMcpQuota
{
    public function __construct(
        protected McpQuotaService $quotaService
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        $workspace = $request->attributes->get('workspace');

        // No workspace context = skip quota check (other middleware handles auth)
        if (! $workspace) {
            return $next($request);
        }

        // Check quota
        $quotaCheck = $this->quotaService->checkQuotaDetailed($workspace);

        if (! $quotaCheck['allowed']) {
            return $this->quotaExceededResponse($quotaCheck, $workspace);
        }

        // Process request
        $response = $next($request);

        // Add quota headers to response
        $this->addQuotaHeaders($response, $workspace);

        return $response;
    }

    /**
     * Build quota exceeded error response.
     */
    protected function quotaExceededResponse(array $quotaCheck, $workspace): Response
    {
        $headers = $this->quotaService->getQuotaHeaders($workspace);

        $errorData = [
            'error' => 'quota_exceeded',
            'message' => $quotaCheck['reason'] ?? 'Monthly quota exceeded',
            'quota' => [
                'tool_calls' => [
                    'used' => $quotaCheck['tool_calls']['used'] ?? 0,
                    'limit' => $quotaCheck['tool_calls']['limit'],
                    'unlimited' => $quotaCheck['tool_calls']['unlimited'] ?? false,
                ],
                'tokens' => [
                    'used' => $quotaCheck['tokens']['used'] ?? 0,
                    'limit' => $quotaCheck['tokens']['limit'],
                    'unlimited' => $quotaCheck['tokens']['unlimited'] ?? false,
                ],
                'resets_at' => now()->endOfMonth()->toIso8601String(),
            ],
            'upgrade_hint' => 'Upgrade your plan to increase MCP quota limits.',
        ];

        return response()->json($errorData, 429, $headers);
    }

    /**
     * Add quota headers to response.
     */
    protected function addQuotaHeaders(Response $response, $workspace): void
    {
        $headers = $this->quotaService->getQuotaHeaders($workspace);

        foreach ($headers as $name => $value) {
            $response->headers->set($name, $value);
        }
    }
}
