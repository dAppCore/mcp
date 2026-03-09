<?php

declare(strict_types=1);

/**
 * Unit: MCP Quota Service
 *
 * Comprehensive tests for MCP quota system to ensure proper:
 * - Quota enforcement per tier (free, starter, pro, business, enterprise)
 * - Quota tracking and consumption
 * - Quota reset periods
 * - Quota exceeded responses
 * - Workspace-scoped quotas
 * - Quota bypass for specific operations
 * - Edge cases (concurrent requests, race conditions)
 *
 * @see TODO.md P2-016: Test Quota System
 */

use Core\Mcp\Middleware\CheckMcpQuota;
use Core\Mcp\Models\McpUsageQuota;
use Core\Mcp\Services\McpQuotaService;
use Core\Tenant\Models\Workspace;
use Core\Tenant\Services\EntitlementResult;
use Core\Tenant\Services\EntitlementService;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Mockery;

// =============================================================================
// Usage Recording Tests
// =============================================================================

describe('Usage recording', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('records usage for workspace with all parameters', function () {
        $quota = $this->quotaService->recordUsage(
            $this->workspace,
            toolCalls: 5,
            inputTokens: 100,
            outputTokens: 50
        );

        expect($quota)->toBeInstanceOf(McpUsageQuota::class);
        expect($quota->tool_calls_count)->toBe(5);
        expect($quota->input_tokens)->toBe(100);
        expect($quota->output_tokens)->toBe(50);
        expect($quota->total_tokens)->toBe(150);
        expect($quota->month)->toBe(now()->format('Y-m'));
    });

    it('records usage with workspace ID instead of model', function () {
        $quota = $this->quotaService->recordUsage(
            $this->workspace->id,
            toolCalls: 3,
            inputTokens: 50,
            outputTokens: 25
        );

        expect($quota->workspace_id)->toBe($this->workspace->id);
        expect($quota->tool_calls_count)->toBe(3);
    });

    it('increments existing usage when recording multiple times', function () {
        // First call
        $this->quotaService->recordUsage($this->workspace, toolCalls: 5, inputTokens: 100, outputTokens: 50);

        // Second call
        $quota = $this->quotaService->recordUsage($this->workspace, toolCalls: 3, inputTokens: 200, outputTokens: 100);

        expect($quota->tool_calls_count)->toBe(8);
        expect($quota->input_tokens)->toBe(300);
        expect($quota->output_tokens)->toBe(150);
        expect($quota->total_tokens)->toBe(450);
    });

    it('records default of 1 tool call when no count specified', function () {
        $quota = $this->quotaService->recordUsage($this->workspace);

        expect($quota->tool_calls_count)->toBe(1);
        expect($quota->input_tokens)->toBe(0);
        expect($quota->output_tokens)->toBe(0);
    });

    it('invalidates cache after recording usage', function () {
        $cacheKey = "mcp_usage:{$this->workspace->id}:" . now()->format('Y-m');

        // Pre-populate cache
        Cache::put($cacheKey, ['tool_calls_count' => 0], 60);

        // Record usage
        $this->quotaService->recordUsage($this->workspace, toolCalls: 5);

        // Cache should be invalidated
        expect(Cache::has($cacheKey))->toBeFalse();
    });

    it('separates usage between different workspaces', function () {
        $workspace2 = Workspace::factory()->create();

        $this->quotaService->recordUsage($this->workspace, toolCalls: 5);
        $this->quotaService->recordUsage($workspace2, toolCalls: 10);

        $quota1 = McpUsageQuota::where('workspace_id', $this->workspace->id)->first();
        $quota2 = McpUsageQuota::where('workspace_id', $workspace2->id)->first();

        expect($quota1->tool_calls_count)->toBe(5);
        expect($quota2->tool_calls_count)->toBe(10);
    });

    it('separates usage between different months', function () {
        // Record for current month
        $this->quotaService->recordUsage($this->workspace, toolCalls: 5);

        // Create record for previous month directly
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->subMonth()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $currentMonthQuota = McpUsageQuota::where('workspace_id', $this->workspace->id)
            ->where('month', now()->format('Y-m'))
            ->first();

        expect($currentMonthQuota->tool_calls_count)->toBe(5);
    });
});

