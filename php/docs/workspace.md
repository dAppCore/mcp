# Workspace Context

Workspace isolation and context resolution for MCP tools.

## Overview

Workspace context ensures that MCP tools operate within the correct workspace boundary, preventing data leaks and unauthorized access.

## Context Resolution

### From Request Headers

```php
use Core\Mcp\Context\WorkspaceContext;

// Resolve from X-Workspace-ID header
$context = WorkspaceContext::fromRequest($request);

// Returns WorkspaceContext with:
// - workspace: Workspace model
// - user: Current user
// - namespace: Current namespace (if applicable)
```

**Request Example:**

```bash
curl -H "Authorization: Bearer sk_live_..." \
     -H "X-Workspace-ID: ws_abc123" \
     https://api.example.com/mcp/query
```

### From API Key

```php
use Mod\Api\Models\ApiKey;

$apiKey = ApiKey::findByKey($providedKey);

// API key is scoped to workspace
$context = WorkspaceContext::fromApiKey($apiKey);
```

### Manual Creation

```php
use Mod\Tenant\Models\Workspace;

$workspace = Workspace::find($id);

$context = new WorkspaceContext(
    workspace: $workspace,
    user: $user,
    namespace: $namespace
);
```

## Requiring Context

### Tool Implementation

```php
<?php

namespace Mod\Blog\Mcp\Tools;

use Core\Mcp\Tools\BaseTool;
use Core\Mcp\Tools\Concerns\RequiresWorkspaceContext;

class ListPosts extends BaseTool
{
    use RequiresWorkspaceContext;

    public function execute(array $params): array
    {
        // Validates workspace context exists
        $this->validateWorkspaceContext();

        // Access workspace
        $workspace = $this->workspaceContext->workspace;

        // Query scoped to workspace
        return Post::where('workspace_id', $workspace->id)
            ->where('status', $params['status'] ?? 'published')
            ->get()
            ->toArray();
    }
}
```

### Middleware

```php
use Core\Mcp\Middleware\ValidateWorkspaceContext;

Route::middleware([ValidateWorkspaceContext::class])
    ->post('/mcp/tools/{tool}', [McpController::class, 'execute']);
```

**Validation:**
- Header `X-Workspace-ID` is present
- Workspace exists
- User has access to workspace
- API key is scoped to workspace

## Automatic Query Scoping

### SELECT Queries

```php
// Query without workspace filter
$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE status = ?',
    'bindings' => ['published'],
]);

// Automatically becomes:
// SELECT * FROM posts
// WHERE status = ?
//   AND workspace_id = ?
// With bindings: ['published', $workspaceId]
```

### BelongsToWorkspace Models

```php
use Core\Mod\Tenant\Concerns\BelongsToWorkspace;

class Post extends Model
{
    use BelongsToWorkspace;

    // Automatically scoped to workspace
}

// All queries automatically filtered:
Post::all(); // Only current workspace's posts
Post::where('status', 'published')->get(); // Scoped
Post::find($id); // Returns null if wrong workspace
```

## Context Properties

### Workspace

```php
$workspace = $context->workspace;

$workspace->id;           // Workspace ID
$workspace->name;         // Workspace name
$workspace->slug;         // URL slug
$workspace->settings;     // Workspace settings
$workspace->subscription; // Subscription plan
```

### User

```php
$user = $context->user;

$user->id;              // User ID
$user->name;            // User name
$user->email;           // User email
$user->workspace_id;    // Primary workspace
$user->permissions;     // User permissions
```

### Namespace

```php
$namespace = $context->namespace;

if ($namespace) {
    $namespace->id;         // Namespace ID
    $namespace->name;       // Namespace name
    $namespace->entitlements; // Feature access
}
```

## Multi-Workspace Access

### Switching Context

```php
// User with access to multiple workspaces
$workspaces = $user->workspaces;

foreach ($workspaces as $workspace) {
    $context = new WorkspaceContext($workspace, $user);

    // Execute in workspace context
    $result = $tool->execute($params, $context);
}
```

