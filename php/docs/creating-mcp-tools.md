# Guide: Creating MCP Tools

This guide covers everything you need to create MCP tools for AI agents, from basic tools to advanced patterns with workspace context, dependencies, and security best practices.

## Overview

MCP (Model Context Protocol) tools allow AI agents to interact with your application. Each tool:

- Has a unique name and description
- Defines input parameters with JSON Schema
- Executes logic and returns structured responses
- Can require workspace context for multi-tenant isolation
- Can declare dependencies on other tools

## Tool Interface

All MCP tools extend `Laravel\Mcp\Server\Tool` and implement two required methods:

```php
<?php

declare(strict_types=1);

namespace Mod\Blog\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

class ListPostsTool extends Tool
{
    protected string $description = 'List all blog posts with optional filters';

    public function handle(Request $request): Response
    {
        // Tool logic here
        $posts = Post::limit(10)->get();

        return Response::text(json_encode($posts->toArray(), JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [
            'status' => $schema->string('Filter by post status'),
            'limit' => $schema->integer('Maximum posts to return')->default(10),
        ];
    }
}
```

### Key Methods

| Method | Purpose |
|--------|---------|
| `$description` | Tool description shown to AI agents |
| `handle(Request)` | Execute the tool and return a Response |
| `schema(JsonSchema)` | Define input parameters |

## Parameter Validation

Define parameters using the `JsonSchema` builder in the `schema()` method:

### String Parameters

```php
public function schema(JsonSchema $schema): array
{
    return [
        // Basic string
        'title' => $schema->string('Post title')->required(),

        // Enum values
        'status' => $schema->string('Post status: draft, published, archived'),

        // With default
        'format' => $schema->string('Output format')->default('json'),
    ];
}
```

### Numeric Parameters

```php
public function schema(JsonSchema $schema): array
{
    return [
        // Integer
        'limit' => $schema->integer('Maximum results')->default(10),

        // Number (float)
        'price' => $schema->number('Product price'),
    ];
}
```

### Boolean Parameters

```php
public function schema(JsonSchema $schema): array
{
    return [
        'include_drafts' => $schema->boolean('Include draft posts')->default(false),
    ];
}
```

### Array Parameters

```php
public function schema(JsonSchema $schema): array
{
    return [
        'tags' => $schema->array('Filter by tags'),
        'ids' => $schema->array('Specific post IDs to fetch'),
    ];
}
```

### Required vs Optional

```php
public function schema(JsonSchema $schema): array
{
    return [
        // Required - AI agent must provide this
        'query' => $schema->string('SQL query to execute')->required(),

        // Optional with default
        'limit' => $schema->integer('Max rows')->default(100),

        // Optional without default
        'status' => $schema->string('Filter status'),
    ];
}
```

### Accessing Parameters

```php
public function handle(Request $request): Response
{
    // Get single parameter
    $query = $request->input('query');

    // Get with default
    $limit = $request->input('limit', 10);

    // Check if parameter exists
    if ($request->has('status')) {
        // ...
    }

    // Get all parameters
    $params = $request->all();
}
```

### Custom Validation

For validation beyond schema types, validate in `handle()`:

```php
public function handle(Request $request): Response
{
    $email = $request->input('email');

    // Custom validation
    if ($email && !filter_var($email, FILTER_VALIDATE_EMAIL)) {
        return Response::text(json_encode([
            'error' => 'Invalid email format',
            'code' => 'VALIDATION_ERROR',
        ]));
    }

    // Validate limit range
    $limit = $request->input('limit', 10);
    if ($limit < 1 || $limit > 100) {
        return Response::text(json_encode([
            'error' => 'Limit must be between 1 and 100',
            'code' => 'VALIDATION_ERROR',
        ]));
    }

    // Continue with tool logic...
}
```

## Workspace Context

For multi-tenant applications, tools must access data scoped to the authenticated workspace. **Never accept workspace ID as a user-supplied parameter** - this prevents cross-tenant data access.

### Using RequiresWorkspaceContext

```php
<?php

declare(strict_types=1);

namespace Mod\Blog\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;
use Mod\Mcp\Tools\Concerns\RequiresWorkspaceContext;

class ListWorkspacePostsTool extends Tool
{
    use RequiresWorkspaceContext;

    protected string $description = 'List posts in your workspace';

    public function handle(Request $request): Response
    {
        // Get workspace from authenticated context (NOT from request params)
        $workspace = $this->getWorkspace();
        $workspaceId = $this->getWorkspaceId();

        $posts = Post::where('workspace_id', $workspaceId)
            ->limit($request->input('limit', 10))
            ->get();

        return Response::text(json_encode([
            'workspace' => $workspace->name,
            'posts' => $posts->toArray(),
        ], JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        // Note: No workspace_id parameter - comes from auth context
        return [
            'limit' => $schema->integer('Maximum posts to return'),
        ];
    }
}
```