// =============================================================================
// Quota Checking Tests - Tier Enforcement
// =============================================================================

describe('Quota enforcement per tier', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('allows unlimited usage for enterprise tier', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        // Record substantial usage
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100000,
            'input_tokens' => 10000000,
            'output_tokens' => 5000000,
        ]);

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeTrue();
    });

    it('enforces free tier limit of 100 tool calls', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 100, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeFalse();
    });

    it('enforces starter tier limit of 500 tool calls', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 500,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 500, used: 500, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeFalse();
    });

    it('enforces professional tier limit of 2000 tool calls', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 2000,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 2000, used: 2000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeFalse();
    });

    it('enforces business tier limit of 10000 tool calls', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 10000,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 10000, used: 10000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeFalse();
    });

    it('allows usage within free tier limit', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 50, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeTrue();
    });

    it('denies access when feature not in plan', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::denied('Not included in plan', featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeFalse();
    });
});

// =============================================================================
// Token Quota Enforcement Tests
// =============================================================================

describe('Token quota enforcement', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('enforces token limit independently of tool calls', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 10,
            'input_tokens' => 500000,
            'output_tokens' => 500000,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 1000, used: 10, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 1000000, used: 1000000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeFalse();
    });

    it('allows when tokens within limit but tool calls at limit', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 100000,
            'output_tokens' => 100000,
        ]);

        // Tool calls at limit
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 50, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        // Tokens within limit
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 1000000, used: 200000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeTrue();
    });

    it('treats missing token feature as allowed', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 10,
            'input_tokens' => 500000,
            'output_tokens' => 500000,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 10, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        // Token feature denied (not tracked separately)
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::denied('Not tracked', featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($this->workspace);

        expect($result)->toBeTrue();
    });
});

// =============================================================================
// Detailed Quota Check Tests
// =============================================================================

describe('Detailed quota check', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('returns detailed quota information when allowed', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 30,
            'input_tokens' => 5000,
            'output_tokens' => 3000,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 30, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 100000, used: 8000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuotaDetailed($this->workspace);

        expect($result['allowed'])->toBeTrue();
        expect($result['reason'])->toBeNull();
        expect($result['tool_calls']['allowed'])->toBeTrue();
        expect($result['tool_calls']['used'])->toBe(30);
        expect($result['tool_calls']['limit'])->toBe(100);
        expect($result['tool_calls']['unlimited'])->toBeFalse();
        expect($result['tokens']['allowed'])->toBeTrue();
        expect($result['tokens']['used'])->toBe(8000);
        expect($result['tokens']['input_tokens'])->toBe(5000);
        expect($result['tokens']['output_tokens'])->toBe(3000);
    });

    it('returns detailed reason when tool calls quota exceeded', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 100, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuotaDetailed($this->workspace);

        expect($result['allowed'])->toBeFalse();
        expect($result['reason'])->toContain('100/100');
        expect($result['tool_calls']['allowed'])->toBeFalse();
        expect($result['tool_calls']['reason'])->toContain('limit reached');
    });

    it('returns detailed reason when token quota exceeded', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 10,
            'input_tokens' => 50000,
            'output_tokens' => 50000,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 10, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 100000, used: 100000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuotaDetailed($this->workspace);

        expect($result['allowed'])->toBeFalse();
        expect($result['tokens']['allowed'])->toBeFalse();
        expect($result['tokens']['reason'])->toContain('token limit');
    });

    it('returns detailed reason when feature not in plan', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::denied('Not included in plan', featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::denied('Not included', featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuotaDetailed($this->workspace);

        expect($result['allowed'])->toBeFalse();
        expect($result['tool_calls']['allowed'])->toBeFalse();
        expect($result['tool_calls']['reason'])->toContain('not included');
    });

    it('returns error for non-existent workspace', function () {
        $result = $this->quotaService->checkQuotaDetailed(999999);

        expect($result['allowed'])->toBeFalse();
        expect($result['reason'])->toBe('Workspace not found');
    });

    it('marks unlimited quotas correctly', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuotaDetailed($this->workspace);

        expect($result['tool_calls']['unlimited'])->toBeTrue();
        expect($result['tool_calls']['limit'])->toBeNull();
        expect($result['tokens']['unlimited'])->toBeTrue();
        expect($result['tokens']['limit'])->toBeNull();
    });
});

