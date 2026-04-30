<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Models\McpUsageQuota;
use Core\Tenant\Models\Workspace;
use Core\Tenant\Services\EntitlementService;
use Illuminate\Support\Facades\Cache;

/**
 * MCP Quota Service - manages workspace-level usage quotas for MCP.
 *
 * Provides quota checking, usage recording, and limit enforcement
 * for tool calls and token consumption.
 */
class McpQuotaService
{
    /**
     * Feature codes for MCP quota limits in the entitlement system.
     */
    public const FEATURE_MONTHLY_TOOL_CALLS = 'mcp.monthly_tool_calls';

    public const FEATURE_MONTHLY_TOKENS = 'mcp.monthly_tokens';

    /**
     * Cache TTL for quota limits (5 minutes).
     */
    protected const CACHE_TTL = 300;

    public function __construct(
        protected EntitlementService $entitlements
    ) {}

    // ─────────────────────────────────────────────────────────────────────────
    // Usage Recording
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Record MCP usage for a workspace.
     */
    public function recordUsage(
        Workspace|int $workspace,
        int $toolCalls = 1,
        int $inputTokens = 0,
        int $outputTokens = 0
    ): McpUsageQuota {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;

        $quota = McpUsageQuota::record($workspaceId, $toolCalls, $inputTokens, $outputTokens);

        // Invalidate cached usage
        $this->invalidateUsageCache($workspaceId);

        return $quota;
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Quota Checking
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Check if workspace is within quota limits.
     *
     * Returns true if within limits (or unlimited), false if quota exceeded.
     */
    public function checkQuota(Workspace|int $workspace): bool
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;
        $workspace = $workspace instanceof Workspace ? $workspace : Workspace::find($workspaceId);

        if (! $workspace) {
            return false;
        }

        // Check tool calls quota
        $toolCallsResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOOL_CALLS);

        if ($toolCallsResult->isDenied()) {
            // Feature not in plan - deny access
            return false;
        }

        if (! $toolCallsResult->isUnlimited()) {
            $usage = $this->getCurrentUsage($workspace);
            $limit = $toolCallsResult->limit;

            if ($limit !== null && $usage['tool_calls_count'] >= $limit) {
                return false;
            }
        }