### Trait Methods

The `RequiresWorkspaceContext` trait provides:

| Method | Returns | Description |
|--------|---------|-------------|
| `getWorkspaceContext()` | `WorkspaceContext` | Full context object |
| `getWorkspaceId()` | `int` | Workspace ID only |
| `getWorkspace()` | `Workspace` | Workspace model |
| `hasWorkspaceContext()` | `bool` | Check if context available |
| `validateResourceOwnership(int, string)` | `void` | Validate resource belongs to workspace |

### Setting Workspace Context

Workspace context is set by middleware from authentication (API key or user session):

```php
// In middleware or controller
$tool = new ListWorkspacePostsTool();
$tool->setWorkspaceContext(WorkspaceContext::fromWorkspace($workspace));

// Or from ID
$tool->setWorkspaceId($workspaceId);

// Or from workspace model
$tool->setWorkspace($workspace);
```

### Validating Resource Ownership

When accessing specific resources, validate they belong to the workspace:

```php
public function handle(Request $request): Response
{
    $postId = $request->input('post_id');
    $post = Post::findOrFail($postId);

    // Throws RuntimeException if post doesn't belong to workspace
    $this->validateResourceOwnership($post->workspace_id, 'post');

    // Safe to proceed
    return Response::text(json_encode($post->toArray()));
}
```

## Tool Dependencies

Tools can declare dependencies that must be satisfied before execution. This is useful for workflows where tools must be called in a specific order.

### Declaring Dependencies

Implement `HasDependencies` or use `ValidatesDependencies` trait:

```php
<?php

declare(strict_types=1);

namespace Mod\Blog\Tools;

use Core\Mod\Mcp\Dependencies\DependencyType;
use Core\Mod\Mcp\Dependencies\HasDependencies;
use Core\Mod\Mcp\Dependencies\ToolDependency;
use Laravel\Mcp\Server\Tool;

class UpdateTaskTool extends Tool implements HasDependencies
{
    protected string $description = 'Update a task in the current plan';

    public function dependencies(): array
    {
        return [
            // Another tool must be called first
            ToolDependency::toolCalled(
                'plan_create',
                'A plan must be created before updating tasks'
            ),

            // Session state must exist
            ToolDependency::sessionState(
                'active_plan_id',
                'An active plan must be selected'
            ),

            // Context value required
            ToolDependency::contextExists(
                'workspace_id',
                'Workspace context is required'
            ),
        ];
    }

    public function handle(Request $request): Response
    {
        // Dependencies are validated before handle() is called
        // ...
    }
}
```

### Dependency Types

| Type | Use Case |
|------|----------|
| `TOOL_CALLED` | Another tool must be executed in session |
| `SESSION_STATE` | A session variable must exist |
| `CONTEXT_EXISTS` | A context value must be present |
| `ENTITY_EXISTS` | A database entity must exist |
| `CUSTOM` | Custom validation logic |

### Creating Dependencies

```php
// Tool must be called first
ToolDependency::toolCalled('list_tables');

// Session state required
ToolDependency::sessionState('selected_table');

// Context value required
ToolDependency::contextExists('workspace_id');

// Entity must exist
ToolDependency::entityExists('Plan', 'A plan must exist', [
    'id_param' => 'plan_id',
]);

// Custom validation
ToolDependency::custom('billing_active', 'Billing must be active');
```

### Optional Dependencies

