<?php

declare(strict_types=1);

namespace Core\Mcp\Console\Commands;

use Core\Mcp\Services\AuditLogService;
use Illuminate\Console\Command;

/**
 * Verify MCP Audit Log Integrity.
 *
 * Checks the hash chain for tamper detection and reports any issues.
 */
class VerifyAuditLogCommand extends Command
{
    /**
     * The name and signature of the console command.
     */
    protected $signature = 'mcp:verify-audit-log
                            {--from= : Start verification from this ID}
                            {--to= : End verification at this ID}
                            {--json : Output results as JSON}';

    /**
     * The console command description.
     */
    protected $description = 'Verify the integrity of the MCP audit log hash chain';

    /**
     * Execute the console command.
     */
    public function handle(AuditLogService $auditLogService): int
    {
        $fromId = $this->option('from') ? (int) $this->option('from') : null;
        $toId = $this->option('to') ? (int) $this->option('to') : null;
        $jsonOutput = $this->option('json');

        if (! $jsonOutput) {
            $this->info('Verifying MCP audit log integrity...');
            $this->newLine();
        }

        $result = $auditLogService->verifyChain($fromId, $toId);

        if ($jsonOutput) {
            $this->line(json_encode($result, JSON_PRETTY_PRINT));

            return $result['valid'] ? self::SUCCESS : self::FAILURE;
        }

        // Display results
        $this->displayResults($result);

        return $result['valid'] ? self::SUCCESS : self::FAILURE;
    }

    /**
     * Display verification results.
     */
    protected function displayResults(array $result): void
    {
        // Summary table
        $this->table(
            ['Metric', 'Value'],
            [
                ['Total Entries', number_format($result['total'])],
                ['Verified', number_format($result['verified'])],
                ['Status', $result['valid'] ? '<fg=green>VALID</>' : '<fg=red>INVALID</>'],
                ['Issues Found', count($result['issues'])],
            ]
        );

        if ($result['valid']) {
            $this->newLine();
            $this->info('Audit log integrity verified successfully.');
            $this->info('The hash chain is intact and no tampering has been detected.');

            return;
        }

        // Display issues
        $this->newLine();
        $this->error('Integrity issues detected!');
        $this->newLine();

        foreach ($result['issues'] as $issue) {
            $this->warn("Entry #{$issue['id']}: {$issue['type']}");
            $this->line("  {$issue['message']}");

            if (isset($issue['expected'])) {
                $this->line("  Expected: {$issue['expected']}");
            }

            if (isset($issue['actual'])) {
                $this->line("  Actual: {$issue['actual']}");
            }

            $this->newLine();
        }

        $this->error('The audit log may have been tampered with. Please investigate immediately.');
    }
}
