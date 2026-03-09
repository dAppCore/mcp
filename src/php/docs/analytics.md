# Tool Analytics

Track MCP tool usage, performance, and patterns with comprehensive analytics.

## Overview

The MCP analytics system provides insights into:
- Tool execution frequency
- Performance metrics
- Error rates
- User patterns
- Workspace usage

## Recording Metrics

### Automatic Tracking

Tool executions are automatically tracked:

```php
use Core\Mcp\Listeners\RecordToolExecution;
use Core\Mcp\Events\ToolExecuted;

// Automatically recorded on tool execution
event(new ToolExecuted(
    tool: 'query_database',
    workspace: $workspace,
    user: $user,
    duration: 5.23,
    success: true
));
```

### Manual Recording

```php
use Core\Mcp\Services\ToolAnalyticsService;

$analytics = app(ToolAnalyticsService::class);

$analytics->record([
    'tool_name' => 'query_database',
    'workspace_id' => $workspace->id,
    'user_id' => $user->id,
    'execution_time_ms' => 5.23,
    'success' => true,
    'error_message' => null,
    'metadata' => [
        'query_rows' => 42,
        'connection' => 'mysql',
    ],
]);
```

## Querying Analytics

### Tool Stats

```php
use Core\Mcp\Services\ToolAnalyticsService;

$analytics = app(ToolAnalyticsService::class);

// Get stats for specific tool
$stats = $analytics->getToolStats('query_database', [
    'workspace_id' => $workspace->id,
    'start_date' => now()->subDays(30),
    'end_date' => now(),
]);
```

**Returns:**

```php
use Core\Mcp\DTO\ToolStats;

$stats = new ToolStats(
    tool_name: 'query_database',
    total_executions: 1234,
    successful_executions: 1200,
    failed_executions: 34,
    avg_execution_time_ms: 5.23,
    p95_execution_time_ms: 12.45,
    p99_execution_time_ms: 24.67,
    error_rate: 2.76, // percentage
);
```

### Most Used Tools

```php
$topTools = $analytics->mostUsedTools([
    'workspace_id' => $workspace->id,
    'limit' => 10,
    'start_date' => now()->subDays(7),
]);

// Returns array:
[
    ['tool_name' => 'query_database', 'count' => 500],
    ['tool_name' => 'list_workspaces', 'count' => 120],
    ['tool_name' => 'get_billing_status', 'count' => 45],
]
```

### Error Analysis

```php
// Get failed executions
$errors = $analytics->getErrors([
    'workspace_id' => $workspace->id,
    'tool_name' => 'query_database',
    'start_date' => now()->subDays(7),
]);

foreach ($errors as $error) {
    echo "Error: {$error->error_message}\n";
    echo "Occurred: {$error->created_at->diffForHumans()}\n";
    echo "User: {$error->user->name}\n";
}
```

### Performance Trends

```php
// Get daily execution counts
$trend = $analytics->dailyTrend([
    'tool_name' => 'query_database',
    'workspace_id' => $workspace->id,
    'days' => 30,
]);

// Returns:
[
    '2026-01-01' => 45,
    '2026-01-02' => 52,
    '2026-01-03' => 48,
    // ...
]
```

## Admin Dashboard

View analytics in admin panel:

```php
<?php

namespace Core\Mcp\View\Modal\Admin;

use Livewire\Component;
use Core\Mcp\Services\ToolAnalyticsService;

class ToolAnalyticsDashboard extends Component
{
    public function render()
    {
        $analytics = app(ToolAnalyticsService::class);

        return view('mcp::admin.analytics.dashboard', [
            'totalExecutions' => $analytics->totalExecutions(),
            'topTools' => $analytics->mostUsedTools(['limit' => 10]),
            'errorRate' => $analytics->errorRate(),
            'avgExecutionTime' => $analytics->averageExecutionTime(),
        ]);
    }
}
```

**View:**

```blade
<x-admin::card>
    <x-slot:header>
        <h3>MCP Tool Analytics</h3>
    </x-slot:header>

    <div class="grid grid-cols-4 gap-4">
        <x-admin::stat
            label="Total Executions"
            :value="$totalExecutions"
            icon="heroicon-o-play-circle"
        />

        <x-admin::stat
            label="Error Rate"
            :value="number_format($errorRate, 2) . '%'"
            icon="heroicon-o-exclamation-triangle"
            :color="$errorRate > 5 ? 'red' : 'green'"
        />

        <x-admin::stat
            label="Avg Execution Time"
            :value="number_format($avgExecutionTime, 2) . 'ms'"
            icon="heroicon-o-clock"
        />

        <x-admin::stat
            label="Active Tools"
            :value="count($topTools)"
            icon="heroicon-o-cube"
        />
    </div>

    <div class="mt-6">
        <h4>Most Used Tools</h4>
        <x-admin::table>
            <x-slot:header>
                <x-admin::table.th>Tool</x-admin::table.th>
                <x-admin::table.th>Executions</x-admin::table.th>
            </x-slot:header>

            @foreach($topTools as $tool)
                <x-admin::table.tr>
                    <x-admin::table.td>{{ $tool['tool_name'] }}</x-admin::table.td>
                    <x-admin::table.td>{{ number_format($tool['count']) }}</x-admin::table.td>
                </x-admin::table.tr>
            @endforeach
        </x-admin::table>
    </div>
</x-admin::card>
```