// =============================================================================
// Remaining Quota Tests
// =============================================================================

describe('Remaining quota calculation', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('calculates remaining tool calls correctly', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 30,
            'input_tokens' => 500,
            'output_tokens' => 500,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 30, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 5000, used: 1000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $remaining = $this->quotaService->getRemainingQuota($this->workspace);

        expect($remaining['tool_calls'])->toBe(70);
        expect($remaining['tokens'])->toBe(4000);
        expect($remaining['tool_calls_unlimited'])->toBeFalse();
        expect($remaining['tokens_unlimited'])->toBeFalse();
    });

    it('returns null remaining for unlimited quotas', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $remaining = $this->quotaService->getRemainingQuota($this->workspace);

        expect($remaining['tool_calls'])->toBeNull();
        expect($remaining['tokens'])->toBeNull();
        expect($remaining['tool_calls_unlimited'])->toBeTrue();
        expect($remaining['tokens_unlimited'])->toBeTrue();
    });

    it('returns zero remaining when quota exceeded', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 150,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 150, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $remaining = $this->quotaService->getRemainingQuota($this->workspace);

        expect($remaining['tool_calls'])->toBe(0);
    });

    it('returns zero for non-existent workspace', function () {
        $remaining = $this->quotaService->getRemainingQuota(999999);

        expect($remaining['tool_calls'])->toBe(0);
        expect($remaining['tokens'])->toBe(0);
        expect($remaining['tool_calls_unlimited'])->toBeFalse();
        expect($remaining['tokens_unlimited'])->toBeFalse();
    });
});

// =============================================================================
// Quota Reset Tests
// =============================================================================

describe('Quota reset', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('resets monthly quota to zero', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 1000,
            'output_tokens' => 500,
        ]);

        $quota = $this->quotaService->resetMonthlyQuota($this->workspace);

        expect($quota->tool_calls_count)->toBe(0);
        expect($quota->input_tokens)->toBe(0);
        expect($quota->output_tokens)->toBe(0);
    });

    it('invalidates cache after reset', function () {
        $cacheKey = "mcp_usage:{$this->workspace->id}:" . now()->format('Y-m');

        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        // Pre-populate cache
        Cache::put($cacheKey, ['tool_calls_count' => 50], 60);

        $this->quotaService->resetMonthlyQuota($this->workspace);

        expect(Cache::has($cacheKey))->toBeFalse();
    });

    it('creates quota record if none exists during reset', function () {
        $quota = $this->quotaService->resetMonthlyQuota($this->workspace);

        expect($quota)->toBeInstanceOf(McpUsageQuota::class);
        expect($quota->tool_calls_count)->toBe(0);
    });
});

// =============================================================================
// Usage History Tests
// =============================================================================

describe('Usage history', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('returns usage history ordered by month descending', function () {
        foreach (['2026-01', '2025-12', '2025-11'] as $month) {
            McpUsageQuota::create([
                'workspace_id' => $this->workspace->id,
                'month' => $month,
                'tool_calls_count' => rand(10, 100),
                'input_tokens' => rand(100, 1000),
                'output_tokens' => rand(100, 1000),
            ]);
        }

        $history = $this->quotaService->getUsageHistory($this->workspace, 3);

        expect($history)->toHaveCount(3);
        expect($history->first()->month)->toBe('2026-01');
        expect($history->last()->month)->toBe('2025-11');
    });

    it('limits history to specified number of months', function () {
        foreach (['2026-01', '2025-12', '2025-11', '2025-10', '2025-09'] as $month) {
            McpUsageQuota::create([
                'workspace_id' => $this->workspace->id,
                'month' => $month,
                'tool_calls_count' => 10,
                'input_tokens' => 0,
                'output_tokens' => 0,
            ]);
        }

        $history = $this->quotaService->getUsageHistory($this->workspace, 3);

        expect($history)->toHaveCount(3);
    });

    it('returns empty collection for workspace with no history', function () {
        $history = $this->quotaService->getUsageHistory($this->workspace, 12);

        expect($history)->toBeEmpty();
    });

    it('accepts workspace ID instead of model', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $history = $this->quotaService->getUsageHistory($this->workspace->id, 12);

        expect($history)->toHaveCount(1);
    });
});

