<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Exceptions\QueryTimeoutException;
use Core\Mcp\Exceptions\ResultSizeLimitException;
use Core\Tenant\Services\EntitlementService;
use Illuminate\Database\Connection;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\DB;
use PDO;

/**
 * Query Execution Service - secure query execution with tier-based limits.
 *
 * Provides:
 * - Tier-based row limits with truncation warnings
 * - Query timeout enforcement
 * - Comprehensive audit logging
 */
class QueryExecutionService
{
    /**
     * Feature code for max rows entitlement.
     */
    public const FEATURE_MAX_ROWS = 'mcp.query_max_rows';

    /**
     * Feature code for query timeout entitlement.
     */
    public const FEATURE_QUERY_TIMEOUT = 'mcp.query_timeout';

    /**
     * Default tier limits.
     */
    protected const DEFAULT_TIER_LIMITS = [
        'free' => [
            'max_rows' => 100,
            'timeout_seconds' => 5,
        ],
        'starter' => [
            'max_rows' => 500,
            'timeout_seconds' => 10,
        ],
        'professional' => [
            'max_rows' => 1000,
            'timeout_seconds' => 30,
        ],
        'enterprise' => [
            'max_rows' => 5000,
            'timeout_seconds' => 60,
        ],
        'unlimited' => [
            'max_rows' => 10000,
            'timeout_seconds' => 120,
        ],
    ];

    public function __construct(
        protected QueryAuditService $auditService,
        protected ?EntitlementService $entitlementService = null
    ) {}

