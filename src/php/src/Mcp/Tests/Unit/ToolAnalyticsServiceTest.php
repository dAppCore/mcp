<?php

declare(strict_types=1);

/**
 * Unit: Tool Analytics Service
 *
 * Comprehensive tests for MCP tool analytics to ensure proper:
 * - Usage tracking and recording
 * - Metrics collection and aggregation
 * - Error rate calculations
 * - Daily trend aggregation
 * - Reporting functions (popular tools, error-prone tools, workspace stats)
 * - Tool combination tracking
 *
 * @see TODO.md P2-015: Test Tool Analytics
 */

use Core\Mcp\DTO\ToolStats;
use Core\Mcp\Models\ToolMetric;
use Core\Mcp\Services\ToolAnalyticsService;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\DB;

// =============================================================================
// Usage Recording Tests
// =============================================================================

describe('Usage tracking', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
        Config::set('mcp.analytics.batch_size', 100);
    });

    it('records a single successful execution', function () {
        $this->analyticsService->recordExecution(
            tool: 'query_database',
            durationMs: 150,
            success: true,
            workspaceId: 'ws-123'
        );
        $this->analyticsService->flush();

        $metric = ToolMetric::forTool('query_database')
            ->forWorkspace('ws-123')
            ->today()
            ->first();

        expect($metric)->not->toBeNull();
        expect($metric->call_count)->toBe(1);
        expect($metric->error_count)->toBe(0);
        expect($metric->total_duration_ms)->toBe(150);
        expect($metric->min_duration_ms)->toBe(150);
        expect($metric->max_duration_ms)->toBe(150);
    });

    it('records a failed execution', function () {
        $this->analyticsService->recordExecution(
            tool: 'query_database',
            durationMs: 50,
            success: false,
            workspaceId: 'ws-123'
        );
        $this->analyticsService->flush();

        $metric = ToolMetric::forTool('query_database')
            ->forWorkspace('ws-123')
            ->today()
            ->first();

        expect($metric)->not->toBeNull();
        expect($metric->call_count)->toBe(1);
        expect($metric->error_count)->toBe(1);
        expect($metric->error_rate)->toBe(100.0);
    });

    it('aggregates multiple executions', function () {
        $this->analyticsService->recordExecution('query_database', 100, true, 'ws-123');
        $this->analyticsService->recordExecution('query_database', 200, true, 'ws-123');
        $this->analyticsService->recordExecution('query_database', 150, false, 'ws-123');
        $this->analyticsService->flush();

        $metric = ToolMetric::forTool('query_database')
            ->forWorkspace('ws-123')
            ->today()
            ->first();

        expect($metric)->not->toBeNull();
        expect($metric->call_count)->toBe(3);
        expect($metric->error_count)->toBe(1);
        expect($metric->total_duration_ms)->toBe(450);
        expect($metric->min_duration_ms)->toBe(100);
        expect($metric->max_duration_ms)->toBe(200);
    });

    it('records execution without workspace', function () {
        $this->analyticsService->recordExecution(
            tool: 'system_info',
            durationMs: 25,
            success: true,
            workspaceId: null
        );
        $this->analyticsService->flush();

        $metric = ToolMetric::forTool('system_info')
            ->forWorkspace(null)
            ->today()
            ->first();

        expect($metric)->not->toBeNull();
        expect($metric->call_count)->toBe(1);
    });

    it('does not record when analytics disabled', function () {
        Config::set('mcp.analytics.enabled', false);

        $this->analyticsService->recordExecution('query_database', 150, true, 'ws-123');
        $this->analyticsService->flush();

        expect(ToolMetric::count())->toBe(0);
    });

    it('auto-flushes when batch size reached', function () {
        Config::set('mcp.analytics.batch_size', 5);
        $service = new ToolAnalyticsService();

        for ($i = 0; $i < 5; $i++) {
            $service->recordExecution('query_database', 100, true, 'ws-123');
        }

        $metric = ToolMetric::forTool('query_database')->first();

        expect($metric)->not->toBeNull();
        expect($metric->call_count)->toBe(5);
    });
});

