# Usage Quotas

Tier-based rate limiting and usage quotas for MCP tools.

## Overview

The quota system enforces usage limits based on workspace subscription tiers:

**Tiers:**
- **Free:** 60 requests/hour, 500 queries/day
- **Pro:** 600 requests/hour, 10,000 queries/day
- **Enterprise:** Unlimited

## Quota Enforcement

### Middleware

```php
use Core\Mcp\Middleware\CheckMcpQuota;

Route::middleware([CheckMcpQuota::class])
    ->post('/mcp/tools/{tool}', [McpController::class, 'execute']);
```

**Process:**
1. Identifies workspace from context
2. Checks current usage against tier limits
3. Allows or denies request
4. Records usage on success

### Manual Checking

```php
use Core\Mcp\Services\McpQuotaService;

$quota = app(McpQuotaService::class);

// Check if within quota
if (!$quota->withinLimit($workspace)) {
    return response()->json([
        'error' => 'Quota exceeded',
        'message' => 'You have reached your hourly limit',
        'reset_at' => $quota->resetTime($workspace),
    ], 429);
}

// Record usage
$quota->recordUsage($workspace, 'query_database');
```

## Quota Configuration

```php
// config/mcp.php
return [
    'quotas' => [
        'free' => [
            'requests_per_hour' => 60,
            'queries_per_day' => 500,
            'max_query_rows' => 1000,
        ],
        'pro' => [
            'requests_per_hour' => 600,
            'queries_per_day' => 10000,
            'max_query_rows' => 10000,
        ],
        'enterprise' => [
            'requests_per_hour' => null, // Unlimited
            'queries_per_day' => null,
            'max_query_rows' => 100000,
        ],
    ],
];
```

## Usage Tracking

### Current Usage

```php
use Core\Mcp\Services\McpQuotaService;

$quota = app(McpQuotaService::class);

// Get current hour's usage
$hourlyUsage = $quota->getHourlyUsage($workspace);

// Get current day's usage
$dailyUsage = $quota->getDailyUsage($workspace);

// Get usage percentage
$percentage = $quota->usagePercentage($workspace);
```

### Usage Response Headers

```
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1706274000
X-RateLimit-Reset-At: 2026-01-26T13:00:00Z
```

**Implementation:**

```php
use Core\Mcp\Middleware\CheckMcpQuota;

class CheckMcpQuota
{
    public function handle($request, Closure $next)
    {
        $workspace = $request->workspace;
        $quota = app(McpQuotaService::class);

        $response = $next($request);

        // Add quota headers
        $response->headers->set('X-RateLimit-Limit', $quota->getLimit($workspace));
        $response->headers->set('X-RateLimit-Remaining', $quota->getRemaining($workspace));
        $response->headers->set('X-RateLimit-Reset', $quota->resetTime($workspace)->timestamp);

        return $response;
    }
}
```

## Quota Exceeded Response

```json
{
  "error": "quota_exceeded",
  "message": "You have exceeded your hourly request limit",
  "current_usage": 60,
  "limit": 60,
  "reset_at": "2026-01-26T13:00:00Z",
  "upgrade_url": "https://example.com/billing/upgrade"
}
```

**HTTP Status:** 429 Too Many Requests

## Upgrading Tiers

```php
use Mod\Tenant\Models\Workspace;

$workspace = Workspace::find($id);

// Upgrade to Pro
$workspace->update([
    'subscription_tier' => 'pro',
]);

// New limits immediately apply
$quota = app(McpQuotaService::class);
$newLimit = $quota->getLimit($workspace); // 600
```

## Quota Monitoring

### Admin Dashboard

```php
class QuotaUsage extends Component
{
    public function render()
    {
        $quota = app(McpQuotaService::class);

        $workspaces = Workspace::all()->map(function ($workspace) use ($quota) {
            return [
                'name' => $workspace->name,
                'tier' => $workspace->subscription_tier,
                'hourly_usage' => $quota->getHourlyUsage($workspace),
                'hourly_limit' => $quota->getLimit($workspace, 'hourly'),
                'daily_usage' => $quota->getDailyUsage($workspace),
                'daily_limit' => $quota->getLimit($workspace, 'daily'),
            ];
        });

        return view('mcp::admin.quota-usage', compact('workspaces'));
    }
}
```

**View:**

