<?php

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Services\QueryAuditService;
use Illuminate\Support\Facades\Log;
use Tests\TestCase;

class QueryAuditServiceTest extends TestCase
{
    protected QueryAuditService $auditService;

    protected function setUp(): void
    {
        parent::setUp();

        $this->auditService = new QueryAuditService();
    }

    public function test_record_logs_success_status(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $level === 'info'
                    && $message === 'MCP query audit'
                    && $context['status'] === QueryAuditService::STATUS_SUCCESS
                    && str_contains($context['query'], 'SELECT');
            });

        $this->auditService->record(
            query: 'SELECT * FROM users',
            bindings: [],
            status: QueryAuditService::STATUS_SUCCESS,
            durationMs: 50,
            rowCount: 10
        );
    }

    public function test_record_logs_blocked_status_with_warning_level(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $level === 'warning'
                    && $context['status'] === QueryAuditService::STATUS_BLOCKED;
            });

        // Security channel logging for blocked queries
        Log::shouldReceive('channel')
            ->with('security')
            ->andReturnSelf();

        Log::shouldReceive('warning')
            ->once()
            ->withArgs(function ($message, $context) {
                return $context['type'] === 'mcp_query_blocked';
            });

        $this->auditService->record(
            query: 'SELECT * FROM users; DROP TABLE users;',
            bindings: [],
            status: QueryAuditService::STATUS_BLOCKED,
            errorMessage: 'Multiple statements detected'
        );
    }

    public function test_record_logs_timeout_status(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $level === 'warning'
                    && $context['status'] === QueryAuditService::STATUS_TIMEOUT
                    && $context['error_code'] === 'QUERY_TIMEOUT';
            });

        $this->auditService->recordTimeout(
            query: 'SELECT * FROM large_table',
            bindings: [],
            timeoutSeconds: 30,
            workspaceId: 1
        );
    }

    public function test_record_logs_truncated_status(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $level === 'notice'
                    && $context['status'] === QueryAuditService::STATUS_TRUNCATED
                    && $context['error_code'] === 'RESULT_TRUNCATED'
                    && $context['truncated_at'] === 100;
            });

        $this->auditService->recordTruncated(
            query: 'SELECT * FROM users',
            bindings: [],
            durationMs: 150,
            returnedRows: 500,
            maxRows: 100,
            workspaceId: 1
        );
    }

    public function test_record_logs_error_status(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $level === 'error'
                    && $context['status'] === QueryAuditService::STATUS_ERROR
                    && str_contains($context['error_message'], 'Table not found');
            });

        $this->auditService->recordError(
            query: 'SELECT * FROM nonexistent',
            bindings: [],
            errorMessage: 'Table not found',
            durationMs: 5
        );
    }

    public function test_record_includes_workspace_and_user_context(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $context['workspace_id'] === 123
                    && $context['user_id'] === 456
                    && $context['user_ip'] === '192.168.1.1';
            });

        $this->auditService->recordSuccess(
            query: 'SELECT 1',
            bindings: [],
            durationMs: 1,
            rowCount: 1,
            workspaceId: 123,
            userId: 456,
            userIp: '192.168.1.1'
        );
    }

    public function test_record_includes_session_and_tier_context(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $context['session_id'] === 'test-session-123'
                    && $context['tier'] === 'enterprise';
            });

        $this->auditService->recordSuccess(
            query: 'SELECT 1',
            bindings: [],
            durationMs: 1,
            rowCount: 1,
            context: [
                'session_id' => 'test-session-123',
                'tier' => 'enterprise',
            ]
        );
    }

    public function test_record_sanitises_long_queries(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return strlen($context['query']) <= 2013 // 2000 + length of "... [TRUNCATED]"
                    && str_contains($context['query'], '[TRUNCATED]');
            });

        $longQuery = 'SELECT ' . str_repeat('a', 3000) . ' FROM table';

        $this->auditService->recordSuccess(
            query: $longQuery,
            bindings: [],
            durationMs: 1,
            rowCount: 1
        );
    }

    public function test_record_sanitises_error_messages(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return str_contains($context['error_message'], '[path]')
                    && str_contains($context['error_message'], '[ip]')
                    && ! str_contains($context['error_message'], '/var/www')
                    && ! str_contains($context['error_message'], '192.168.1.100');
            });

        $this->auditService->recordError(
            query: 'SELECT 1',
            bindings: [],
            errorMessage: 'Error at /var/www/app/file.php connecting to 192.168.1.100'
        );
    }

    public function test_blocked_queries_also_log_to_security_channel(): void
    {
        Log::shouldReceive('channel')
            ->with('mcp-queries')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once();

        Log::shouldReceive('channel')
            ->with('security')
            ->andReturnSelf();

        Log::shouldReceive('warning')
            ->once()
            ->withArgs(function ($message, $context) {
                return $message === 'MCP query blocked by security policy'
                    && $context['type'] === 'mcp_query_blocked'
                    && isset($context['query_hash'])
                    && $context['reason'] === 'SQL injection detected';
            });

        $this->auditService->recordBlocked(
            query: "SELECT * FROM users WHERE id = '1' OR '1'='1'",
            bindings: [],
            reason: 'SQL injection detected',
            workspaceId: 1,
            userId: 2,
            userIp: '10.0.0.1'
        );
    }

    public function test_record_counts_bindings_without_logging_values(): void
    {
        Log::shouldReceive('channel')
            ->andReturnSelf();

        Log::shouldReceive('log')
            ->once()
            ->withArgs(function ($level, $message, $context) {
                return $context['bindings_count'] === 3;
            });

        $this->auditService->recordSuccess(
            query: 'SELECT * FROM users WHERE id = ? AND status = ? AND role = ?',
            bindings: [1, 'active', 'admin'],
            durationMs: 10,
            rowCount: 1
        );
    }
}