// =============================================================================
// ToolStats DTO Tests
// =============================================================================

describe('ToolStats DTO', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
    });

    it('returns DTO with correct values from getToolStats', function () {
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 100,
            'error_count' => 5,
            'total_duration_ms' => 15000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 500,
        ]);

        $stats = $this->analyticsService->getToolStats('query_database');

        expect($stats)->toBeInstanceOf(ToolStats::class);
        expect($stats->toolName)->toBe('query_database');
        expect($stats->totalCalls)->toBe(100);
        expect($stats->errorCount)->toBe(5);
        expect($stats->errorRate)->toBe(5.0);
        expect($stats->avgDurationMs)->toBe(150.0);
        expect($stats->minDurationMs)->toBe(50);
        expect($stats->maxDurationMs)->toBe(500);
    });

    it('aggregates stats across dates', function () {
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->subDays(2)->toDateString(),
            'call_count' => 50,
            'error_count' => 2,
            'total_duration_ms' => 5000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 200,
        ]);

        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->subDays(1)->toDateString(),
            'call_count' => 100,
            'error_count' => 8,
            'total_duration_ms' => 10000,
            'min_duration_ms' => 30,
            'max_duration_ms' => 400,
        ]);

        $stats = $this->analyticsService->getToolStats('query_database');

        expect($stats->totalCalls)->toBe(150);
        expect($stats->errorCount)->toBe(10);
        expect($stats->minDurationMs)->toBe(30);
        expect($stats->maxDurationMs)->toBe(400);
    });

    it('returns empty DTO for unknown tool', function () {
        $stats = $this->analyticsService->getToolStats('nonexistent_tool');

        expect($stats)->toBeInstanceOf(ToolStats::class);
        expect($stats->toolName)->toBe('nonexistent_tool');
        expect($stats->totalCalls)->toBe(0);
        expect($stats->errorCount)->toBe(0);
        expect($stats->errorRate)->toBe(0.0);
    });

    it('handles snake_case in fromArray', function () {
        $stats = ToolStats::fromArray([
            'tool_name' => 'test_tool',
            'total_calls' => 50,
            'error_count' => 5,
            'error_rate' => 10.0,
            'avg_duration_ms' => 100.0,
            'min_duration_ms' => 25,
            'max_duration_ms' => 300,
        ]);

        expect($stats->toolName)->toBe('test_tool');
        expect($stats->totalCalls)->toBe(50);
        expect($stats->errorCount)->toBe(5);
    });

    it('handles camelCase in fromArray', function () {
        $stats = ToolStats::fromArray([
            'toolName' => 'test_tool',
            'totalCalls' => 50,
            'errorCount' => 5,
            'errorRate' => 10.0,
            'avgDurationMs' => 100.0,
            'minDurationMs' => 25,
            'maxDurationMs' => 300,
        ]);

        expect($stats->toolName)->toBe('test_tool');
        expect($stats->totalCalls)->toBe(50);
    });

    it('calculates success rate correctly', function () {
        $stats = new ToolStats(
            toolName: 'test_tool',
            totalCalls: 100,
            errorCount: 15,
            errorRate: 15.0,
            avgDurationMs: 100.0,
            minDurationMs: 50,
            maxDurationMs: 200,
        );

        expect($stats->getSuccessRate())->toBe(85.0);
    });

    it('formats duration for humans in milliseconds', function () {
        $stats = new ToolStats(
            toolName: 'fast_tool',
            totalCalls: 10,
            errorCount: 0,
            errorRate: 0.0,
            avgDurationMs: 250.0,
            minDurationMs: 100,
            maxDurationMs: 500,
        );

        expect($stats->getAvgDurationForHumans())->toBe('250ms');
    });

    it('formats duration for humans in seconds', function () {
        $stats = new ToolStats(
            toolName: 'slow_tool',
            totalCalls: 10,
            errorCount: 0,
            errorRate: 0.0,
            avgDurationMs: 2500.0,
            minDurationMs: 1000,
            maxDurationMs: 5000,
        );

        expect($stats->getAvgDurationForHumans())->toBe('2.5s');
    });

    it('formats zero duration as dash', function () {
        $stats = new ToolStats(
            toolName: 'unused_tool',
            totalCalls: 0,
            errorCount: 0,
            errorRate: 0.0,
            avgDurationMs: 0.0,
            minDurationMs: 0,
            maxDurationMs: 0,
        );

        expect($stats->getAvgDurationForHumans())->toBe('-');
    });

    it('detects high error rate', function () {
        $stats = new ToolStats(
            toolName: 'flaky_tool',
            totalCalls: 100,
            errorCount: 15,
            errorRate: 15.0,
            avgDurationMs: 100.0,
            minDurationMs: 50,
            maxDurationMs: 200,
        );

        expect($stats->hasHighErrorRate(10.0))->toBeTrue();
        expect($stats->hasHighErrorRate(20.0))->toBeFalse();
    });

    it('detects slow response', function () {
        $stats = new ToolStats(
            toolName: 'slow_tool',
            totalCalls: 10,
            errorCount: 0,
            errorRate: 0.0,
            avgDurationMs: 6000.0,
            minDurationMs: 5000,
            maxDurationMs: 8000,
        );

        expect($stats->isSlowResponding(5000))->toBeTrue();
        expect($stats->isSlowResponding(10000))->toBeFalse();
    });
});