        // Check tokens quota
        $tokensResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOKENS);

        if (! $tokensResult->isUnlimited() && $tokensResult->isAllowed()) {
            $usage = $this->getCurrentUsage($workspace);
            $limit = $tokensResult->limit;

            if ($limit !== null && $usage['total_tokens'] >= $limit) {
                return false;
            }
        }

        return true;
    }

    /**
     * Get detailed quota check result with reasons.
     *
     * @return array{allowed: bool, reason: ?string, tool_calls: array, tokens: array}
     */
    public function checkQuotaDetailed(Workspace|int $workspace): array
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;
        $workspace = $workspace instanceof Workspace ? $workspace : Workspace::find($workspaceId);

        if (! $workspace) {
            return [
                'allowed' => false,
                'reason' => 'Workspace not found',
                'tool_calls' => ['allowed' => false],
                'tokens' => ['allowed' => false],
            ];
        }

        $usage = $this->getCurrentUsage($workspace);

        // Check tool calls
        $toolCallsResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOOL_CALLS);
        $toolCallsAllowed = true;
        $toolCallsReason = null;

        if ($toolCallsResult->isDenied()) {
            $toolCallsAllowed = false;
            $toolCallsReason = 'MCP tool calls not included in your plan';
        } elseif (! $toolCallsResult->isUnlimited()) {
            $limit = $toolCallsResult->limit;
            if ($limit !== null && $usage['tool_calls_count'] >= $limit) {
                $toolCallsAllowed = false;
                $toolCallsReason = "Monthly tool calls limit reached ({$usage['tool_calls_count']}/{$limit})";
            }
        }

        // Check tokens
        $tokensResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOKENS);
        $tokensAllowed = true;
        $tokensReason = null;

        if ($tokensResult->isDenied()) {
            // Tokens might not be tracked separately - this is OK
            $tokensAllowed = true;
        } elseif (! $tokensResult->isUnlimited() && $tokensResult->isAllowed()) {
            $limit = $tokensResult->limit;
            if ($limit !== null && $usage['total_tokens'] >= $limit) {
                $tokensAllowed = false;
                $tokensReason = "Monthly token limit reached ({$usage['total_tokens']}/{$limit})";
            }
        }

        $allowed = $toolCallsAllowed && $tokensAllowed;
        $reason = $toolCallsReason ?? $tokensReason;

        return [
            'allowed' => $allowed,
            'reason' => $reason,
            'tool_calls' => [
                'allowed' => $toolCallsAllowed,
                'reason' => $toolCallsReason,
                'used' => $usage['tool_calls_count'],
                'limit' => $toolCallsResult->isUnlimited() ? null : $toolCallsResult->limit,
                'unlimited' => $toolCallsResult->isUnlimited(),
            ],
            'tokens' => [
                'allowed' => $tokensAllowed,
                'reason' => $tokensReason,
                'used' => $usage['total_tokens'],
                'input_tokens' => $usage['input_tokens'],
                'output_tokens' => $usage['output_tokens'],
                'limit' => $tokensResult->isUnlimited() ? null : $tokensResult->limit,
                'unlimited' => $tokensResult->isUnlimited(),
            ],
        ];
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Usage Retrieval
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Get current month's usage for a workspace.
     *
     * @return array{tool_calls_count: int, input_tokens: int, output_tokens: int, total_tokens: int, month: string}
     */
    public function getCurrentUsage(Workspace|int $workspace): array
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;

        return Cache::remember(
            $this->getUsageCacheKey($workspaceId),
            60, // 1 minute cache for current usage
            function () use ($workspaceId) {
                $quota = McpUsageQuota::getCurrentForWorkspace($workspaceId);

                return [
                    'tool_calls_count' => $quota->tool_calls_count,
                    'input_tokens' => $quota->input_tokens,
                    'output_tokens' => $quota->output_tokens,
                    'total_tokens' => $quota->total_tokens,
                    'month' => $quota->month,
                ];
            }
        );
    }

    /**
     * Get remaining quota for a workspace.
     *
     * @return array{tool_calls: int|null, tokens: int|null, tool_calls_unlimited: bool, tokens_unlimited: bool}
     */
    public function getRemainingQuota(Workspace|int $workspace): array
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;
        $workspace = $workspace instanceof Workspace ? $workspace : Workspace::find($workspaceId);

        if (! $workspace) {
            return [
                'tool_calls' => 0,
                'tokens' => 0,
                'tool_calls_unlimited' => false,
                'tokens_unlimited' => false,
            ];
        }

        $usage = $this->getCurrentUsage($workspace);

        // Tool calls remaining
        $toolCallsResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOOL_CALLS);
        $toolCallsRemaining = null;
        $toolCallsUnlimited = $toolCallsResult->isUnlimited();

        if ($toolCallsResult->isAllowed() && ! $toolCallsUnlimited && $toolCallsResult->limit !== null) {
            $toolCallsRemaining = max(0, $toolCallsResult->limit - $usage['tool_calls_count']);
        }

        // Tokens remaining
        $tokensResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOKENS);
        $tokensRemaining = null;
        $tokensUnlimited = $tokensResult->isUnlimited();

        if ($tokensResult->isAllowed() && ! $tokensUnlimited && $tokensResult->limit !== null) {
            $tokensRemaining = max(0, $tokensResult->limit - $usage['total_tokens']);
        }

        return [
            'tool_calls' => $toolCallsRemaining,
            'tokens' => $tokensRemaining,
            'tool_calls_unlimited' => $toolCallsUnlimited,
            'tokens_unlimited' => $tokensUnlimited,
        ];
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Quota Management
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Reset monthly quota for a workspace (for billing cycle reset).
     */
    public function resetMonthlyQuota(Workspace|int $workspace): McpUsageQuota
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;

        $quota = McpUsageQuota::getCurrentForWorkspace($workspaceId);
        $quota->reset();

        $this->invalidateUsageCache($workspaceId);

        return $quota;
    }

    /**
     * Get usage history for a workspace (last N months).
     *
     * @return \Illuminate\Support\Collection<McpUsageQuota>
     */
    public function getUsageHistory(Workspace|int $workspace, int $months = 12): \Illuminate\Support\Collection
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;

        return McpUsageQuota::where('workspace_id', $workspaceId)
            ->orderByDesc('month')
            ->limit($months)
            ->get();
    }

    /**
     * Get quota limits from entitlements.
     *
     * @return array{tool_calls_limit: int|null, tokens_limit: int|null, tool_calls_unlimited: bool, tokens_unlimited: bool}
     */
    public function getQuotaLimits(Workspace|int $workspace): array
    {
        $workspaceId = $workspace instanceof Workspace ? $workspace->id : $workspace;
        $workspace = $workspace instanceof Workspace ? $workspace : Workspace::find($workspaceId);

        if (! $workspace) {
            return [
                'tool_calls_limit' => 0,
                'tokens_limit' => 0,
                'tool_calls_unlimited' => false,
                'tokens_unlimited' => false,
            ];
        }

        $cacheKey = "mcp_quota_limits:{$workspaceId}";

        return Cache::remember($cacheKey, self::CACHE_TTL, function () use ($workspace) {
            $toolCallsResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOOL_CALLS);
            $tokensResult = $this->entitlements->can($workspace, self::FEATURE_MONTHLY_TOKENS);

            return [
                'tool_calls_limit' => $toolCallsResult->isUnlimited() ? null : $toolCallsResult->limit,
                'tokens_limit' => $tokensResult->isUnlimited() ? null : $tokensResult->limit,
                'tool_calls_unlimited' => $toolCallsResult->isUnlimited(),
                'tokens_unlimited' => $tokensResult->isUnlimited(),
            ];
        });
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Response Headers
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Get quota info formatted for HTTP response headers.
     *
     * @return array<string, string>
     */
    public function getQuotaHeaders(Workspace|int $workspace): array
    {
        $usage = $this->getCurrentUsage($workspace);
        $remaining = $this->getRemainingQuota($workspace);
        $limits = $this->getQuotaLimits($workspace);

        $headers = [
            'X-MCP-Quota-Tool-Calls-Used' => (string) $usage['tool_calls_count'],
            'X-MCP-Quota-Tokens-Used' => (string) $usage['total_tokens'],
        ];

        if ($limits['tool_calls_unlimited']) {
            $headers['X-MCP-Quota-Tool-Calls-Limit'] = 'unlimited';
            $headers['X-MCP-Quota-Tool-Calls-Remaining'] = 'unlimited';
        } else {
            $headers['X-MCP-Quota-Tool-Calls-Limit'] = (string) ($limits['tool_calls_limit'] ?? 0);
            $headers['X-MCP-Quota-Tool-Calls-Remaining'] = (string) ($remaining['tool_calls'] ?? 0);
        }

        if ($limits['tokens_unlimited']) {
            $headers['X-MCP-Quota-Tokens-Limit'] = 'unlimited';
            $headers['X-MCP-Quota-Tokens-Remaining'] = 'unlimited';
        } else {
            $headers['X-MCP-Quota-Tokens-Limit'] = (string) ($limits['tokens_limit'] ?? 0);
            $headers['X-MCP-Quota-Tokens-Remaining'] = (string) ($remaining['tokens'] ?? 0);
        }

        $headers['X-MCP-Quota-Reset'] = now()->endOfMonth()->toIso8601String();

        return $headers;
    }

    // ─────────────────────────────────────────────────────────────────────────
    // Cache Management
    // ─────────────────────────────────────────────────────────────────────────

    /**
     * Invalidate usage cache for a workspace.
     */
    public function invalidateUsageCache(int $workspaceId): void
    {
        Cache::forget($this->getUsageCacheKey($workspaceId));
        Cache::forget("mcp_quota_limits:{$workspaceId}");
    }

    /**
     * Get cache key for workspace usage.
     */
    protected function getUsageCacheKey(int $workspaceId): string
    {
        $month = now()->format('Y-m');

        return "mcp_usage:{$workspaceId}:{$month}";
    }
}
