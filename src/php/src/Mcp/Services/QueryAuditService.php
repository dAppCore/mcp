<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Support\Facades\Log;

/**
 * Query Audit Service - records all SQL query attempts for compliance and security.
 *
 * Provides comprehensive logging of query attempts including:
 * - User and workspace identification
 * - Query text and parameters
 * - Execution status and timing
 * - Security violation detection
 */
class QueryAuditService
{
    /**
     * Log channel for query audits.
     */
    protected const LOG_CHANNEL = 'mcp-queries';

    /**
     * Query status constants.
     */
    public const STATUS_SUCCESS = 'success';

    public const STATUS_BLOCKED = 'blocked';

    public const STATUS_TIMEOUT = 'timeout';

    public const STATUS_ERROR = 'error';

    public const STATUS_TRUNCATED = 'truncated';

    /**
     * Record a query attempt.
     *
     * @param array<string, mixed> $bindings
     * @param array<string, mixed> $context
     */
    public function record(
        string $query,
        array $bindings,
        string $status,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        ?int $durationMs = null,
        ?int $rowCount = null,
        ?string $errorMessage = null,
        ?string $errorCode = null,
        array $context = []
    ): void {
        $logData = [
            'timestamp' => now()->toIso8601String(),
            'query' => $this->sanitiseQuery($query),
            'bindings_count' => count($bindings),
            'status' => $status,
            'workspace_id' => $workspaceId,
            'user_id' => $userId,
            'user_ip' => $userIp,
            'duration_ms' => $durationMs,
            'row_count' => $rowCount,
            'request_id' => request()?->header('X-Request-ID'),
            'session_id' => $context['session_id'] ?? null,
            'agent_type' => $context['agent_type'] ?? null,
            'tier' => $context['tier'] ?? 'default',
        ];

        if ($errorMessage !== null) {
            $logData['error_message'] = $this->sanitiseErrorMessage($errorMessage);
        }

        if ($errorCode !== null) {
            $logData['error_code'] = $errorCode;
        }

        // Add additional context fields
        foreach (['connection', 'explain_requested', 'truncated_at'] as $key) {
            if (isset($context[$key])) {
                $logData[$key] = $context[$key];
            }
        }

        // Determine log level based on status
        $level = match ($status) {
            self::STATUS_SUCCESS => 'info',
            self::STATUS_TRUNCATED => 'notice',
            self::STATUS_TIMEOUT => 'warning',
            self::STATUS_BLOCKED => 'warning',
            self::STATUS_ERROR => 'error',
            default => 'info',
        };

        $this->log($level, 'MCP query audit', $logData);

        // Additional security logging for blocked queries
        if ($status === self::STATUS_BLOCKED) {
            $this->logSecurityEvent($query, $bindings, $workspaceId, $userId, $userIp, $errorMessage);
        }
    }

    /**
     * Record a successful query.
     *
     * @param array<string, mixed> $bindings
     * @param array<string, mixed> $context
     */
    public function recordSuccess(
        string $query,
        array $bindings,
        int $durationMs,
        int $rowCount,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        array $context = []
    ): void {
        $this->record(
            query: $query,
            bindings: $bindings,
            status: self::STATUS_SUCCESS,
            workspaceId: $workspaceId,
            userId: $userId,
            userIp: $userIp,
            durationMs: $durationMs,
            rowCount: $rowCount,
            context: $context
        );
    }

    /**
     * Record a blocked query (security violation).
     *
     * @param array<string, mixed> $bindings
     * @param array<string, mixed> $context
     */
    public function recordBlocked(
        string $query,
        array $bindings,
        string $reason,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        array $context = []
    ): void {
        $this->record(
            query: $query,
            bindings: $bindings,
            status: self::STATUS_BLOCKED,
            workspaceId: $workspaceId,
            userId: $userId,
            userIp: $userIp,
            errorMessage: $reason,
            errorCode: 'QUERY_BLOCKED',
            context: $context
        );
    }