```blade
<x-admin::table>
    <x-slot:header>
        <x-admin::table.th>Workspace</x-admin::table.th>
        <x-admin::table.th>Tier</x-admin::table.th>
        <x-admin::table.th>Hourly Usage</x-admin::table.th>
        <x-admin::table.th>Daily Usage</x-admin::table.th>
    </x-slot:header>

    @foreach($workspaces as $workspace)
        <x-admin::table.tr>
            <x-admin::table.td>{{ $workspace['name'] }}</x-admin::table.td>
            <x-admin::table.td>
                <x-admin::badge :color="$workspace['tier'] === 'enterprise' ? 'purple' : 'blue'">
                    {{ ucfirst($workspace['tier']) }}
                </x-admin::badge>
            </x-admin::table.td>
            <x-admin::table.td>
                {{ $workspace['hourly_usage'] }} / {{ $workspace['hourly_limit'] ?? '∞' }}
                <progress
                    value="{{ $workspace['hourly_usage'] }}"
                    max="{{ $workspace['hourly_limit'] ?? 100 }}"
                ></progress>
            </x-admin::table.td>
            <x-admin::table.td>
                {{ $workspace['daily_usage'] }} / {{ $workspace['daily_limit'] ?? '∞' }}
            </x-admin::table.td>
        </x-admin::table.tr>
    @endforeach
</x-admin::table>
```

### Alerts

Send notifications when nearing limits:

```php
use Core\Mcp\Services\McpQuotaService;

$quota = app(McpQuotaService::class);

$usage = $quota->usagePercentage($workspace);

if ($usage >= 80) {
    // Alert: 80% of quota used
    $workspace->owner->notify(
        new QuotaWarningNotification($workspace, $usage)
    );
}

if ($usage >= 100) {
    // Alert: Quota exceeded
    $workspace->owner->notify(
        new QuotaExceededNotification($workspace)
    );
}
```

## Custom Quotas

Override default quotas for specific workspaces:

```php
use Core\Mcp\Models\McpUsageQuota;

// Set custom quota
McpUsageQuota::create([
    'workspace_id' => $workspace->id,
    'requests_per_hour' => 1000, // Custom limit
    'queries_per_day' => 50000,
    'expires_at' => now()->addMonths(3), // Temporary increase
]);

// Custom quota takes precedence over tier defaults
```

## Resetting Quotas

```bash
# Reset all quotas
php artisan mcp:reset-quotas

# Reset specific workspace
php artisan mcp:reset-quotas --workspace=123

# Reset specific period
php artisan mcp:reset-quotas --period=hourly
```

## Bypass Quotas (Admin)

```php
// Bypass quota enforcement
$result = $tool->execute($params, [
    'bypass_quota' => true, // Requires admin permission
]);
```

**Use cases:**
- Internal tools
- Admin operations
- System maintenance
- Testing

## Testing

```php
use Tests\TestCase;
use Core\Mcp\Services\McpQuotaService;

class QuotaTest extends TestCase
{
    public function test_enforces_hourly_limit(): void
    {
        $workspace = Workspace::factory()->create(['tier' => 'free']);
        $quota = app(McpQuotaService::class);

        // Exhaust quota
        for ($i = 0; $i < 60; $i++) {
            $quota->recordUsage($workspace, 'test');
        }

        $this->assertFalse($quota->withinLimit($workspace));
    }

    public function test_resets_after_hour(): void
    {
        $workspace = Workspace::factory()->create();
        $quota = app(McpQuotaService::class);

        // Use quota
        $quota->recordUsage($workspace, 'test');

        // Travel 1 hour
        $this->travel(1)->hour();

        $this->assertTrue($quota->withinLimit($workspace));
    }

    public function test_enterprise_has_no_limit(): void
    {
        $workspace = Workspace::factory()->create(['tier' => 'enterprise']);
        $quota = app(McpQuotaService::class);

        // Use quota 1000 times
        for ($i = 0; $i < 1000; $i++) {
            $quota->recordUsage($workspace, 'test');
        }

        $this->assertTrue($quota->withinLimit($workspace));
    }
}
```

## Best Practices

### 1. Check Quotas Early

```php
// ✅ Good - check before processing
if (!$quota->withinLimit($workspace)) {
    return response()->json(['error' => 'Quota exceeded'], 429);
}

$result = $tool->execute($params);

// ❌ Bad - check after processing
$result = $tool->execute($params);
if (!$quota->withinLimit($workspace)) {
    // Too late!
}
```

### 2. Provide Clear Feedback

```php
// ✅ Good - helpful error message
return response()->json([
    'error' => 'Quota exceeded',
    'reset_at' => $quota->resetTime($workspace),
    'upgrade_url' => route('billing.upgrade'),
], 429);

// ❌ Bad - generic error
return response()->json(['error' => 'Too many requests'], 429);
```

### 3. Monitor Usage Patterns

```php
// ✅ Good - alert at 80%
if ($usage >= 80) {
    $this->notifyUser();
}

// ❌ Bad - only alert when exhausted
if ($usage >= 100) {
    // User already hit limit
}
```

### 4. Use Appropriate Limits

```php
// ✅ Good - reasonable limits
'free' => ['requests_per_hour' => 60],
'pro' => ['requests_per_hour' => 600],

// ❌ Bad - too restrictive
'free' => ['requests_per_hour' => 5], // Unusable
```

## Learn More

- [Analytics →](/packages/mcp/analytics)
- [Security →](/packages/mcp/security)
- [Multi-Tenancy →](/packages/core/tenancy)