// =============================================================================
// Error Rate Calculation Tests
// =============================================================================

describe('Error rate calculations', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
    });

    it('calculates error rate correctly', function () {
        $this->analyticsService->recordExecution('test_tool', 100, true, 'ws-123');
        $this->analyticsService->recordExecution('test_tool', 100, true, 'ws-123');
        $this->analyticsService->recordExecution('test_tool', 100, false, 'ws-123');
        $this->analyticsService->recordExecution('test_tool', 100, false, 'ws-123');
        $this->analyticsService->flush();

        $metric = ToolMetric::forTool('test_tool')->first();

        expect($metric->error_rate)->toBe(50.0);
    });

    it('returns zero error rate when no errors', function () {
        $this->analyticsService->recordExecution('test_tool', 100, true, 'ws-123');
        $this->analyticsService->recordExecution('test_tool', 100, true, 'ws-123');
        $this->analyticsService->flush();

        $metric = ToolMetric::forTool('test_tool')->first();

        expect($metric->error_rate)->toBe(0.0);
    });

    it('handles zero calls gracefully', function () {
        $metric = new ToolMetric([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 0,
            'error_count' => 0,
            'total_duration_ms' => 0,
        ]);

        expect($metric->error_rate)->toBe(0.0);
    });
});

// =============================================================================
// Daily Trend Aggregation Tests
// =============================================================================

describe('Daily trend aggregation', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
    });

    it('returns daily data for usage trends', function () {
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->subDays(2)->toDateString(),
            'call_count' => 50,
            'error_count' => 5,
            'total_duration_ms' => 5000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 200,
        ]);

        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->subDays(1)->toDateString(),
            'call_count' => 75,
            'error_count' => 3,
            'total_duration_ms' => 7500,
            'min_duration_ms' => 60,
            'max_duration_ms' => 180,
        ]);

        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 100,
            'error_count' => 2,
            'total_duration_ms' => 10000,
            'min_duration_ms' => 40,
            'max_duration_ms' => 300,
        ]);

        $trends = $this->analyticsService->getUsageTrends('query_database', 7);

        expect($trends)->toHaveCount(7);

        $todayTrend = collect($trends)->firstWhere('date', now()->toDateString());
        expect($todayTrend['calls'])->toBe(100);
        expect($todayTrend['errors'])->toBe(2);
    });

    it('fills missing days with zeros', function () {
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 10,
            'error_count' => 0,
            'total_duration_ms' => 1000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 150,
        ]);

        $trends = $this->analyticsService->getUsageTrends('query_database', 7);

        expect($trends)->toHaveCount(7);

        $daysWithCalls = collect($trends)->filter(fn ($day) => $day['calls'] > 0)->count();
        expect($daysWithCalls)->toBe(1);
    });

    it('includes formatted dates', function () {
        $trends = $this->analyticsService->getUsageTrends('query_database', 7);

        foreach ($trends as $trend) {
            expect($trend)->toHaveKey('date');
            expect($trend)->toHaveKey('date_formatted');
            expect($trend['date'])->toMatch('/\d{4}-\d{2}-\d{2}/');
            expect($trend['date_formatted'])->toMatch('/[A-Z][a-z]{2} \d{1,2}/');
        }
    });
});

