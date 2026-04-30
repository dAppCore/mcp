<?php

declare(strict_types=1);

namespace Core\Mcp\Console\Commands;

use Illuminate\Console\Command;
use Mod\Mcp\Models\McpApiRequest;
use Mod\Mcp\Models\McpToolCall;
use Mod\Mcp\Models\McpToolCallStat;

/**
 * Cleanup old MCP tool call logs and API request logs.
 *
 * Prunes records older than the configured retention period to prevent
 * unbounded table growth. Aggregated statistics are retained longer
 * than detailed logs.
 */
class CleanupToolCallLogsCommand extends Command
{
    protected $signature = 'mcp:cleanup-logs
                            {--days= : Override the default retention period for detailed logs}
                            {--stats-days= : Override the default retention period for statistics}
                            {--dry-run : Show what would be deleted without actually deleting}';

    protected $description = 'Clean up old MCP tool call logs and API request logs';

    public function handle(): int
    {
        $dryRun = $this->option('dry-run');
        $logRetentionDays = (int) ($this->option('days') ?? config('mcp.log_retention.days', 90));
        $statsRetentionDays = (int) ($this->option('stats-days') ?? config('mcp.log_retention.stats_days', 365));

        $this->info('MCP Log Cleanup'.($dryRun ? ' (DRY RUN)' : ''));
        $this->line('');
        $this->line("Detailed logs retention: {$logRetentionDays} days");
        $this->line("Statistics retention: {$statsRetentionDays} days");
        $this->line('');

        $logsCutoff = now()->subDays($logRetentionDays);
        $statsCutoff = now()->subDays($statsRetentionDays);

        // Clean up tool call logs
        $toolCallsCount = McpToolCall::where('created_at', '<', $logsCutoff)->count();
        if ($toolCallsCount > 0) {
            if ($dryRun) {
                $this->line("Would delete {$toolCallsCount} tool call log(s) older than {$logsCutoff->toDateString()}");
            } else {
                // Delete in chunks to avoid memory issues and lock contention
                $deleted = $this->deleteInChunks(McpToolCall::class, 'created_at', $logsCutoff);
                $this->info("Deleted {$deleted} tool call log(s)");
            }
        } else {
            $this->line('No tool call logs to clean up');
        }

        // Clean up API request logs
        $apiRequestsCount = McpApiRequest::where('created_at', '<', $logsCutoff)->count();
        if ($apiRequestsCount > 0) {
            if ($dryRun) {
                $this->line("Would delete {$apiRequestsCount} API request log(s) older than {$logsCutoff->toDateString()}");
            } else {
                $deleted = $this->deleteInChunks(McpApiRequest::class, 'created_at', $logsCutoff);
                $this->info("Deleted {$deleted} API request log(s)");
            }
        } else {
            $this->line('No API request logs to clean up');
        }

        // Clean up aggregated statistics (longer retention)
        $statsCount = McpToolCallStat::where('date', '<', $statsCutoff->toDateString())->count();
        if ($statsCount > 0) {
            if ($dryRun) {
                $this->line("Would delete {$statsCount} tool call stat(s) older than {$statsCutoff->toDateString()}");
            } else {
                $deleted = McpToolCallStat::where('date', '<', $statsCutoff->toDateString())->delete();
                $this->info("Deleted {$deleted} tool call stat(s)");
            }
        } else {
            $this->line('No tool call stats to clean up');
        }

        $this->line('');
        $this->info('Cleanup complete.');

        return self::SUCCESS;
    }

    /**
     * Delete records in chunks to avoid memory issues.
     */
    protected function deleteInChunks(string $model, string $column, \DateTimeInterface $cutoff, int $chunkSize = 1000): int
    {
        $totalDeleted = 0;

        do {
            $deleted = $model::where($column, '<', $cutoff)
                ->limit($chunkSize)
                ->delete();

            $totalDeleted += $deleted;

            // Small pause to reduce database pressure
            if ($deleted > 0) {
                usleep(10000); // 10ms
            }
        } while ($deleted > 0);

        return $totalDeleted;
    }
}