    /**
     * Record a query timeout.
     *
     * @param array<string, mixed> $bindings
     * @param array<string, mixed> $context
     */
    public function recordTimeout(
        string $query,
        array $bindings,
        int $timeoutSeconds,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        array $context = []
    ): void {
        $this->record(
            query: $query,
            bindings: $bindings,
            status: self::STATUS_TIMEOUT,
            workspaceId: $workspaceId,
            userId: $userId,
            userIp: $userIp,
            durationMs: $timeoutSeconds * 1000,
            errorMessage: "Query exceeded timeout of {$timeoutSeconds} seconds",
            errorCode: 'QUERY_TIMEOUT',
            context: $context
        );
    }

    /**
     * Record a query error.
     *
     * @param array<string, mixed> $bindings
     * @param array<string, mixed> $context
     */
    public function recordError(
        string $query,
        array $bindings,
        string $errorMessage,
        ?int $durationMs = null,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        array $context = []
    ): void {
        $this->record(
            query: $query,
            bindings: $bindings,
            status: self::STATUS_ERROR,
            workspaceId: $workspaceId,
            userId: $userId,
            userIp: $userIp,
            durationMs: $durationMs,
            errorMessage: $errorMessage,
            errorCode: 'QUERY_ERROR',
            context: $context
        );
    }

    /**
     * Record a truncated result (result size limit exceeded).
     *
     * @param array<string, mixed> $bindings
     * @param array<string, mixed> $context
     */
    public function recordTruncated(
        string $query,
        array $bindings,
        int $durationMs,
        int $returnedRows,
        int $maxRows,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        array $context = []
    ): void {
        $context['truncated_at'] = $maxRows;

        $this->record(
            query: $query,
            bindings: $bindings,
            status: self::STATUS_TRUNCATED,
            workspaceId: $workspaceId,
            userId: $userId,
            userIp: $userIp,
            durationMs: $durationMs,
            rowCount: $returnedRows,
            errorMessage: "Results truncated from {$returnedRows}+ to {$maxRows} rows",
            errorCode: 'RESULT_TRUNCATED',
            context: $context
        );
    }

    /**
     * Log a security event for blocked queries.
     *
     * @param array<string, mixed> $bindings
     */
    protected function logSecurityEvent(
        string $query,
        array $bindings,
        ?int $workspaceId,
        ?int $userId,
        ?string $userIp,
        ?string $reason
    ): void {
        Log::channel('security')->warning('MCP query blocked by security policy', [
            'type' => 'mcp_query_blocked',
            'query_hash' => hash('sha256', $query),
            'query_length' => strlen($query),
            'workspace_id' => $workspaceId,
            'user_id' => $userId,
            'user_ip' => $userIp,
            'reason' => $reason,
            'timestamp' => now()->toIso8601String(),
        ]);
    }

    /**
     * Sanitise query for logging (remove sensitive data patterns).
     */
    protected function sanitiseQuery(string $query): string
    {
        // Truncate very long queries
        if (strlen($query) > 2000) {
            $query = substr($query, 0, 2000).'... [TRUNCATED]';
        }

        return $query;
    }

    /**
     * Sanitise error messages to avoid leaking sensitive information.
     */
    protected function sanitiseErrorMessage(string $message): string
    {
        // Remove specific file paths
        $message = preg_replace('/\/[^\s]+/', '[path]', $message) ?? $message;

        // Remove IP addresses
        $message = preg_replace('/\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/', '[ip]', $message) ?? $message;

        // Truncate long messages
        if (strlen($message) > 500) {
            $message = substr($message, 0, 500).'...';
        }

        return $message;
    }

    /**
     * Write to the appropriate log channel.
     *
     * @param array<string, mixed> $context
     */
    protected function log(string $level, string $message, array $context): void
    {
        // Use dedicated channel if configured, otherwise use default
        $channel = config('mcp.audit.log_channel', self::LOG_CHANNEL);

        try {
            Log::channel($channel)->log($level, $message, $context);
        } catch (\Exception $e) {
            // Fallback to default logger if channel doesn't exist
            Log::log($level, $message, $context);
        }
    }
}