// =============================================================================
// Reporting Function Tests
// =============================================================================

describe('Reporting functions', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
    });

    it('getAllToolStats returns collection', function () {
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 100,
            'error_count' => 5,
            'total_duration_ms' => 10000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 300,
        ]);

        ToolMetric::create([
            'tool_name' => 'file_read',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 50,
            'error_count' => 1,
            'total_duration_ms' => 2500,
            'min_duration_ms' => 20,
            'max_duration_ms' => 100,
        ]);

        $stats = $this->analyticsService->getAllToolStats();

        expect($stats)->toHaveCount(2);
        expect($stats->first())->toBeInstanceOf(ToolStats::class);
        expect($stats->first()->toolName)->toBe('query_database');
        expect($stats->last()->toolName)->toBe('file_read');
    });

    it('getPopularTools returns top tools by call count', function () {
        foreach (['tool_a', 'tool_b', 'tool_c', 'tool_d', 'tool_e'] as $index => $toolName) {
            ToolMetric::create([
                'tool_name' => $toolName,
                'workspace_id' => null,
                'date' => now()->toDateString(),
                'call_count' => 100 - ($index * 20),
                'error_count' => 0,
                'total_duration_ms' => 5000,
                'min_duration_ms' => 50,
                'max_duration_ms' => 100,
            ]);
        }

        $popular = $this->analyticsService->getPopularTools(3);

        expect($popular)->toHaveCount(3);
        expect($popular[0]->toolName)->toBe('tool_a');
        expect($popular[1]->toolName)->toBe('tool_b');
        expect($popular[2]->toolName)->toBe('tool_c');
    });

    it('getErrorProneTools filters by minimum calls', function () {
        // Tool with high error rate but few calls (should be excluded)
        ToolMetric::create([
            'tool_name' => 'rarely_used_tool',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 5,
            'error_count' => 5,
            'total_duration_ms' => 500,
            'min_duration_ms' => 100,
            'max_duration_ms' => 100,
        ]);

        // Tool with moderate error rate and enough calls (should be included)
        ToolMetric::create([
            'tool_name' => 'problematic_tool',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 50,
            'error_count' => 15,
            'total_duration_ms' => 5000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 200,
        ]);

        // Tool with low error rate and many calls (should be lower priority)
        ToolMetric::create([
            'tool_name' => 'reliable_tool',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 100,
            'error_count' => 2,
            'total_duration_ms' => 10000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 150,
        ]);

        $errorProne = $this->analyticsService->getErrorProneTools(10);

        $toolNames = $errorProne->map(fn (ToolStats $s) => $s->toolName)->toArray();
        expect($toolNames)->not->toContain('rarely_used_tool');
        expect($errorProne->first()->toolName)->toBe('problematic_tool');
    });

    it('getWorkspaceStats returns aggregated workspace data', function () {
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => 'ws-123',
            'date' => now()->toDateString(),
            'call_count' => 50,
            'error_count' => 2,
            'total_duration_ms' => 5000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 200,
        ]);

        ToolMetric::create([
            'tool_name' => 'file_read',
            'workspace_id' => 'ws-123',
            'date' => now()->toDateString(),
            'call_count' => 30,
            'error_count' => 1,
            'total_duration_ms' => 1500,
            'min_duration_ms' => 20,
            'max_duration_ms' => 100,
        ]);

        // Different workspace (should not be included)
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => 'ws-456',
            'date' => now()->toDateString(),
            'call_count' => 100,
            'error_count' => 10,
            'total_duration_ms' => 10000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 300,
        ]);

        $stats = $this->analyticsService->getWorkspaceStats('ws-123');

        expect($stats['workspace_id'])->toBe('ws-123');
        expect($stats['total_calls'])->toBe(80);
        expect($stats['error_count'])->toBe(3);
        expect($stats['error_rate'])->toBe(3.75);
        expect($stats['unique_tools'])->toBe(2);
    });
});

