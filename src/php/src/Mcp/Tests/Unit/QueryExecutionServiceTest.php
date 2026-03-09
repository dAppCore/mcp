<?php

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Exceptions\QueryTimeoutException;
use Core\Mcp\Services\QueryAuditService;
use Core\Mcp\Services\QueryExecutionService;
use Core\Tenant\Services\EntitlementService;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\DB;
use Mockery;
use Tests\TestCase;

class QueryExecutionServiceTest extends TestCase
{
    protected QueryExecutionService $executionService;

    protected QueryAuditService $auditMock;

    protected function setUp(): void
    {
        parent::setUp();

        $this->auditMock = Mockery::mock(QueryAuditService::class);
        $this->auditMock->shouldReceive('recordSuccess')->byDefault();
        $this->auditMock->shouldReceive('recordTruncated')->byDefault();
        $this->auditMock->shouldReceive('recordError')->byDefault();
        $this->auditMock->shouldReceive('recordTimeout')->byDefault();

        $this->executionService = new QueryExecutionService($this->auditMock);
    }

    protected function tearDown(): void
    {
        Mockery::close();
        parent::tearDown();
    }

    public function test_get_limits_for_tier_returns_correct_defaults(): void
    {
        $freeLimits = $this->executionService->getLimitsForTier('free');
        $this->assertEquals(100, $freeLimits['max_rows']);
        $this->assertEquals(5, $freeLimits['timeout_seconds']);

        $starterLimits = $this->executionService->getLimitsForTier('starter');
        $this->assertEquals(500, $starterLimits['max_rows']);
        $this->assertEquals(10, $starterLimits['timeout_seconds']);

        $professionalLimits = $this->executionService->getLimitsForTier('professional');
        $this->assertEquals(1000, $professionalLimits['max_rows']);
        $this->assertEquals(30, $professionalLimits['timeout_seconds']);

        $enterpriseLimits = $this->executionService->getLimitsForTier('enterprise');
        $this->assertEquals(5000, $enterpriseLimits['max_rows']);
        $this->assertEquals(60, $enterpriseLimits['timeout_seconds']);

        $unlimitedLimits = $this->executionService->getLimitsForTier('unlimited');
        $this->assertEquals(10000, $unlimitedLimits['max_rows']);
        $this->assertEquals(120, $unlimitedLimits['timeout_seconds']);
    }

    public function test_get_limits_for_tier_uses_config_overrides(): void
    {
        Config::set('mcp.database.tier_limits', [
            'free' => [
                'max_rows' => 50,
                'timeout_seconds' => 3,
            ],
        ]);

        $limits = $this->executionService->getLimitsForTier('free');

        $this->assertEquals(50, $limits['max_rows']);
        $this->assertEquals(3, $limits['timeout_seconds']);
    }

    public function test_get_limits_for_unknown_tier_falls_back_to_free(): void
    {
        $limits = $this->executionService->getLimitsForTier('nonexistent');

        $this->assertEquals(100, $limits['max_rows']);
        $this->assertEquals(5, $limits['timeout_seconds']);
    }

    public function test_get_available_tiers_returns_all_tiers(): void
    {
        $tiers = $this->executionService->getAvailableTiers();

        $this->assertArrayHasKey('free', $tiers);
        $this->assertArrayHasKey('starter', $tiers);
        $this->assertArrayHasKey('professional', $tiers);
        $this->assertArrayHasKey('enterprise', $tiers);
        $this->assertArrayHasKey('unlimited', $tiers);

        foreach ($tiers as $tier => $limits) {
            $this->assertArrayHasKey('max_rows', $limits);
            $this->assertArrayHasKey('timeout_seconds', $limits);
        }
    }

    public function test_execute_returns_data_with_metadata(): void
    {
        // Use SQLite in-memory for testing
        Config::set('database.default', 'sqlite');
        Config::set('database.connections.sqlite', [
            'driver' => 'sqlite',
            'database' => ':memory:',
        ]);

        DB::connection('sqlite')->statement('CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)');
        DB::connection('sqlite')->insert('INSERT INTO test_table (id, name) VALUES (1, "Test")');

        $this->auditMock->shouldReceive('recordSuccess')
            ->once()
            ->withArgs(function ($query, $bindings, $durationMs, $rowCount) {
                return str_contains($query, 'test_table') && $rowCount === 1;
            });

        $result = $this->executionService->execute(
            query: 'SELECT * FROM test_table',
            connection: 'sqlite'
        );

        $this->assertArrayHasKey('data', $result);
        $this->assertArrayHasKey('meta', $result);
        $this->assertCount(1, $result['data']);
        $this->assertEquals(1, $result['meta']['rows_returned']);
        $this->assertFalse($result['meta']['truncated']);
        $this->assertNull($result['meta']['warning']);
    }