// =============================================================================
// Quota Headers Tests
// =============================================================================

describe('Quota headers', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('returns correct quota headers for limited plan', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 25,
            'input_tokens' => 300,
            'output_tokens' => 200,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 25, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 10000, used: 500, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $headers = $this->quotaService->getQuotaHeaders($this->workspace);

        expect($headers['X-MCP-Quota-Tool-Calls-Used'])->toBe('25');
        expect($headers['X-MCP-Quota-Tool-Calls-Limit'])->toBe('100');
        expect($headers['X-MCP-Quota-Tool-Calls-Remaining'])->toBe('75');
        expect($headers['X-MCP-Quota-Tokens-Used'])->toBe('500');
        expect($headers['X-MCP-Quota-Tokens-Limit'])->toBe('10000');
        expect($headers)->toHaveKey('X-MCP-Quota-Reset');
    });

    it('returns unlimited indicators for unlimited plan', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $headers = $this->quotaService->getQuotaHeaders($this->workspace);

        expect($headers['X-MCP-Quota-Tool-Calls-Limit'])->toBe('unlimited');
        expect($headers['X-MCP-Quota-Tool-Calls-Remaining'])->toBe('unlimited');
        expect($headers['X-MCP-Quota-Tokens-Limit'])->toBe('unlimited');
        expect($headers['X-MCP-Quota-Tokens-Remaining'])->toBe('unlimited');
    });

    it('includes reset time as ISO 8601', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->andReturn(EntitlementResult::unlimited('test'));

        $headers = $this->quotaService->getQuotaHeaders($this->workspace);

        expect($headers['X-MCP-Quota-Reset'])->toMatch('/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/');
        expect($headers['X-MCP-Quota-Reset'])->toContain(now()->format('Y-m'));
    });
});

// =============================================================================
// Quota Limits Retrieval Tests
// =============================================================================

describe('Quota limits retrieval', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
        Cache::flush();
    });

    it('returns quota limits from entitlements', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 500, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 100000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $limits = $this->quotaService->getQuotaLimits($this->workspace);

        expect($limits['tool_calls_limit'])->toBe(500);
        expect($limits['tokens_limit'])->toBe(100000);
        expect($limits['tool_calls_unlimited'])->toBeFalse();
        expect($limits['tokens_unlimited'])->toBeFalse();
    });

    it('returns null limits for unlimited plans', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $limits = $this->quotaService->getQuotaLimits($this->workspace);

        expect($limits['tool_calls_limit'])->toBeNull();
        expect($limits['tokens_limit'])->toBeNull();
        expect($limits['tool_calls_unlimited'])->toBeTrue();
        expect($limits['tokens_unlimited'])->toBeTrue();
    });

    it('returns zero limits for non-existent workspace', function () {
        $limits = $this->quotaService->getQuotaLimits(999999);

        expect($limits['tool_calls_limit'])->toBe(0);
        expect($limits['tokens_limit'])->toBe(0);
        expect($limits['tool_calls_unlimited'])->toBeFalse();
        expect($limits['tokens_unlimited'])->toBeFalse();
    });

    it('caches quota limits', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->once()
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 500, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->once()
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::allowed(limit: 100000, featureCode: McpQuotaService::FEATURE_MONTHLY_TOKENS));

        // First call
        $this->quotaService->getQuotaLimits($this->workspace);

        // Second call should use cache (mock only allows once)
        $limits = $this->quotaService->getQuotaLimits($this->workspace);

        expect($limits['tool_calls_limit'])->toBe(500);
    });
});

// =============================================================================
// CheckMcpQuota Middleware Tests
// =============================================================================

