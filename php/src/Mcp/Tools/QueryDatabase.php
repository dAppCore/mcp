<?php

declare(strict_types=1);

namespace Core\Mcp\Tools;

use Core\Mcp\Exceptions\ForbiddenQueryException;
use Core\Mcp\Exceptions\QueryTimeoutException;
use Core\Mcp\Services\QueryAuditService;
use Core\Mcp\Services\QueryExecutionService;
use Core\Mcp\Services\SqlQueryValidator;
use Illuminate\Contracts\JsonSchema\JsonSchema;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\DB;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

/**
 * MCP Tool for executing read-only SQL queries.
 *
 * Security measures:
 * 1. Uses configurable read-only database connection
 * 2. Validates queries against blocked keywords and patterns
 * 3. Optional whitelist-based query validation
 * 4. Blocks access to sensitive tables
 * 5. Enforces tier-based row limits with truncation warnings
 * 6. Enforces per-query timeout limits
 * 7. Comprehensive audit logging of all query attempts
 */
class QueryDatabase extends Tool
{
    protected string $description = 'Execute a read-only SQL SELECT query against the database';

    private SqlQueryValidator $validator;

    private QueryExecutionService $executionService;

    private QueryAuditService $auditService;

    public function __construct(
        ?QueryExecutionService $executionService = null,
        ?QueryAuditService $auditService = null
    ) {
        $this->validator = $this->createValidator();
        $this->auditService = $auditService ?? app(QueryAuditService::class);
        $this->executionService = $executionService ?? app(QueryExecutionService::class);
    }

    public function handle(Request $request): Response
    {
        $query = $request->input('query');
        $explain = $request->input('explain', false);

        // Extract context from request for audit logging
        $workspaceId = $this->getWorkspaceId($request);
        $userId = $this->getUserId($request);
        $userIp = $this->getUserIp($request);
        $sessionId = $request->input('session_id');

        if (empty($query)) {
            return $this->errorResponse('Query is required');
        }

        // Validate the query - log blocked queries
        try {
            $this->validator->validate($query);
        } catch (ForbiddenQueryException $e) {
            $this->auditService->recordBlocked(
                query: $query,
                bindings: [],
                reason: $e->reason,
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: ['session_id' => $sessionId]
            );

            return $this->errorResponse($e->getMessage());
        }

        // Check for blocked tables
        $blockedTable = $this->checkBlockedTables($query);
        if ($blockedTable !== null) {
            $this->auditService->recordBlocked(
                query: $query,
                bindings: [],
                reason: "Access to blocked table: {$blockedTable}",
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: ['session_id' => $sessionId, 'blocked_table' => $blockedTable]
            );

            return $this->errorResponse(
                sprintf("Access to table '%s' is not permitted", $blockedTable)
            );
        }

        try {
            $connection = $this->getConnection();

            // If explain is requested, run EXPLAIN first
            if ($explain) {
                return $this->handleExplain($connection, $query, $workspaceId, $userId, $userIp, $sessionId);
            }

            // Execute query with tier-based limits, timeout, and audit logging
            $result = $this->executionService->execute(
                query: $query,
                connection: $connection,
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: [
                    'session_id' => $sessionId,
                    'explain_requested' => false,
                ]
            );

            // Build response with data and metadata
            $response = [
                'data' => $result['data'],
                'meta' => $result['meta'],
            ];

            // Add warning if results were truncated
            if ($result['meta']['truncated']) {
                $response['warning'] = $result['meta']['warning'];
            }

            return Response::text(json_encode($response, JSON_PRETTY_PRINT));
        } catch (QueryTimeoutException $e) {
            return $this->errorResponse(
                'Query timed out: '.$e->getMessage().
                ' Consider adding more specific filters or indexes.'
            );
        } catch (\Exception $e) {
            // Log the actual error for debugging but return sanitised message
            report($e);

            return $this->errorResponse('Query execution failed: '.$this->sanitiseErrorMessage($e->getMessage()));
        }
    }

    public function schema(JsonSchema $schema): array
    {
        return [
            'query' => $schema->string('SQL SELECT query to execute. Only read-only SELECT queries are permitted.'),
            'explain' => $schema->boolean('If true, runs EXPLAIN on the query instead of executing it. Useful for query optimisation and debugging.')->default(false),
        ];
    }

    /**
     * Create the SQL validator with configuration.
     */
    private function createValidator(): SqlQueryValidator
    {
        $useWhitelist = Config::get('mcp.database.use_whitelist', true);
        $customPatterns = Config::get('mcp.database.whitelist_patterns', []);

        $validator = new SqlQueryValidator(null, $useWhitelist);

        foreach ($customPatterns as $pattern) {
            $validator->addWhitelistPattern($pattern);
        }

        return $validator;
    }

    /**
     * Get the database connection to use.
     *
     * @throws \RuntimeException If the configured connection is invalid
     */
    private function getConnection(): ?string
    {
        $connection = Config::get('mcp.database.connection');

        // If configured connection doesn't exist, throw exception
        if ($connection && ! Config::has("database.connections.{$connection}")) {
            throw new \RuntimeException(
                "Invalid MCP database connection '{$connection}' configured. ".
                "Please ensure 'database.connections.{$connection}' exists in your database configuration."
            );
        }

        return $connection;
    }