// =============================================================================
// Tool Combination Tracking Tests
// =============================================================================

describe('Tool combination tracking', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
    });

    it('tracks tool combinations within session', function () {
        $sessionId = 'session-abc-123';

        $this->analyticsService->recordExecution('query_database', 100, true, 'ws-123', $sessionId);
        $this->analyticsService->recordExecution('file_read', 50, true, 'ws-123', $sessionId);
        $this->analyticsService->recordExecution('api_call', 200, true, 'ws-123', $sessionId);
        $this->analyticsService->flush();

        $combinations = DB::table('mcp_tool_combinations')
            ->where('workspace_id', 'ws-123')
            ->get();

        // 3 tools = 3 unique pairs (api_call+file_read, api_call+query_database, file_read+query_database)
        expect($combinations)->toHaveCount(3);
    });

    it('orders tool combinations alphabetically', function () {
        $sessionId = 'session-xyz';

        $this->analyticsService->recordExecution('zebra_tool', 100, true, 'ws-123', $sessionId);
        $this->analyticsService->recordExecution('alpha_tool', 100, true, 'ws-123', $sessionId);
        $this->analyticsService->flush();

        $combination = DB::table('mcp_tool_combinations')
            ->where('workspace_id', 'ws-123')
            ->first();

        expect($combination->tool_a)->toBe('alpha_tool');
        expect($combination->tool_b)->toBe('zebra_tool');
    });

    it('getToolCombinations returns most frequent', function () {
        $today = now()->toDateString();

        DB::table('mcp_tool_combinations')->insert([
            [
                'tool_a' => 'query_database',
                'tool_b' => 'file_read',
                'workspace_id' => 'ws-123',
                'date' => $today,
                'occurrence_count' => 50,
                'created_at' => now(),
                'updated_at' => now(),
            ],
            [
                'tool_a' => 'api_call',
                'tool_b' => 'query_database',
                'workspace_id' => 'ws-123',
                'date' => $today,
                'occurrence_count' => 30,
                'created_at' => now(),
                'updated_at' => now(),
            ],
        ]);

        $combinations = $this->analyticsService->getToolCombinations(5);

        expect($combinations)->toHaveCount(2);
        expect($combinations[0]['tool_a'])->toBe('query_database');
        expect($combinations[0]['tool_b'])->toBe('file_read');
        expect($combinations[0]['occurrences'])->toBe(50);
    });
});

// =============================================================================
// Date Range Filtering Tests
// =============================================================================

describe('Date range filtering', function () {
    beforeEach(function () {
        $this->analyticsService = new ToolAnalyticsService();
        Config::set('mcp.analytics.enabled', true);
    });

    it('getToolStats respects date range', function () {
        // Metric within range
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->subDays(5)->toDateString(),
            'call_count' => 50,
            'error_count' => 2,
            'total_duration_ms' => 5000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 200,
        ]);

        // Metric outside range (too old)
        ToolMetric::create([
            'tool_name' => 'query_database',
            'workspace_id' => null,
            'date' => now()->subDays(60)->toDateString(),
            'call_count' => 100,
            'error_count' => 10,
            'total_duration_ms' => 10000,
            'min_duration_ms' => 50,
            'max_duration_ms' => 300,
        ]);

        $stats = $this->analyticsService->getToolStats(
            'query_database',
            from: now()->subDays(30),
            to: now()
        );

        expect($stats->totalCalls)->toBe(50);
        expect($stats->errorCount)->toBe(2);
    });
});