### Cross-Workspace Queries (Admin)

```php
// Requires admin permission
$result = $tool->execute([
    'query' => 'SELECT * FROM posts',
    'bypass_workspace_scope' => true,
], $context);

// Returns posts from all workspaces
```

## Error Handling

### Missing Context

```php
use Core\Mcp\Exceptions\MissingWorkspaceContextException;

try {
    $tool->execute($params); // No context provided
} catch (MissingWorkspaceContextException $e) {
    return response()->json([
        'error' => 'Workspace context required',
        'message' => 'Please provide X-Workspace-ID header',
    ], 400);
}
```

### Invalid Workspace

```php
use Core\Mod\Tenant\Exceptions\WorkspaceNotFoundException;

try {
    $context = WorkspaceContext::fromRequest($request);
} catch (WorkspaceNotFoundException $e) {
    return response()->json([
        'error' => 'Invalid workspace',
        'message' => 'Workspace not found',
    ], 404);
}
```

### Unauthorized Access

```php
use Illuminate\Auth\Access\AuthorizationException;

try {
    $context = WorkspaceContext::fromRequest($request);
} catch (AuthorizationException $e) {
    return response()->json([
        'error' => 'Unauthorized',
        'message' => 'You do not have access to this workspace',
    ], 403);
}
```

## Testing

```php
use Tests\TestCase;
use Core\Mcp\Context\WorkspaceContext;

class WorkspaceContextTest extends TestCase
{
    public function test_resolves_from_header(): void
    {
        $workspace = Workspace::factory()->create();

        $response = $this->withHeaders([
            'X-Workspace-ID' => $workspace->id,
        ])->postJson('/mcp/query', [...]);

        $response->assertStatus(200);
    }

    public function test_scopes_queries_to_workspace(): void
    {
        $workspace1 = Workspace::factory()->create();
        $workspace2 = Workspace::factory()->create();

        Post::factory()->create(['workspace_id' => $workspace1->id]);
        Post::factory()->create(['workspace_id' => $workspace2->id]);

        $context = new WorkspaceContext($workspace1);

        $result = $tool->execute([
            'query' => 'SELECT * FROM posts',
        ], $context);

        $this->assertCount(1, $result['rows']);
        $this->assertEquals($workspace1->id, $result['rows'][0]['workspace_id']);
    }

    public function test_throws_when_context_missing(): void
    {
        $this->expectException(MissingWorkspaceContextException::class);

        $tool->execute(['query' => 'SELECT * FROM posts']);
    }
}
```

## Best Practices

### 1. Always Validate Context

```php
// ✅ Good - validate context
public function execute(array $params)
{
    $this->validateWorkspaceContext();
    // ...
}

// ❌ Bad - no validation
public function execute(array $params)
{
    // Potential workspace bypass
}
```

### 2. Use BelongsToWorkspace Trait

```php
// ✅ Good - automatic scoping
class Post extends Model
{
    use BelongsToWorkspace;
}

// ❌ Bad - manual filtering
Post::where('workspace_id', $workspace->id)->get();
```

### 3. Provide Clear Errors

```php
// ✅ Good - helpful error
throw new MissingWorkspaceContextException(
    'Please provide X-Workspace-ID header'
);

// ❌ Bad - generic error
throw new Exception('Error');
```

### 4. Test Context Isolation

```php
// ✅ Good - test workspace boundaries
public function test_cannot_access_other_workspace(): void
{
    $workspace1 = Workspace::factory()->create();
    $workspace2 = Workspace::factory()->create();

    $context = new WorkspaceContext($workspace1);

    $post = Post::factory()->create(['workspace_id' => $workspace2->id]);

    $result = Post::find($post->id); // Should be null

    $this->assertNull($result);
}
```

## Learn More

- [Multi-Tenancy →](/packages/core/tenancy)
- [Security →](/packages/mcp/security)
- [Creating Tools →](/packages/mcp/tools)