## Tool Detail View

Detailed analytics for specific tool:

```blade
<x-admin::card>
    <x-slot:header>
        <h3>{{ $toolName }} Analytics</h3>
    </x-slot:header>

    <div class="grid grid-cols-3 gap-4">
        <x-admin::stat
            label="Total Executions"
            :value="$stats->total_executions"
        />

        <x-admin::stat
            label="Success Rate"
            :value="number_format((1 - $stats->error_rate / 100) * 100, 1) . '%'"
            :color="$stats->error_rate < 5 ? 'green' : 'red'"
        />

        <x-admin::stat
            label="P95 Latency"
            :value="number_format($stats->p95_execution_time_ms, 2) . 'ms'"
        />
    </div>

    <div class="mt-6">
        <h4>Performance Trend</h4>
        <canvas id="performance-chart"></canvas>
    </div>

    <div class="mt-6">
        <h4>Recent Errors</h4>
        @foreach($recentErrors as $error)
            <x-admin::alert type="error">
                <strong>{{ $error->created_at->diffForHumans() }}</strong>
                {{ $error->error_message }}
            </x-admin::alert>
        @endforeach
    </div>
</x-admin::card>
```

## Pruning Old Metrics

```bash
# Prune metrics older than 90 days
php artisan mcp:prune-metrics --days=90

# Dry run
php artisan mcp:prune-metrics --days=90 --dry-run
```

**Scheduled Pruning:**

```php
// app/Console/Kernel.php
protected function schedule(Schedule $schedule)
{
    $schedule->command('mcp:prune-metrics --days=90')
        ->daily()
        ->at('02:00');
}
```

## Alerting

Set up alerts for anomalies:

```php
use Core\Mcp\Services\ToolAnalyticsService;

$analytics = app(ToolAnalyticsService::class);

// Check error rate
$errorRate = $analytics->errorRate([
    'tool_name' => 'query_database',
    'start_date' => now()->subHours(1),
]);

if ($errorRate > 10) {
    // Alert: High error rate
    Notification::route('slack', config('slack.webhook'))
        ->notify(new HighErrorRateNotification('query_database', $errorRate));
}

// Check slow executions
$p99 = $analytics->getToolStats('query_database')->p99_execution_time_ms;

if ($p99 > 1000) {
    // Alert: Slow performance
    Notification::route('slack', config('slack.webhook'))
        ->notify(new SlowToolNotification('query_database', $p99));
}
```

## Export Analytics

```php
use Core\Mcp\Services\ToolAnalyticsService;

$analytics = app(ToolAnalyticsService::class);

// Export to CSV
$csv = $analytics->exportToCsv([
    'workspace_id' => $workspace->id,
    'start_date' => now()->subDays(30),
    'end_date' => now(),
]);

return response()->streamDownload(function () use ($csv) {
    echo $csv;
}, 'mcp-analytics.csv');
```

## Best Practices

### 1. Set Retention Policies

```php
// config/mcp.php
return [
    'analytics' => [
        'retention_days' => 90, // Keep 90 days
        'prune_schedule' => 'daily',
    ],
];
```

### 2. Monitor Error Rates

```php
// ✅ Good - alert on high error rate
if ($errorRate > 10) {
    $this->alert('High error rate');
}

// ❌ Bad - ignore errors
// (problems go unnoticed)
```

### 3. Track Performance

```php
// ✅ Good - measure execution time
$start = microtime(true);
$result = $tool->execute($params);
$duration = (microtime(true) - $start) * 1000;

$analytics->record([
    'execution_time_ms' => $duration,
]);
```

### 4. Use Aggregated Queries

```php
// ✅ Good - use analytics service
$stats = $analytics->getToolStats('query_database');

// ❌ Bad - query metrics table directly
$count = ToolMetric::where('tool_name', 'query_database')->count();
```

## Testing

```php
use Tests\TestCase;
use Core\Mcp\Services\ToolAnalyticsService;

class AnalyticsTest extends TestCase
{
    public function test_records_tool_execution(): void
    {
        $analytics = app(ToolAnalyticsService::class);

        $analytics->record([
            'tool_name' => 'test_tool',
            'workspace_id' => 1,
            'success' => true,
        ]);

        $this->assertDatabaseHas('mcp_tool_metrics', [
            'tool_name' => 'test_tool',
            'workspace_id' => 1,
        ]);
    }

    public function test_calculates_error_rate(): void
    {
        $analytics = app(ToolAnalyticsService::class);

        // Record 100 successful, 10 failed
        for ($i = 0; $i < 100; $i++) {
            $analytics->record(['tool_name' => 'test', 'success' => true]);
        }
        for ($i = 0; $i < 10; $i++) {
            $analytics->record(['tool_name' => 'test', 'success' => false]);
        }

        $errorRate = $analytics->errorRate(['tool_name' => 'test']);

        $this->assertEquals(9.09, round($errorRate, 2)); // 10/110 = 9.09%
    }
}
```

## Learn More

- [Quotas →](/packages/mcp/quotas)
- [Creating Tools →](/packages/mcp/tools)