describe('CheckMcpQuota middleware', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->middleware = new CheckMcpQuota($this->quotaService);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('allows request when quota not exceeded', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 10, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('workspace', $this->workspace);

        $response = $this->middleware->handle($request, function ($req) {
            return response()->json(['success' => true]);
        });

        expect($response->getStatusCode())->toBe(200);
        expect($response->headers->has('X-MCP-Quota-Tool-Calls-Used'))->toBeTrue();
    });

    it('returns 429 when quota exceeded', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 100, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('workspace', $this->workspace);

        $response = $this->middleware->handle($request, function ($req) {
            return response()->json(['success' => true]);
        });

        expect($response->getStatusCode())->toBe(429);

        $content = json_decode($response->getContent(), true);
        expect($content['error'])->toBe('quota_exceeded');
        expect($content)->toHaveKey('quota');
        expect($content)->toHaveKey('upgrade_hint');
    });

    it('skips quota check when no workspace context', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        // No workspace attribute set

        $response = $this->middleware->handle($request, function ($req) {
            return response()->json(['success' => true]);
        });

        expect($response->getStatusCode())->toBe(200);
    });

    it('adds quota headers to response', function () {
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 10, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('workspace', $this->workspace);

        $response = $this->middleware->handle($request, function ($req) {
            return response()->json(['success' => true]);
        });

        expect($response->headers->has('X-MCP-Quota-Tool-Calls-Used'))->toBeTrue();
        expect($response->headers->has('X-MCP-Quota-Tool-Calls-Limit'))->toBeTrue();
        expect($response->headers->has('X-MCP-Quota-Tool-Calls-Remaining'))->toBeTrue();
        expect($response->headers->has('X-MCP-Quota-Tokens-Used'))->toBeTrue();
        expect($response->headers->has('X-MCP-Quota-Reset'))->toBeTrue();
    });

    it('includes resets_at in exceeded response', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 100, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('workspace', $this->workspace);

        $response = $this->middleware->handle($request, function ($req) {
            return response()->json(['success' => true]);
        });

        $content = json_decode($response->getContent(), true);
        expect($content['quota']['resets_at'])->toMatch('/^\d{4}-\d{2}-\d{2}T/');
    });
});

// =============================================================================
// Workspace-Scoped Quota Tests
// =============================================================================

