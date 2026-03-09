<?php

declare(strict_types=1);

namespace Core\Mcp\Console\Commands;

use Core\Mcp\Models\ToolMetric;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;

/**
 * Prune old MCP tool metrics data.
 *
 * Deletes metrics records older than the configured retention period
 * to prevent unbounded database growth.
 */
class PruneMetricsCommand extends Command
{
    protected $signature = 'mcp:prune-metrics
                            {--days= : Override the default retention period (in days)}
                            {--dry-run : Show what would be deleted without actually deleting}';

    protected $description = 'Delete MCP tool metrics older than the retention period';

    public function handle(): int
    {
        $dryRun = $this->option('dry-run');
        $retentionDays = (int) ($this->option('days') ?? config('mcp.analytics.retention_days', 90));

        $this->info('MCP Metrics Pruning'.($dryRun ? ' (DRY RUN)' : ''));
        $this->line('');
        $this->line("Retention period: {$retentionDays} days");
        $this->line('');

        $cutoffDate = now()->subDays($retentionDays)->toDateString();

        // Prune tool metrics
        $metricsCount = ToolMetric::where('date', '<', $cutoffDate)->count();

        if ($metricsCount > 0) {
            if ($dryRun) {
                $this->line("Would delete {$metricsCount} tool metric record(s) older than {$cutoffDate}");
            } else {
                $deleted = $this->deleteInChunks(ToolMetric::class, 'date', $cutoffDate);
                $this->info("Deleted {$deleted} tool metric record(s)");
            }
        } else {
            $this->line('No tool metrics to prune');
        }

        // Prune tool combinations
        $combinationsCount = DB::table('mcp_tool_combinations')
            ->where('date', '<', $cutoffDate)
            ->count();

        if ($combinationsCount > 0) {
            if ($dryRun) {
                $this->line("Would delete {$combinationsCount} tool combination record(s) older than {$cutoffDate}");
            } else {
                $deleted = DB::table('mcp_tool_combinations')
                    ->where('date', '<', $cutoffDate)
                    ->delete();
                $this->info("Deleted {$deleted} tool combination record(s)");
            }
        } else {
            $this->line('No tool combinations to prune');
        }

        $this->line('');
        $this->info('Pruning complete.');

        return self::SUCCESS;
    }

    /**
     * Delete records in chunks to avoid memory issues.
     */
    protected function deleteInChunks(string $model, string $column, string $cutoff, int $chunkSize = 1000): int
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