    /**
     * Check if the query references any blocked tables.
     */
    private function checkBlockedTables(string $query): ?string
    {
        $blockedTables = Config::get('mcp.database.blocked_tables', []);

        foreach ($blockedTables as $table) {
            // Check for table references in various formats
            $patterns = [
                '/\bFROM\s+`?'.preg_quote($table, '/').'`?\b/i',
                '/\bJOIN\s+`?'.preg_quote($table, '/').'`?\b/i',
                '/\b'.preg_quote($table, '/').'\./i', // table.column format
            ];

            foreach ($patterns as $pattern) {
                if (preg_match($pattern, $query)) {
                    return $table;
                }
            }
        }

        return null;
    }

    /**
     * Extract workspace ID from request context.
     */
    private function getWorkspaceId(Request $request): ?int
    {
        // Try to get from request context or metadata
        $workspaceId = $request->input('workspace_id');
        if ($workspaceId !== null) {
            return (int) $workspaceId;
        }

        // Try from auth context
        if (function_exists('workspace') && workspace()) {
            return workspace()->id;
        }

        return null;
    }

    /**
     * Extract user ID from request context.
     */
    private function getUserId(Request $request): ?int
    {
        // Try to get from request context
        $userId = $request->input('user_id');
        if ($userId !== null) {
            return (int) $userId;
        }

        // Try from auth
        if (auth()->check()) {
            return auth()->id();
        }

        return null;
    }

    /**
     * Extract user IP from request context.
     */
    private function getUserIp(Request $request): ?string
    {
        // Try from request metadata
        $ip = $request->input('user_ip');
        if ($ip !== null) {
            return $ip;
        }

        // Try from HTTP request
        if (request()) {
            return request()->ip();
        }

        return null;
    }

    /**
     * Sanitise database error messages to avoid leaking sensitive information.
     */
    private function sanitiseErrorMessage(string $message): string
    {
        // Remove specific database paths, credentials, etc.
        $message = preg_replace('/\/[^\s]+/', '[path]', $message);
        $message = preg_replace('/at \d+\.\d+\.\d+\.\d+/', 'at [ip]', $message);

        // Truncate long messages
        if (strlen($message) > 200) {
            $message = substr($message, 0, 200).'...';
        }

        return $message;
    }

    /**
     * Handle EXPLAIN query execution.
     */
    private function handleExplain(
        ?string $connection,
        string $query,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        ?string $sessionId = null
    ): Response {
        $startTime = microtime(true);

        try {
            // Run EXPLAIN on the query
            $explainResults = DB::connection($connection)->select("EXPLAIN {$query}");
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            // Also try to get extended information if MySQL/MariaDB
            $warnings = [];
            try {
                $warnings = DB::connection($connection)->select('SHOW WARNINGS');
            } catch (\Exception $e) {
                // SHOW WARNINGS may not be available on all databases
            }

            $response = [
                'explain' => $explainResults,
                'query' => $query,
            ];

            if (! empty($warnings)) {
                $response['warnings'] = $warnings;
            }

            // Add helpful interpretation
            $response['interpretation'] = $this->interpretExplain($explainResults);

            // Log the EXPLAIN query
            $this->auditService->recordSuccess(
                query: "EXPLAIN {$query}",
                bindings: [],
                durationMs: $durationMs,
                rowCount: count($explainResults),
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: ['session_id' => $sessionId, 'explain_requested' => true]
            );

            return Response::text(json_encode($response, JSON_PRETTY_PRINT));
        } catch (\Exception $e) {
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            $this->auditService->recordError(
                query: "EXPLAIN {$query}",
                bindings: [],
                errorMessage: $e->getMessage(),
                durationMs: $durationMs,
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: ['session_id' => $sessionId, 'explain_requested' => true]
            );

            report($e);

            return $this->errorResponse('EXPLAIN failed: '.$this->sanitiseErrorMessage($e->getMessage()));
        }
    }

    /**
     * Provide human-readable interpretation of EXPLAIN results.
     */
    private function interpretExplain(array $explainResults): array
    {
        $interpretation = [];

        foreach ($explainResults as $row) {
            $rowAnalysis = [];

            // Convert stdClass to array for easier access
            $rowArray = (array) $row;

            // Check for full table scan
            if (isset($rowArray['type']) && $rowArray['type'] === 'ALL') {
                $rowAnalysis[] = 'WARNING: Full table scan detected. Consider adding an index.';
            }

            // Check for filesort
            if (isset($rowArray['Extra']) && str_contains($rowArray['Extra'], 'Using filesort')) {
                $rowAnalysis[] = 'INFO: Using filesort. Query may benefit from an index on ORDER BY columns.';
            }

            // Check for temporary table
            if (isset($rowArray['Extra']) && str_contains($rowArray['Extra'], 'Using temporary')) {
                $rowAnalysis[] = 'INFO: Using temporary table. Consider optimizing the query.';
            }

            // Check rows examined
            if (isset($rowArray['rows']) && $rowArray['rows'] > 10000) {
                $rowAnalysis[] = sprintf('WARNING: High row count (%d rows). Query may be slow.', $rowArray['rows']);
            }

            // Check if index is used
            if (isset($rowArray['key']) && $rowArray['key'] !== null) {
                $rowAnalysis[] = sprintf('GOOD: Using index: %s', $rowArray['key']);
            }

            if (! empty($rowAnalysis)) {
                $interpretation[] = [
                    'table' => $rowArray['table'] ?? 'unknown',
                    'analysis' => $rowAnalysis,
                ];
            }
        }

        return $interpretation;
    }

    /**
     * Create an error response.
     */
    private function errorResponse(string $message): Response
    {
        return Response::text(json_encode(['error' => $message]));
    }
}
