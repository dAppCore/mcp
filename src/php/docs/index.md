# MCP Package

The MCP (Model Context Protocol) package provides AI agent tools for database queries, commerce operations, and workspace management with built-in security and quota enforcement.

## Installation

```bash
composer require host-uk/core-mcp
```

## Quick Start

```php
<?php

namespace Mod\Blog;

use Core\Events\McpToolsRegistering;

class Boot
{
    public static array $listens = [
        McpToolsRegistering::class => 'onMcpTools',
    ];

    public function onMcpTools(McpToolsRegistering $event): void
    {
        $event->tool('blog:create-post', Tools\CreatePostTool::class);
        $event->tool('blog:list-posts', Tools\ListPostsTool::class);
    }
}
```

## Key Features

### Database Tools

- **[Query Database](/packages/mcp/query-database)** - SQL query execution with validation and security
- **[SQL Validation](/packages/mcp/security#sql-validation)** - Prevent destructive queries and SQL injection
- **[EXPLAIN Plans](/packages/mcp/query-database#explain)** - Query optimization analysis

### Commerce Tools

- **[Get Billing Status](/packages/mcp/commerce#billing)** - Current billing and subscription status
- **[List Invoices](/packages/mcp/commerce#invoices)** - Invoice history and details
- **[Upgrade Plan](/packages/mcp/commerce#upgrades)** - Tier upgrades with entitlement validation

### Workspace Tools

- **[Workspace Context](/packages/mcp/workspace)** - Automatic workspace/namespace resolution
- **[Quota Enforcement](/packages/mcp/quotas)** - Tool usage limits and monitoring
- **[Tool Analytics](/packages/mcp/analytics)** - Usage tracking and statistics

### Developer Tools

- **[Tool Discovery](/packages/mcp/tools#discovery)** - Automatic tool registration
- **[Dependency Management](/packages/mcp/tools#dependencies)** - Tool dependency resolution
- **[Error Handling](/packages/mcp/tools#errors)** - Consistent error responses

## Creating Tools

### Basic Tool

```php
<?php

namespace Mod\Blog\Tools;

use Core\Mcp\Tools\BaseTool;

class ListPostsTool extends BaseTool
{
    public function getName(): string
    {
        return 'blog:list-posts';
    }

    public function getDescription(): string
    {
        return 'List all blog posts with optional filters';
    }

    public function getParameters(): array
    {
        return [
            'status' => [
                'type' => 'string',
                'description' => 'Filter by status',
                'enum' => ['published', 'draft'],
                'required' => false,
            ],
            'limit' => [
                'type' => 'integer',
                'description' => 'Number of posts to return',
                'default' => 10,
                'required' => false,
            ],
        ];
    }

    public function execute(array $params): array
    {
        $query = Post::query();

        if (isset($params['status'])) {
            $query->where('status', $params['status']);
        }

        $posts = $query->limit($params['limit'] ?? 10)->get();

        return [
            'posts' => $posts->map(fn ($post) => [
                'id' => $post->id,
                'title' => $post->title,
                'slug' => $post->slug,
                'status' => $post->status,
            ])->toArray(),
        ];
    }
}
```

### Tool with Workspace Context

```php
<?php

namespace Mod\Blog\Tools;

use Core\Mcp\Tools\BaseTool;
use Core\Mcp\Tools\Concerns\RequiresWorkspaceContext;

class CreatePostTool extends BaseTool
{
    use RequiresWorkspaceContext;

    public function getName(): string
    {
        return 'blog:create-post';
    }

    public function execute(array $params): array
    {
        // Workspace context automatically validated
        $workspace = $this->getWorkspaceContext();

        $post = Post::create([
            'title' => $params['title'],
            'content' => $params['content'],
            'workspace_id' => $workspace->id,
        ]);

        return [
            'success' => true,
            'post_id' => $post->id,
        ];
    }
}
```

### Tool with Dependencies

```php
<?php

namespace Mod\Blog\Tools;

use Core\Mcp\Tools\BaseTool;
use Core\Mcp\Dependencies\HasDependencies;
use Core\Mcp\Dependencies\ToolDependency;

class ImportPostsTool extends BaseTool
{
    use HasDependencies;

    public function getDependencies(): array
    {
        return [
            new ToolDependency('blog:list-posts', DependencyType::REQUIRED),
            new ToolDependency('media:upload', DependencyType::OPTIONAL),
        ];
    }

    public function execute(array $params): array
    {
        // Dependencies automatically validated
        // ...
    }
}
```

## Query Database Tool

Execute SQL queries with built-in security:

```php
use Core\Mcp\Tools\QueryDatabase;

$tool = new QueryDatabase();

$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE status = ?',
    'bindings' => ['published'],
    'connection' => 'mysql',
]);

// Returns:
// [
//   'rows' => [...],
//   'count' => 10,
//   'execution_time_ms' => 5.23
// ]
```

### Security Features

- **Whitelist-based validation** - Only SELECT queries allowed by default
- **No destructive operations** - DROP, TRUNCATE, DELETE blocked
- **Binding enforcement** - Prevents SQL injection
- **Connection validation** - Only allowed connections accessible
- **EXPLAIN analysis** - Query optimization insights

[Learn more about SQL Security →](/packages/mcp/security)

## Quota System

Enforce tool usage limits per workspace:

```php
// config/mcp.php
'quotas' => [
    'enabled' => true,
    'limits' => [
        'free' => ['calls' => 100, 'per' => 'day'],
        'pro' => ['calls' => 1000, 'per' => 'day'],
        'business' => ['calls' => 10000, 'per' => 'day'],
        'enterprise' => ['calls' => null], // Unlimited
    ],
],
```

Check quota before execution:

```php
use Core\Mcp\Services\McpQuotaService;

$quotaService = app(McpQuotaService::class);

if (!$quotaService->canExecute($workspace, 'blog:create-post')) {
    throw new QuotaExceededException('Daily tool quota exceeded');
}

$quotaService->recordExecution($workspace, 'blog:create-post');
```

[Learn more about Quotas →](/packages/mcp/quotas)

## Tool Analytics

Track tool usage and performance:

```php
use Core\Mcp\Services\ToolAnalyticsService;

$analytics = app(ToolAnalyticsService::class);

// Get tool stats
$stats = $analytics->getToolStats('blog:create-post', period: 'week');
// Returns: ToolStats with executions, errors, avg_duration_ms

// Get workspace usage
$usage = $analytics->getWorkspaceUsage($workspace, period: 'month');

// Get most used tools
$topTools = $analytics->getTopTools(limit: 10, period: 'week');
```

[Learn more about Analytics →](/packages/mcp/analytics)

## Configuration

```php
// config/mcp.php
return [
    'enabled' => true,

    'tools' => [
        'auto_discover' => true,
        'cache_enabled' => true,
    ],

    'query_database' => [
        'allowed_connections' => ['mysql', 'pgsql'],
        'forbidden_keywords' => [
            'DROP', 'TRUNCATE', 'DELETE', 'UPDATE', 'INSERT',
            'ALTER', 'CREATE', 'GRANT', 'REVOKE',
        ],
        'max_execution_time' => 5000, // ms
        'enable_explain' => true,
    ],

    'quotas' => [
        'enabled' => true,
        'limits' => [
            'free' => ['calls' => 100, 'per' => 'day'],
            'pro' => ['calls' => 1000, 'per' => 'day'],
            'business' => ['calls' => 10000, 'per' => 'day'],
            'enterprise' => ['calls' => null],
        ],
    ],

    'analytics' => [
        'enabled' => true,
        'retention_days' => 90,
    ],
];
```

## Middleware

```php
use Core\Mcp\Middleware\ValidateWorkspaceContext;
use Core\Mcp\Middleware\CheckMcpQuota;
use Core\Mcp\Middleware\ValidateToolDependencies;

Route::middleware([
    ValidateWorkspaceContext::class,
    CheckMcpQuota::class,
    ValidateToolDependencies::class,
])->group(function () {
    // MCP tool routes
});
```

## Best Practices

### 1. Use Workspace Context

```php
// ✅ Good - workspace aware
class CreatePostTool extends BaseTool
{
    use RequiresWorkspaceContext;
}

// ❌ Bad - no workspace context
class CreatePostTool extends BaseTool
{
    public function execute(array $params): array
    {
        $post = Post::create($params); // No workspace_id!
    }
}
```

### 2. Validate Parameters

```php
// ✅ Good - strict validation
public function getParameters(): array
{
    return [
        'title' => [
            'type' => 'string',
            'required' => true,
            'maxLength' => 255,
        ],
    ];
}
```

### 3. Handle Errors Gracefully

```php
// ✅ Good - clear error messages
public function execute(array $params): array
{
    try {
        return ['success' => true, 'data' => $result];
    } catch (\Exception $e) {
        return [
            'success' => false,
            'error' => $e->getMessage(),
            'code' => 'TOOL_EXECUTION_FAILED',
        ];
    }
}
```

### 4. Document Tools Well

```php
// ✅ Good - comprehensive description
public function getDescription(): string
{
    return 'Create a new blog post with title, content, and optional metadata. '
         . 'Requires workspace context. Validates entitlements before creation.';
}
```

## Testing

```php
<?php

namespace Tests\Feature\Mcp;

use Tests\TestCase;
use Mod\Blog\Tools\ListPostsTool;

class ListPostsToolTest extends TestCase
{
    public function test_lists_posts(): void
    {
        Post::factory()->count(5)->create(['status' => 'published']);

        $tool = new ListPostsTool();

        $result = $tool->execute([
            'status' => 'published',
            'limit' => 10,
        ]);

        $this->assertArrayHasKey('posts', $result);
        $this->assertCount(5, $result['posts']);
    }
}
```

## Learn More

- [Query Database →](/packages/mcp/query-database)
- [SQL Security →](/packages/mcp/security)
- [Workspace Context →](/packages/mcp/workspace)
- [Tool Analytics →](/packages/mcp/analytics)
- [Quota System →](/packages/mcp/quotas)