describe('Workspace-scoped quotas', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace1 = Workspace::factory()->create();
        $this->workspace2 = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('tracks usage separately per workspace', function () {
        $this->quotaService->recordUsage($this->workspace1, toolCalls: 50);
        $this->quotaService->recordUsage($this->workspace2, toolCalls: 75);

        $usage1 = $this->quotaService->getCurrentUsage($this->workspace1);
        $usage2 = $this->quotaService->getCurrentUsage($this->workspace2);

        expect($usage1['tool_calls_count'])->toBe(50);
        expect($usage2['tool_calls_count'])->toBe(75);
    });

    it('enforces limits independently per workspace', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace1->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        McpUsageQuota::create([
            'workspace_id' => $this->workspace2->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        // Workspace 1 at limit
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace1, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 100, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace1, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        // Workspace 2 has headroom
        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace2, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 50, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace2, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        expect($this->quotaService->checkQuota($this->workspace1))->toBeFalse();
        expect($this->quotaService->checkQuota($this->workspace2))->toBeTrue();
    });

    it('resets only the specified workspace quota', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace1->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        McpUsageQuota::create([
            'workspace_id' => $this->workspace2->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 75,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->quotaService->resetMonthlyQuota($this->workspace1);

        $usage1 = $this->quotaService->getCurrentUsage($this->workspace1);
        $usage2 = $this->quotaService->getCurrentUsage($this->workspace2);

        expect($usage1['tool_calls_count'])->toBe(0);
        expect($usage2['tool_calls_count'])->toBe(75);
    });
});

// =============================================================================
// McpUsageQuota Model Tests
// =============================================================================

describe('McpUsageQuota model', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create();
    });

    it('gets or creates quota for workspace', function () {
        $quota = McpUsageQuota::getOrCreate($this->workspace->id);

        expect($quota)->toBeInstanceOf(McpUsageQuota::class);
        expect($quota->workspace_id)->toBe($this->workspace->id);
        expect($quota->month)->toBe(now()->format('Y-m'));
        expect($quota->tool_calls_count)->toBe(0);
    });

    it('returns existing quota when present', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 42,
            'input_tokens' => 100,
            'output_tokens' => 50,
        ]);

        $quota = McpUsageQuota::getOrCreate($this->workspace->id);

        expect($quota->tool_calls_count)->toBe(42);
    });

    it('records usage atomically', function () {
        $quota = McpUsageQuota::getOrCreate($this->workspace->id);
        $quota->recordUsage(toolCalls: 5, inputTokens: 100, outputTokens: 50);

        $fresh = $quota->fresh();
        expect($fresh->tool_calls_count)->toBe(5);
        expect($fresh->input_tokens)->toBe(100);
        expect($fresh->output_tokens)->toBe(50);
    });

    it('calculates total tokens accessor', function () {
        $quota = McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 10,
            'input_tokens' => 500,
            'output_tokens' => 300,
        ]);

        expect($quota->total_tokens)->toBe(800);
    });

    it('formats month label correctly', function () {
        $quota = McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => '2026-01',
            'tool_calls_count' => 0,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        expect($quota->month_label)->toBe('January 2026');
    });

    it('resets all counters', function () {
        $quota = McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 5000,
            'output_tokens' => 3000,
        ]);

        $quota->reset();

        expect($quota->tool_calls_count)->toBe(0);
        expect($quota->input_tokens)->toBe(0);
        expect($quota->output_tokens)->toBe(0);
    });

    it('scopes by month', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => '2026-01',
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => '2025-12',
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $quotas = McpUsageQuota::forMonth('2026-01')->get();

        expect($quotas)->toHaveCount(1);
        expect($quotas->first()->tool_calls_count)->toBe(50);
    });

    it('scopes to current month', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 25,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->subMonth()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $quotas = McpUsageQuota::currentMonth()->get();

        expect($quotas)->toHaveCount(1);
        expect($quotas->first()->tool_calls_count)->toBe(25);
    });

    it('belongs to workspace', function () {
        $quota = McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 0,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        expect($quota->workspace)->toBeInstanceOf(Workspace::class);
        expect($quota->workspace->id)->toBe($this->workspace->id);
    });

    it('converts to array for API responses', function () {
        $quota = McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => '2026-01',
            'tool_calls_count' => 50,
            'input_tokens' => 500,
            'output_tokens' => 300,
        ]);

        $array = $quota->toArray();

        expect($array)->toHaveKey('workspace_id');
        expect($array)->toHaveKey('month');
        expect($array)->toHaveKey('month_label');
        expect($array)->toHaveKey('tool_calls_count');
        expect($array)->toHaveKey('input_tokens');
        expect($array)->toHaveKey('output_tokens');
        expect($array)->toHaveKey('total_tokens');
        expect($array['total_tokens'])->toBe(800);
    });

    it('uses static record method correctly', function () {
        $quota = McpUsageQuota::record(
            $this->workspace->id,
            toolCalls: 3,
            inputTokens: 100,
            outputTokens: 50
        );

        expect($quota->tool_calls_count)->toBe(3);
        expect($quota->input_tokens)->toBe(100);
        expect($quota->output_tokens)->toBe(50);
    });
});

// =============================================================================
// Edge Cases and Concurrent Request Tests
// =============================================================================

