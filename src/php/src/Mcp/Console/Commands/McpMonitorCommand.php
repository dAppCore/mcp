<?php

declare(strict_types=1);

namespace Core\Mcp\Console\Commands;

use Illuminate\Console\Command;
use Mod\Mcp\Services\McpMetricsService;
use Mod\Mcp\Services\McpMonitoringService;

/**
 * MCP Monitor Command.
 *
 * Provides CLI access to MCP monitoring features including
 * health checks, metrics export, and alert checking.
 */
class McpMonitorCommand extends Command
{
    protected $signature = 'mcp:monitor
                            {action=status : Action to perform (status, alerts, export, report, prometheus)}
                            {--days=7 : Number of days for report period}
                            {--json : Output as JSON}';

    protected $description = 'Monitor MCP tool performance and health';

    public function handle(McpMetricsService $metrics, McpMonitoringService $monitoring): int
    {
        $action = $this->argument('action');

        return match ($action) {
            'status' => $this->showStatus($monitoring),
            'alerts' => $this->checkAlerts($monitoring),
            'export' => $this->exportMetrics($monitoring),
            'report' => $this->showReport($monitoring),
            'prometheus' => $this->showPrometheus($monitoring),
            default => $this->showHelp(),
        };
    }

    protected function showStatus(McpMonitoringService $monitoring): int
    {
        $health = $monitoring->getHealthStatus();

        if ($this->option('json')) {
            $this->line(json_encode($health, JSON_PRETTY_PRINT));

            return 0;
        }

        $statusColor = match ($health['status']) {
            'healthy' => 'green',
            'degraded' => 'yellow',
            'critical' => 'red',
            default => 'white',
        };

        $this->newLine();
        $this->line("<fg={$statusColor};options=bold>MCP Health Status: ".strtoupper($health['status']).'</>');
        $this->newLine();

        $this->table(
            ['Metric', 'Value'],
            [
                ['Total Calls (24h)', number_format($health['metrics']['total_calls'])],
                ['Success Rate', $health['metrics']['success_rate'].'%'],
                ['Error Rate', $health['metrics']['error_rate'].'%'],
                ['Avg Duration', $health['metrics']['avg_duration_ms'].'ms'],
            ]
        );

        if (count($health['issues']) > 0) {
            $this->newLine();
            $this->warn('Issues Detected:');

            foreach ($health['issues'] as $issue) {
                $icon = $issue['severity'] === 'critical' ? '!!' : '!';
                $this->line("  [{$icon}] {$issue['message']}");
            }
        }

        $this->newLine();
        $this->line('<fg=gray>Checked at: '.$health['checked_at'].'</>');

        return $health['status'] === 'critical' ? 1 : 0;
    }

    protected function checkAlerts(McpMonitoringService $monitoring): int
    {
        $alerts = $monitoring->checkAlerts();

        if ($this->option('json')) {
            $this->line(json_encode($alerts, JSON_PRETTY_PRINT));

            return count($alerts) > 0 ? 1 : 0;
        }

        if (count($alerts) === 0) {
            $this->info('No alerts detected.');

            return 0;
        }

        $this->warn(count($alerts).' alert(s) detected:');
        $this->newLine();

        foreach ($alerts as $alert) {
            $severityColor = $alert['severity'] === 'critical' ? 'red' : 'yellow';
            $this->line("<fg={$severityColor}>[{$alert['severity']}]</> {$alert['message']}");
        }

        return 1;
    }

    protected function exportMetrics(McpMonitoringService $monitoring): int
    {
        $monitoring->exportMetrics();
        $this->info('Metrics exported to monitoring channel.');

        return 0;
    }

    protected function showReport(McpMonitoringService $monitoring): int
    {
        $days = (int) $this->option('days');
        $report = $monitoring->getSummaryReport($days);

        if ($this->option('json')) {
            $this->line(json_encode($report, JSON_PRETTY_PRINT));

            return 0;
        }

        $this->newLine();
        $this->line("<options=bold>MCP Summary Report ({$days} days)</>");
        $this->line("Period: {$report['period']['from']} to {$report['period']['to']}");
        $this->newLine();

        // Overview
        $this->line('<fg=cyan>Overview:</>');
        $this->table(
            ['Metric', 'Value'],
            [
                ['Total Calls', number_format($report['overview']['total_calls'])],
                ['Success Rate', $report['overview']['success_rate'].'%'],
                ['Avg Duration', $report['overview']['avg_duration_ms'].'ms'],
                ['Unique Tools', $report['overview']['unique_tools']],
                ['Unique Servers', $report['overview']['unique_servers']],
            ]
        );

        // Top tools
        if (count($report['top_tools']) > 0) {
            $this->newLine();
            $this->line('<fg=cyan>Top Tools:</>');

            $toolRows = [];
            foreach ($report['top_tools'] as $tool) {
                $toolRows[] = [
                    $tool->tool_name,
                    number_format($tool->total_calls),
                    $tool->success_rate.'%',
                    round($tool->avg_duration ?? 0).'ms',
                ];
            }

            $this->table(['Tool', 'Calls', 'Success Rate', 'Avg Duration'], $toolRows);
        }

        // Anomalies
        if (count($report['anomalies']) > 0) {
            $this->newLine();
            $this->warn('Anomalies Detected:');

            foreach ($report['anomalies'] as $anomaly) {
                $this->line("  - [{$anomaly['tool']}] {$anomaly['message']}");
            }
        }

        $this->newLine();
        $this->line('<fg=gray>Generated: '.$report['generated_at'].'</>');

        return 0;
    }

    protected function showPrometheus(McpMonitoringService $monitoring): int
    {
        $metrics = $monitoring->getPrometheusMetrics();
        $this->line($metrics);

        return 0;
    }

    protected function showHelp(): int
    {
        $this->error('Unknown action. Available actions: status, alerts, export, report, prometheus');

        return 1;
    }
}
