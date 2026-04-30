<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Models\McpAuditLog;
use Core\Mcp\Models\McpSensitiveTool;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Log;

/**
 * Audit Log Service - records immutable tool execution logs with hash chain.
 *
 * Provides tamper-evident logging for MCP tool calls, supporting
 * compliance requirements and forensic analysis.
 */
class AuditLogService
{
    /**
     * Cache key for sensitive tool list.
     */
    protected const SENSITIVE_TOOLS_CACHE_KEY = 'mcp:audit:sensitive_tools';

    /**
     * Cache TTL for sensitive tools (5 minutes).
     */
    protected const SENSITIVE_TOOLS_CACHE_TTL = 300;

    /**
     * Default fields to always redact.
     */
    protected array $defaultRedactFields = [
        'password',
        'secret',
        'token',
        'api_key',
        'apiKey',
        'access_token',
        'refresh_token',
        'private_key',
        'credit_card',
        'card_number',
        'cvv',
        'ssn',
    ];

    /**
     * Record a tool execution in the audit log.
     */
    public function record(
        string $serverId,
        string $toolName,
        array $inputParams = [],
        ?array $outputSummary = null,
        bool $success = true,
        ?int $durationMs = null,
        ?string $errorCode = null,
        ?string $errorMessage = null,
        ?string $sessionId = null,
        ?int $workspaceId = null,
        ?string $actorType = null,
        ?int $actorId = null,
        ?string $actorIp = null,
        ?string $agentType = null,
        ?string $planSlug = null
    ): McpAuditLog {
        return DB::transaction(function () use (
            $serverId,
            $toolName,
            $inputParams,
            $outputSummary,
            $success,
            $durationMs,
            $errorCode,
            $errorMessage,
            $sessionId,
            $workspaceId,
            $actorType,
            $actorId,
            $actorIp,
            $agentType,
            $planSlug
        ) {
            // Get sensitivity info for this tool
            $sensitivityInfo = $this->getSensitivityInfo($toolName);
            $isSensitive = $sensitivityInfo !== null;
            $sensitivityReason = $sensitivityInfo['reason'] ?? null;
            $redactFields = $sensitivityInfo['redact_fields'] ?? [];

            // Redact sensitive fields from input
            $redactedInput = $this->redactFields($inputParams, $redactFields);

            // Redact output if it contains sensitive data
            $redactedOutput = $outputSummary ? $this->redactFields($outputSummary, $redactFields) : null;

            // Get the previous entry's hash for chain linking
            $previousEntry = McpAuditLog::orderByDesc('id')->first();
            $previousHash = $previousEntry?->entry_hash;

            // Create the audit log entry
            $auditLog = new McpAuditLog([
                'server_id' => $serverId,
                'tool_name' => $toolName,
                'workspace_id' => $workspaceId,
                'session_id' => $sessionId,
                'input_params' => $redactedInput,
                'output_summary' => $redactedOutput,
                'success' => $success,
                'duration_ms' => $durationMs,
                'error_code' => $errorCode,
                'error_message' => $errorMessage,
                'actor_type' => $actorType,
                'actor_id' => $actorId,
                'actor_ip' => $actorIp,
                'is_sensitive' => $isSensitive,
                'sensitivity_reason' => $sensitivityReason,
                'previous_hash' => $previousHash,
                'agent_type' => $agentType,
                'plan_slug' => $planSlug,
            ]);

            $auditLog->save();

            // Compute and store the entry hash
            $auditLog->entry_hash = $auditLog->computeHash();
            $auditLog->saveQuietly(); // Bypass updating event to allow hash update

            return $auditLog;
        });
    }