    public function test_execute_truncates_results_when_exceeding_tier_limit(): void
    {
        // Use SQLite in-memory for testing
        Config::set('database.default', 'sqlite');
        Config::set('database.connections.sqlite', [
            'driver' => 'sqlite',
            'database' => ':memory:',
        ]);
        Config::set('mcp.database.default_tier', 'free'); // 100 row limit

        // Create a table with more than 100 rows
        DB::connection('sqlite')->statement('CREATE TABLE large_table (id INTEGER PRIMARY KEY, name TEXT)');
        for ($i = 1; $i <= 150; $i++) {
            DB::connection('sqlite')->insert('INSERT INTO large_table (id, name) VALUES (?, ?)', [$i, "Row {$i}"]);
        }

        $this->auditMock->shouldReceive('recordTruncated')
            ->once()
            ->withArgs(function ($query, $bindings, $durationMs, $returnedRows, $maxRows) {
                return $returnedRows === 150 && $maxRows === 100;
            });

        $result = $this->executionService->execute(
            query: 'SELECT * FROM large_table',
            connection: 'sqlite'
        );

        $this->assertCount(100, $result['data']);
        $this->assertTrue($result['meta']['truncated']);
        $this->assertEquals(100, $result['meta']['rows_returned']);
        $this->assertStringContains('150+', (string) $result['meta']['rows_total']);
        $this->assertNotNull($result['meta']['warning']);
    }

    public function test_execute_includes_tier_in_metadata(): void
    {
        Config::set('database.default', 'sqlite');
        Config::set('database.connections.sqlite', [
            'driver' => 'sqlite',
            'database' => ':memory:',
        ]);
        Config::set('mcp.database.default_tier', 'professional');

        DB::connection('sqlite')->statement('CREATE TABLE test_table (id INTEGER PRIMARY KEY)');

        $result = $this->executionService->execute(
            query: 'SELECT * FROM test_table',
            connection: 'sqlite'
        );

        $this->assertEquals('professional', $result['meta']['tier']);
        $this->assertEquals(1000, $result['meta']['max_rows']);
    }

    public function test_execute_logs_errors_on_failure(): void
    {
        Config::set('database.default', 'sqlite');
        Config::set('database.connections.sqlite', [
            'driver' => 'sqlite',
            'database' => ':memory:',
        ]);

        $this->auditMock->shouldReceive('recordError')
            ->once()
            ->withArgs(function ($query, $bindings, $errorMessage) {
                return str_contains($query, 'nonexistent_table');
            });

        $this->expectException(\Exception::class);

        $this->executionService->execute(
            query: 'SELECT * FROM nonexistent_table',
            connection: 'sqlite'
        );
    }

    public function test_execute_passes_context_to_audit_service(): void
    {
        Config::set('database.default', 'sqlite');
        Config::set('database.connections.sqlite', [
            'driver' => 'sqlite',
            'database' => ':memory:',
        ]);

        DB::connection('sqlite')->statement('CREATE TABLE test_table (id INTEGER PRIMARY KEY)');

        $this->auditMock->shouldReceive('recordSuccess')
            ->once()
            ->withArgs(function ($query, $bindings, $durationMs, $rowCount, $workspaceId, $userId, $userIp, $context) {
                return $workspaceId === 123
                    && $userId === 456
                    && $userIp === '192.168.1.1'
                    && isset($context['session_id'])
                    && $context['session_id'] === 'test-session';
            });

        $this->executionService->execute(
            query: 'SELECT * FROM test_table',
            connection: 'sqlite',
            workspaceId: 123,
            userId: 456,
            userIp: '192.168.1.1',
            context: ['session_id' => 'test-session']
        );
    }

    /**
     * Helper to assert string contains substring.
     */
    protected function assertStringContains(string $needle, string $haystack): void
    {
        $this->assertTrue(
            str_contains($haystack, $needle),
            "Failed asserting that '{$haystack}' contains '{$needle}'"
        );
    }
}