Mark dependencies as optional (warns but doesn't block):

```php
public function dependencies(): array
{
    return [
        ToolDependency::toolCalled('cache_warm')
            ->asOptional(), // Soft dependency
    ];
}
```

### Inline Dependency Validation

Use the `ValidatesDependencies` trait for inline validation:

```php
use Core\Mod\Mcp\Tools\Concerns\ValidatesDependencies;

class MyTool extends Tool
{
    use ValidatesDependencies;

    public function handle(Request $request): Response
    {
        $context = ['session_id' => $request->input('session_id')];

        // Throws if dependencies not met
        $this->validateDependencies($context);

        // Or check without throwing
        if (!$this->dependenciesMet($context)) {
            $missing = $this->getMissingDependencies($context);
            return Response::text(json_encode([
                'error' => 'Dependencies not met',
                'missing' => array_map(fn($d) => $d->key, $missing),
            ]));
        }

        // Continue...
    }
}
```

## Registering Tools

Register tools via the `McpToolsRegistering` event in your module:

```php
<?php

namespace Mod\Blog;

use Core\Events\McpToolsRegistering;
use Mod\Blog\Tools\CreatePostTool;
use Mod\Blog\Tools\ListPostsTool;

class Boot
{
    public static array $listens = [
        McpToolsRegistering::class => 'onMcpTools',
    ];

    public function onMcpTools(McpToolsRegistering $event): void
    {
        $event->tool('blog:list-posts', ListPostsTool::class);
        $event->tool('blog:create-post', CreatePostTool::class);
    }
}
```

### Tool Naming Conventions

Use consistent naming:

```php
// Pattern: module:action-resource
'blog:list-posts'      // List resources
'blog:get-post'        // Get single resource
'blog:create-post'     // Create resource
'blog:update-post'     // Update resource
'blog:delete-post'     // Delete resource

// Sub-modules
'commerce:billing:get-status'
'commerce:coupon:create'
```

## Response Formats

### Success Response

```php
return Response::text(json_encode([
    'success' => true,
    'data' => $result,
], JSON_PRETTY_PRINT));
```

### Error Response

```php
return Response::text(json_encode([
    'error' => 'Specific error message',
    'code' => 'ERROR_CODE',
]));
```

### Paginated Response

```php
$posts = Post::paginate($perPage);

return Response::text(json_encode([
    'data' => $posts->items(),
    'pagination' => [
        'current_page' => $posts->currentPage(),
        'last_page' => $posts->lastPage(),
        'per_page' => $posts->perPage(),
        'total' => $posts->total(),
    ],
], JSON_PRETTY_PRINT));
```

### List Response

```php
return Response::text(json_encode([
    'count' => $items->count(),
    'items' => $items->map(fn($item) => [
        'id' => $item->id,
        'name' => $item->name,
    ])->all(),
], JSON_PRETTY_PRINT));
```

## Security Best Practices

### 1. Never Trust User-Supplied IDs for Authorization

```php
// BAD: Using workspace_id from request
public function handle(Request $request): Response
{
    $workspaceId = $request->input('workspace_id'); // Attacker can change this!
    $posts = Post::where('workspace_id', $workspaceId)->get();
}

// GOOD: Using authenticated workspace context
public function handle(Request $request): Response
{
    $workspaceId = $this->getWorkspaceId(); // From auth context
    $posts = Post::where('workspace_id', $workspaceId)->get();
}
```

### 2. Validate Resource Ownership

```php
public function handle(Request $request): Response
{
    $postId = $request->input('post_id');
    $post = Post::findOrFail($postId);

    // Always validate ownership before access
    $this->validateResourceOwnership($post->workspace_id, 'post');

    return Response::text(json_encode($post->toArray()));
}
```

### 3. Sanitize and Limit Input

```php
public function handle(Request $request): Response
{
    // Limit result sets
    $limit = min($request->input('limit', 10), 100);

    // Sanitize string input
    $search = strip_tags($request->input('search', ''));
    $search = substr($search, 0, 255);

    // Validate enum values
    $status = $request->input('status');
    if ($status && !in_array($status, ['draft', 'published', 'archived'])) {
        return Response::text(json_encode(['error' => 'Invalid status']));
    }
}
```

### 4. Log Sensitive Operations

```php
public function handle(Request $request): Response
{
    Log::info('MCP tool executed', [
        'tool' => 'delete-post',
        'workspace_id' => $this->getWorkspaceId(),
        'post_id' => $request->input('post_id'),
        'user' => auth()->id(),
    ]);

    // Perform operation...
}
```

### 5. Use Read-Only Database Connections for Queries

```php
// For query tools, use read-only connection
$connection = config('mcp.database.connection', 'readonly');
$results = DB::connection($connection)->select($query);
```

### 6. Sanitize Error Messages

```php
try {
    // Operation...
} catch (\Exception $e) {
    // Log full error for debugging
    report($e);

    // Return sanitized message to client
    return Response::text(json_encode([
        'error' => 'Operation failed. Please try again.',
        'code' => 'OPERATION_FAILED',
    ]));
}
```

### 7. Implement Rate Limiting

Tools should respect quota limits:

```php
use Core\Mcp\Services\McpQuotaService;

public function handle(Request $request): Response
{
    $quota = app(McpQuotaService::class);
    $workspace = $this->getWorkspace();

    if (!$quota->canExecute($workspace, $this->name())) {
        return Response::text(json_encode([
            'error' => 'Rate limit exceeded',
            'code' => 'QUOTA_EXCEEDED',
        ]));
    }

    // Execute tool...

    $quota->recordExecution($workspace, $this->name());
}
```

## Testing Tools

```php
<?php

namespace Tests\Feature\Mcp;

use Tests\TestCase;
use Mod\Blog\Tools\ListPostsTool;
use Mod\Blog\Models\Post;
use Core\Mod\Tenant\Models\Workspace;
use Mod\Mcp\Context\WorkspaceContext;

class ListPostsToolTest extends TestCase
{
    public function test_lists_posts(): void
    {
        $workspace = Workspace::factory()->create();
        Post::factory()->count(5)->create([
            'workspace_id' => $workspace->id,
        ]);

        $tool = new ListPostsTool();
        $tool->setWorkspaceContext(
            WorkspaceContext::fromWorkspace($workspace)
        );

        $request = new \Laravel\Mcp\Request([
            'limit' => 10,
        ]);

        $response = $tool->handle($request);
        $data = json_decode($response->getContent(), true);

        $this->assertCount(5, $data['posts']);
    }

    public function test_respects_workspace_isolation(): void
    {
        $workspace1 = Workspace::factory()->create();
        $workspace2 = Workspace::factory()->create();

        Post::factory()->count(3)->create(['workspace_id' => $workspace1->id]);
        Post::factory()->count(2)->create(['workspace_id' => $workspace2->id]);

        $tool = new ListPostsTool();
        $tool->setWorkspace($workspace1);

        $request = new \Laravel\Mcp\Request([]);
        $response = $tool->handle($request);
        $data = json_decode($response->getContent(), true);

        // Should only see workspace1's posts
        $this->assertCount(3, $data['posts']);
    }

    public function test_throws_without_workspace_context(): void
    {
        $this->expectException(MissingWorkspaceContextException::class);

        $tool = new ListPostsTool();
        // Not setting workspace context

        $tool->handle(new \Laravel\Mcp\Request([]));
    }
}
```

## Complete Example

Here's a complete tool implementation following all best practices:

```php
<?php

declare(strict_types=1);

namespace Mod\Commerce\Tools;

use Core\Mod\Commerce\Models\Invoice;
use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;
use Mod\Mcp\Tools\Concerns\RequiresWorkspaceContext;

/**
 * List invoices for the authenticated workspace.
 *
 * SECURITY: Uses authenticated workspace context to prevent cross-tenant access.
 */
class ListInvoicesTool extends Tool
{
    use RequiresWorkspaceContext;

    protected string $description = 'List invoices for your workspace with optional status filter';

    public function handle(Request $request): Response
    {
        // Get workspace from auth context (never from request params)
        $workspaceId = $this->getWorkspaceId();

        // Validate and sanitize inputs
        $status = $request->input('status');
        if ($status && !in_array($status, ['paid', 'pending', 'overdue', 'void'])) {
            return Response::text(json_encode([
                'error' => 'Invalid status. Use: paid, pending, overdue, void',
                'code' => 'VALIDATION_ERROR',
            ]));
        }

        $limit = min($request->input('limit', 10), 50);

        // Query with workspace scope
        $query = Invoice::with('order')
            ->where('workspace_id', $workspaceId)
            ->latest();

        if ($status) {
            $query->where('status', $status);
        }

        $invoices = $query->limit($limit)->get();

        return Response::text(json_encode([
            'workspace_id' => $workspaceId,
            'count' => $invoices->count(),
            'invoices' => $invoices->map(fn ($invoice) => [
                'id' => $invoice->id,
                'invoice_number' => $invoice->invoice_number,
                'status' => $invoice->status,
                'total' => (float) $invoice->total,
                'currency' => $invoice->currency,
                'issue_date' => $invoice->issue_date?->toDateString(),
                'due_date' => $invoice->due_date?->toDateString(),
                'is_overdue' => $invoice->isOverdue(),
            ])->all(),
        ], JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [
            'status' => $schema->string('Filter by status: paid, pending, overdue, void'),
            'limit' => $schema->integer('Maximum invoices to return (default 10, max 50)'),
        ];
    }
}
```

## Learn More

- [SQL Security](/packages/mcp/sql-security) - Safe query patterns
- [Workspace Context](/packages/mcp/workspace) - Multi-tenant isolation
- [Tool Analytics](/packages/mcp/analytics) - Usage tracking
- [Quotas](/packages/mcp/quotas) - Rate limiting