    /**
     * Verify the integrity of the entire audit log chain.
     *
     * @return array{valid: bool, total: int, verified: int, issues: array}
     */
    public function verifyChain(?int $fromId = null, ?int $toId = null): array
    {
        $query = McpAuditLog::orderBy('id');

        if ($fromId !== null) {
            $query->where('id', '>=', $fromId);
        }

        if ($toId !== null) {
            $query->where('id', '<=', $toId);
        }

        $issues = [];
        $verified = 0;
        $previousHash = null;
        $isFirst = true;

        // If starting from a specific ID, get the previous entry's hash
        if ($fromId !== null && $fromId > 1) {
            $previousEntry = McpAuditLog::where('id', '<', $fromId)
                ->orderByDesc('id')
                ->first();
            $previousHash = $previousEntry?->entry_hash;
            $isFirst = false;
        }

        $total = $query->count();

        // Process in chunks to avoid memory issues
        $query->chunk(1000, function ($entries) use (&$issues, &$verified, &$previousHash, &$isFirst) {
            foreach ($entries as $entry) {
                // Verify hash
                if (! $entry->verifyHash()) {
                    $issues[] = [
                        'id' => $entry->id,
                        'type' => 'hash_mismatch',
                        'message' => "Entry #{$entry->id}: Hash mismatch - data may have been tampered",
                        'expected' => $entry->computeHash(),
                        'actual' => $entry->entry_hash,
                    ];
                }

                // Verify chain link
                if ($isFirst) {
                    if ($entry->previous_hash !== null) {
                        $issues[] = [
                            'id' => $entry->id,
                            'type' => 'chain_break',
                            'message' => "Entry #{$entry->id}: First entry should have null previous_hash",
                        ];
                    }
                    $isFirst = false;
                } else {
                    if ($entry->previous_hash !== $previousHash) {
                        $issues[] = [
                            'id' => $entry->id,
                            'type' => 'chain_break',
                            'message' => "Entry #{$entry->id}: Chain link broken",
                            'expected' => $previousHash,
                            'actual' => $entry->previous_hash,
                        ];
                    }
                }

                $previousHash = $entry->entry_hash;
                $verified++;
            }
        });

        return [
            'valid' => empty($issues),
            'total' => $total,
            'verified' => $verified,
            'issues' => $issues,
        ];
    }

    /**
     * Get audit logs for export.
     */
    public function export(
        ?int $workspaceId = null,
        ?Carbon $from = null,
        ?Carbon $to = null,
        ?string $toolName = null,
        bool $sensitiveOnly = false
    ): Collection {
        $query = McpAuditLog::orderBy('id');

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        if ($from !== null) {
            $query->where('created_at', '>=', $from);
        }

        if ($to !== null) {
            $query->where('created_at', '<=', $to);
        }

        if ($toolName !== null) {
            $query->where('tool_name', $toolName);
        }

        if ($sensitiveOnly) {
            $query->where('is_sensitive', true);
        }

        return $query->get()->map(fn ($entry) => $entry->toExportArray());
    }

    /**
     * Export to CSV format.
     */
    public function exportToCsv(
        ?int $workspaceId = null,
        ?Carbon $from = null,
        ?Carbon $to = null,
        ?string $toolName = null,
        bool $sensitiveOnly = false
    ): string {
        $data = $this->export($workspaceId, $from, $to, $toolName, $sensitiveOnly);

        if ($data->isEmpty()) {
            return '';
        }

        $headers = array_keys($data->first());
        $output = fopen('php://temp', 'r+');

        fputcsv($output, $headers);

        foreach ($data as $row) {
            fputcsv($output, array_values($row));
        }

        rewind($output);
        $csv = stream_get_contents($output);
        fclose($output);

        return $csv;
    }

    /**
     * Export to JSON format.
     */
    public function exportToJson(
        ?int $workspaceId = null,
        ?Carbon $from = null,
        ?Carbon $to = null,
        ?string $toolName = null,
        bool $sensitiveOnly = false
    ): string {
        $data = $this->export($workspaceId, $from, $to, $toolName, $sensitiveOnly);

        // Include integrity verification in export
        $verification = $this->verifyChain();

        return json_encode([
            'exported_at' => now()->toIso8601String(),
            'integrity' => [
                'valid' => $verification['valid'],
                'total_entries' => $verification['total'],
                'verified' => $verification['verified'],
                'issues_count' => count($verification['issues']),
            ],
            'filters' => [
                'workspace_id' => $workspaceId,
                'from' => $from?->toIso8601String(),
                'to' => $to?->toIso8601String(),
                'tool_name' => $toolName,
                'sensitive_only' => $sensitiveOnly,
            ],
            'entries' => $data->toArray(),
        ], JSON_PRETTY_PRINT | JSON_THROW_ON_ERROR);
    }