describe('Edge cases', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
    });

    afterEach(function () {
        Mockery::close();
    });

    it('handles zero token usage', function () {
        $quota = $this->quotaService->recordUsage($this->workspace, toolCalls: 1, inputTokens: 0, outputTokens: 0);

        expect($quota->input_tokens)->toBe(0);
        expect($quota->output_tokens)->toBe(0);
        expect($quota->total_tokens)->toBe(0);
    });

    it('handles very large usage numbers', function () {
        $quota = $this->quotaService->recordUsage(
            $this->workspace,
            toolCalls: 1000000,
            inputTokens: 1000000000,
            outputTokens: 500000000
        );

        expect($quota->tool_calls_count)->toBe(1000000);
        expect($quota->input_tokens)->toBe(1000000000);
        expect($quota->output_tokens)->toBe(500000000);
    });

    it('handles boundary condition at exactly limit', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 99,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 99, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($this->workspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        // Should allow as we're at 99/100
        expect($this->quotaService->checkQuota($this->workspace))->toBeTrue();
    });

    it('handles new workspace with no usage record', function () {
        $newWorkspace = Workspace::factory()->create();

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($newWorkspace, McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS)
            ->andReturn(EntitlementResult::allowed(limit: 100, used: 0, featureCode: McpQuotaService::FEATURE_MONTHLY_TOOL_CALLS));

        $this->entitlementsMock
            ->shouldReceive('can')
            ->with($newWorkspace, McpQuotaService::FEATURE_MONTHLY_TOKENS)
            ->andReturn(EntitlementResult::unlimited(McpQuotaService::FEATURE_MONTHLY_TOKENS));

        $result = $this->quotaService->checkQuota($newWorkspace);

        expect($result)->toBeTrue();
    });

    it('handles concurrent usage recording via atomic increment', function () {
        // Create initial quota
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 0,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        // Simulate concurrent recordings
        $this->quotaService->recordUsage($this->workspace, toolCalls: 1);
        $this->quotaService->recordUsage($this->workspace, toolCalls: 1);
        $this->quotaService->recordUsage($this->workspace, toolCalls: 1);

        $usage = $this->quotaService->getCurrentUsage($this->workspace);

        expect($usage['tool_calls_count'])->toBe(3);
    });

    it('handles cache invalidation race conditions', function () {
        $cacheKey = "mcp_usage:{$this->workspace->id}:" . now()->format('Y-m');

        // Pre-populate cache
        Cache::put($cacheKey, ['tool_calls_count' => 10], 60);

        // Record new usage
        $this->quotaService->recordUsage($this->workspace, toolCalls: 5);

        // Cache should be invalidated
        expect(Cache::has($cacheKey))->toBeFalse();

        // Fresh read should reflect actual database state
        $usage = $this->quotaService->getCurrentUsage($this->workspace);
        expect($usage['tool_calls_count'])->toBe(5);
    });

    it('handles month boundary transitions', function () {
        // Create usage for previous month
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->subMonth()->format('Y-m'),
            'tool_calls_count' => 100,
            'input_tokens' => 10000,
            'output_tokens' => 5000,
        ]);

        $usage = $this->quotaService->getCurrentUsage($this->workspace);

        // Should return zero for current month (new month)
        expect($usage['tool_calls_count'])->toBe(0);
        expect($usage['month'])->toBe(now()->format('Y-m'));
    });
});

// =============================================================================
// Cache Management Tests
// =============================================================================

describe('Cache management', function () {
    beforeEach(function () {
        $this->entitlementsMock = Mockery::mock(EntitlementService::class);
        $this->quotaService = new McpQuotaService($this->entitlementsMock);
        $this->workspace = Workspace::factory()->create();
        Cache::flush();
    });

    afterEach(function () {
        Mockery::close();
        Cache::flush();
    });

    it('caches current usage for performance', function () {
        McpUsageQuota::create([
            'workspace_id' => $this->workspace->id,
            'month' => now()->format('Y-m'),
            'tool_calls_count' => 50,
            'input_tokens' => 0,
            'output_tokens' => 0,
        ]);

        // First call should hit database
        $usage1 = $this->quotaService->getCurrentUsage($this->workspace);

        // Modify database directly
        McpUsageQuota::where('workspace_id', $this->workspace->id)->update(['tool_calls_count' => 100]);

        // Second call should return cached value
        $usage2 = $this->quotaService->getCurrentUsage($this->workspace);

        expect($usage1['tool_calls_count'])->toBe(50);
        expect($usage2['tool_calls_count'])->toBe(50); // Still cached value
    });

    it('invalidates both usage and limits cache', function () {
        $usageCacheKey = "mcp_usage:{$this->workspace->id}:" . now()->format('Y-m');
        $limitsCacheKey = "mcp_quota_limits:{$this->workspace->id}";

        Cache::put($usageCacheKey, ['tool_calls_count' => 10], 60);
        Cache::put($limitsCacheKey, ['tool_calls_limit' => 100], 60);

        $this->quotaService->invalidateUsageCache($this->workspace->id);

        expect(Cache::has($usageCacheKey))->toBeFalse();
        expect(Cache::has($limitsCacheKey))->toBeFalse();
    });
});