// =============================================================================
// ToolMetric Model Tests
// =============================================================================

describe('ToolMetric model', function () {
    it('recordCall creates new record', function () {
        $metric = ToolMetric::recordCall('new_tool', 150, 'ws-123');

        expect($metric)->toBeInstanceOf(ToolMetric::class);
        expect($metric->tool_name)->toBe('new_tool');
        expect($metric->call_count)->toBe(1);
        expect($metric->error_count)->toBe(0);
        expect($metric->total_duration_ms)->toBe(150);
    });

    it('recordCall increments existing record', function () {
        ToolMetric::recordCall('existing_tool', 100, 'ws-123');
        $metric = ToolMetric::recordCall('existing_tool', 200, 'ws-123');

        expect($metric->call_count)->toBe(2);
        expect($metric->total_duration_ms)->toBe(300);
        expect($metric->min_duration_ms)->toBe(100);
        expect($metric->max_duration_ms)->toBe(200);
    });

    it('recordError increments error count', function () {
        $metric = ToolMetric::recordError('error_tool', 50, 'ws-123');

        expect($metric->call_count)->toBe(1);
        expect($metric->error_count)->toBe(1);
        expect($metric->error_rate)->toBe(100.0);
    });

    it('average_duration accessor works correctly', function () {
        $metric = new ToolMetric([
            'call_count' => 10,
            'total_duration_ms' => 1500,
        ]);

        expect($metric->average_duration)->toBe(150.0);
    });

    it('average_duration_for_humans returns milliseconds', function () {
        $metric = new ToolMetric([
            'call_count' => 10,
            'total_duration_ms' => 5000,
        ]);

        expect($metric->average_duration_for_humans)->toBe('500ms');
    });

    it('average_duration_for_humans returns seconds', function () {
        $metric = new ToolMetric([
            'call_count' => 10,
            'total_duration_ms' => 25000,
        ]);

        expect($metric->average_duration_for_humans)->toBe('2.5s');
    });

    it('average_duration_for_humans returns dash for zero', function () {
        $metric = new ToolMetric([
            'call_count' => 0,
            'total_duration_ms' => 0,
        ]);

        expect($metric->average_duration_for_humans)->toBe('-');
    });

    it('forDateRange scope filters correctly', function () {
        ToolMetric::create([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->subDays(5)->toDateString(),
            'call_count' => 10,
            'error_count' => 0,
            'total_duration_ms' => 1000,
        ]);

        ToolMetric::create([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->subDays(15)->toDateString(),
            'call_count' => 20,
            'error_count' => 0,
            'total_duration_ms' => 2000,
        ]);

        $metrics = ToolMetric::forDateRange(now()->subDays(10), now())->get();

        expect($metrics)->toHaveCount(1);
        expect($metrics->first()->call_count)->toBe(10);
    });

    it('lastDays scope filters correctly', function () {
        ToolMetric::create([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->subDays(3)->toDateString(),
            'call_count' => 10,
            'error_count' => 0,
            'total_duration_ms' => 1000,
        ]);

        ToolMetric::create([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->subDays(10)->toDateString(),
            'call_count' => 20,
            'error_count' => 0,
            'total_duration_ms' => 2000,
        ]);

        $metrics = ToolMetric::lastDays(7)->get();

        expect($metrics)->toHaveCount(1);
    });

    it('today scope filters correctly', function () {
        ToolMetric::create([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->toDateString(),
            'call_count' => 10,
            'error_count' => 0,
            'total_duration_ms' => 1000,
        ]);

        ToolMetric::create([
            'tool_name' => 'test_tool',
            'workspace_id' => null,
            'date' => now()->subDays(1)->toDateString(),
            'call_count' => 20,
            'error_count' => 0,
            'total_duration_ms' => 2000,
        ]);

        $metrics = ToolMetric::today()->get();

        expect($metrics)->toHaveCount(1);
        expect($metrics->first()->call_count)->toBe(10);
    });
});