    /**
     * Get statistics for the audit log.
     */
    public function getStats(?int $workspaceId = null, ?int $days = 30): array
    {
        $query = McpAuditLog::query();

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        if ($days !== null) {
            $query->where('created_at', '>=', now()->subDays($days));
        }

        $total = $query->count();
        $successful = (clone $query)->where('success', true)->count();
        $failed = (clone $query)->where('success', false)->count();
        $sensitive = (clone $query)->where('is_sensitive', true)->count();

        $topTools = (clone $query)
            ->select('tool_name', DB::raw('COUNT(*) as count'))
            ->groupBy('tool_name')
            ->orderByDesc('count')
            ->limit(10)
            ->pluck('count', 'tool_name')
            ->toArray();

        $dailyCounts = (clone $query)
            ->select(DB::raw('DATE(created_at) as date'), DB::raw('COUNT(*) as count'))
            ->groupBy('date')
            ->orderBy('date')
            ->limit($days ?? 30)
            ->pluck('count', 'date')
            ->toArray();

        return [
            'total' => $total,
            'successful' => $successful,
            'failed' => $failed,
            'success_rate' => $total > 0 ? round(($successful / $total) * 100, 2) : 0,
            'sensitive_calls' => $sensitive,
            'top_tools' => $topTools,
            'daily_counts' => $dailyCounts,
        ];
    }

    /**
     * Register a sensitive tool.
     */
    public function registerSensitiveTool(
        string $toolName,
        string $reason,
        array $redactFields = [],
        bool $requireConsent = false
    ): void {
        McpSensitiveTool::register($toolName, $reason, $redactFields, $requireConsent);
        $this->clearSensitiveToolsCache();
    }

    /**
     * Unregister a sensitive tool.
     */
    public function unregisterSensitiveTool(string $toolName): bool
    {
        $result = McpSensitiveTool::unregister($toolName);
        $this->clearSensitiveToolsCache();

        return $result;
    }

    /**
     * Get all registered sensitive tools.
     */
    public function getSensitiveTools(): Collection
    {
        return McpSensitiveTool::all();
    }

    /**
     * Check if a tool requires explicit consent.
     */
    public function requiresConsent(string $toolName): bool
    {
        $info = $this->getSensitivityInfo($toolName);

        return $info !== null && ($info['require_explicit_consent'] ?? false);
    }

    // -------------------------------------------------------------------------
    // Protected Methods
    // -------------------------------------------------------------------------

    /**
     * Get sensitivity info for a tool (cached).
     */
    protected function getSensitivityInfo(string $toolName): ?array
    {
        $sensitiveTools = Cache::remember(
            self::SENSITIVE_TOOLS_CACHE_KEY,
            self::SENSITIVE_TOOLS_CACHE_TTL,
            fn () => McpSensitiveTool::all()->keyBy('tool_name')->toArray()
        );

        if (! isset($sensitiveTools[$toolName])) {
            return null;
        }

        $tool = $sensitiveTools[$toolName];

        return [
            'is_sensitive' => true,
            'reason' => $tool['reason'],
            'redact_fields' => $tool['redact_fields'] ?? [],
            'require_explicit_consent' => $tool['require_explicit_consent'] ?? false,
        ];
    }

    /**
     * Redact sensitive fields from data.
     */
    protected function redactFields(array $data, array $additionalFields = []): array
    {
        $fieldsToRedact = array_merge($this->defaultRedactFields, $additionalFields);

        return $this->redactRecursive($data, $fieldsToRedact);
    }

    /**
     * Recursively redact fields in nested arrays.
     */
    protected function redactRecursive(array $data, array $fieldsToRedact): array
    {
        foreach ($data as $key => $value) {
            $keyLower = strtolower((string) $key);

            // Check if this key should be redacted
            foreach ($fieldsToRedact as $field) {
                if (str_contains($keyLower, strtolower($field))) {
                    $data[$key] = '[REDACTED]';

                    continue 2;
                }
            }

            // Recurse into nested arrays
            if (is_array($value)) {
                $data[$key] = $this->redactRecursive($value, $fieldsToRedact);
            }
        }

        return $data;
    }

    /**
     * Clear the sensitive tools cache.
     */
    protected function clearSensitiveToolsCache(): void
    {
        Cache::forget(self::SENSITIVE_TOOLS_CACHE_KEY);
    }
}