    /**
     * Execute a query with tier-based limits and audit logging.
     *
     * @param array<string, mixed> $context Additional context for logging
     * @return array{data: array, meta: array}
     *
     * @throws QueryTimeoutException
     */
    public function execute(
        string $query,
        ?string $connection = null,
        ?int $workspaceId = null,
        ?int $userId = null,
        ?string $userIp = null,
        array $context = []
    ): array {
        $startTime = microtime(true);
        $tier = $this->determineTier($workspaceId);
        $limits = $this->getLimitsForTier($tier);
        $context['tier'] = $tier;
        $context['connection'] = $connection;

        try {
            // Set up the connection with timeout
            $db = $this->getConnection($connection);
            $this->applyTimeout($db, $limits['timeout_seconds']);

            // Execute the query
            $results = $db->select($query);
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);
            $totalRows = count($results);

            // Check result size and truncate if necessary
            $truncated = false;
            $maxRows = $limits['max_rows'];

            if ($totalRows > $maxRows) {
                $truncated = true;
                $results = array_slice($results, 0, $maxRows);
            }

            // Log the query execution
            if ($truncated) {
                $this->auditService->recordTruncated(
                    query: $query,
                    bindings: [],
                    durationMs: $durationMs,
                    returnedRows: $totalRows,
                    maxRows: $maxRows,
                    workspaceId: $workspaceId,
                    userId: $userId,
                    userIp: $userIp,
                    context: $context
                );
            } else {
                $this->auditService->recordSuccess(
                    query: $query,
                    bindings: [],
                    durationMs: $durationMs,
                    rowCount: $totalRows,
                    workspaceId: $workspaceId,
                    userId: $userId,
                    userIp: $userIp,
                    context: $context
                );
            }

            // Build response with metadata
            return [
                'data' => $results,
                'meta' => [
                    'rows_returned' => count($results),
                    'rows_total' => $truncated ? "{$totalRows}+" : $totalRows,
                    'truncated' => $truncated,
                    'max_rows' => $maxRows,
                    'tier' => $tier,
                    'duration_ms' => $durationMs,
                    'warning' => $truncated
                        ? "Results truncated to {$maxRows} rows (tier limit: {$tier}). Add more specific filters to reduce result size."
                        : null,
                ],
            ];
        } catch (\PDOException $e) {
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            // Check if this is a timeout error
            if ($this->isTimeoutError($e)) {
                $this->auditService->recordTimeout(
                    query: $query,
                    bindings: [],
                    timeoutSeconds: $limits['timeout_seconds'],
                    workspaceId: $workspaceId,
                    userId: $userId,
                    userIp: $userIp,
                    context: $context
                );

                throw QueryTimeoutException::exceeded($query, $limits['timeout_seconds']);
            }

            // Log general errors
            $this->auditService->recordError(
                query: $query,
                bindings: [],
                errorMessage: $e->getMessage(),
                durationMs: $durationMs,
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: $context
            );

            throw $e;
        } catch (\Exception $e) {
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            $this->auditService->recordError(
                query: $query,
                bindings: [],
                errorMessage: $e->getMessage(),
                durationMs: $durationMs,
                workspaceId: $workspaceId,
                userId: $userId,
                userIp: $userIp,
                context: $context
            );

            throw $e;
        }
    }

    /**
     * Get the effective limits for a tier.
     *
     * @return array{max_rows: int, timeout_seconds: int}
     */
    public function getLimitsForTier(string $tier): array
    {
        $configuredLimits = Config::get('mcp.database.tier_limits', []);
        $defaultLimits = self::DEFAULT_TIER_LIMITS[$tier] ?? self::DEFAULT_TIER_LIMITS['free'];

        return [
            'max_rows' => $configuredLimits[$tier]['max_rows'] ?? $defaultLimits['max_rows'],
            'timeout_seconds' => $configuredLimits[$tier]['timeout_seconds'] ?? $defaultLimits['timeout_seconds'],
        ];
    }

    /**
     * Get available tiers and their limits.
     *
     * @return array<string, array{max_rows: int, timeout_seconds: int}>
     */
    public function getAvailableTiers(): array
    {
        $tiers = [];

        foreach (array_keys(self::DEFAULT_TIER_LIMITS) as $tier) {
            $tiers[$tier] = $this->getLimitsForTier($tier);
        }

        return $tiers;
    }

    /**
     * Determine the tier for a workspace.
     */
    protected function determineTier(?int $workspaceId): string
    {
        if ($workspaceId === null) {
            return Config::get('mcp.database.default_tier', 'free');
        }

        // Check entitlements if service is available
        if ($this->entitlementService !== null) {
            try {
                $workspace = \Core\Tenant\Models\Workspace::find($workspaceId);

                if ($workspace) {
                    // Check for custom max_rows entitlement
                    $maxRowsResult = $this->entitlementService->can($workspace, self::FEATURE_MAX_ROWS);

                    if ($maxRowsResult->isAllowed() && $maxRowsResult->limit !== null) {
                        // Map the limit to a tier
                        return $this->mapLimitToTier($maxRowsResult->limit);
                    }
                }
            } catch (\Exception $e) {
                // Fall back to default tier on error
                report($e);
            }
        }

        return Config::get('mcp.database.default_tier', 'free');
    }

    /**
     * Map a row limit to the corresponding tier.
     */
    protected function mapLimitToTier(int $limit): string
    {
        foreach (self::DEFAULT_TIER_LIMITS as $tier => $limits) {
            if ($limits['max_rows'] >= $limit) {
                return $tier;
            }
        }

        return 'unlimited';
    }

    /**
     * Get the database connection.
     */
    protected function getConnection(?string $connection): Connection
    {
        return DB::connection($connection);
    }

    /**
     * Apply timeout to the database connection.
     */
    protected function applyTimeout(Connection $connection, int $timeoutSeconds): void
    {
        $driver = $connection->getDriverName();

        try {
            $pdo = $connection->getPdo();

            switch ($driver) {
                case 'mysql':
                case 'mariadb':
                    // MySQL/MariaDB: Use session variable for max execution time
                    $timeoutMs = $timeoutSeconds * 1000;
                    $statement = $pdo->prepare('SET SESSION max_execution_time = ?');
                    $statement->execute([$timeoutMs]);
                    break;

                case 'pgsql':
                    // PostgreSQL: Use statement_timeout
                    $timeoutMs = $timeoutSeconds * 1000;
                    $statement = $pdo->prepare('SET statement_timeout = ?');
                    $statement->execute([$timeoutMs]);
                    break;

                case 'sqlite':
                    // SQLite: Use busy_timeout (in milliseconds)
                    $timeoutMs = $timeoutSeconds * 1000;
                    $pdo->setAttribute(PDO::ATTR_TIMEOUT, $timeoutSeconds);
                    break;

                default:
                    // Use PDO timeout as fallback
                    $pdo->setAttribute(PDO::ATTR_TIMEOUT, $timeoutSeconds);
                    break;
            }
        } catch (\Exception $e) {
            // Log but don't fail - timeout is a safety measure
            report($e);
        }
    }

    /**
     * Check if an exception indicates a timeout.
     */
    protected function isTimeoutError(\PDOException $e): bool
    {
        $message = strtolower($e->getMessage());
        $code = $e->getCode();

        // MySQL timeout indicators
        if (str_contains($message, 'query execution was interrupted')) {
            return true;
        }

        if (str_contains($message, 'max_execution_time exceeded')) {
            return true;
        }

        // PostgreSQL timeout indicators
        if (str_contains($message, 'statement timeout')) {
            return true;
        }

        if (str_contains($message, 'canceling statement due to statement timeout')) {
            return true;
        }

        // SQLite timeout indicators
        if (str_contains($message, 'database is locked')) {
            return true;
        }

        // Generic timeout indicators
        if (str_contains($message, 'timeout')) {
            return true;
        }

        // Check SQLSTATE codes
        if ($code === 'HY000' && str_contains($message, 'execution time')) {
            return true;
        }

        return false;
    }
}
