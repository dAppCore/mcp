# Creating MCP Tools

Learn how to create custom MCP tools for AI agents with parameter validation, dependency management, and workspace context.

## Tool Structure

Every MCP tool extends `BaseTool`:

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
                'enum' => ['published', 'draft', 'archived'],
                'required' => false,
            ],
            'limit' => [
                'type' => 'integer',
                'description' => 'Number of posts to return',
                'default' => 10,
                'min' => 1,
                'max' => 100,
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
            'success' => true,
            'posts' => $posts->map(fn ($post) => [
                'id' => $post->id,
                'title' => $post->title,
                'slug' => $post->slug,
                'status' => $post->status,
                'created_at' => $post->created_at->toIso8601String(),
            ])->toArray(),
            'count' => $posts->count(),
        ];
    }
}
```

## Registering Tools

Register tools in your module's `Boot.php`:

```php
<?php

namespace Mod\Blog;

use Core\Events\McpToolsRegistering;
use Mod\Blog\Tools\ListPostsTool;
use Mod\Blog\Tools\CreatePostTool;

class Boot
{
    public static array $listens = [
        McpToolsRegistering::class => 'onMcpTools',
    ];

    public function onMcpTools(McpToolsRegistering $event): void
    {
        $event->tool('blog:list-posts', ListPostsTool::class);
        $event->tool('blog:create-post', CreatePostTool::class);
        $event->tool('blog:get-post', GetPostTool::class);
    }
}
```

## Parameter Validation

### Parameter Types

```php
public function getParameters(): array
{
    return [
        // String
        'title' => [
            'type' => 'string',
            'description' => 'Post title',
            'minLength' => 1,
            'maxLength' => 255,
            'required' => true,
        ],

        // Integer
        'views' => [
            'type' => 'integer',
            'description' => 'Number of views',
            'min' => 0,
            'max' => 1000000,
            'required' => false,
        ],

        // Boolean
        'published' => [
            'type' => 'boolean',
            'description' => 'Is published',
            'required' => false,
        ],

        // Enum
        'status' => [
            'type' => 'string',
            'enum' => ['draft', 'published', 'archived'],
            'description' => 'Post status',
            'required' => true,
        ],

        // Array
        'tags' => [
            'type' => 'array',
            'description' => 'Post tags',
            'items' => ['type' => 'string'],
            'required' => false,
        ],

        // Object
        'metadata' => [
            'type' => 'object',
            'description' => 'Additional metadata',
            'properties' => [
                'featured' => ['type' => 'boolean'],
                'views' => ['type' => 'integer'],
            ],
            'required' => false,
        ],
    ];
}
```

### Default Values

```php
'limit' => [
    'type' => 'integer',
    'default' => 10,  // Used if not provided
    'required' => false,
]
```

### Custom Validation

```php
public function execute(array $params): array
{
    // Additional validation
    if (isset($params['email']) && !filter_var($params['email'], FILTER_VALIDATE_EMAIL)) {
        return [
            'success' => false,
            'error' => 'Invalid email address',
            'code' => 'INVALID_EMAIL',
        ];
    }

    // Tool logic...
}
```

## Workspace Context

### Requiring Workspace

Use the `RequiresWorkspaceContext` trait:

```php
<?php

namespace Mod\Blog\Tools;

use Core\Mcp\Tools\BaseTool;
use Core\Mcp\Tools\Concerns\RequiresWorkspaceContext;

class CreatePostTool extends BaseTool
{
    use RequiresWorkspaceContext;

    public function execute(array $params): array
    {
        // Workspace automatically validated and available
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

### Optional Workspace

```php
public function execute(array $params): array
{
    $workspace = $this->getWorkspaceContext(); // May be null

    $query = Post::query();

    if ($workspace) {
        $query->where('workspace_id', $workspace->id);
    }

    return ['posts' => $query->get()];
}
```

## Tool Dependencies

### Declaring Dependencies

```php
<?php

namespace Mod\Blog\Tools;

use Core\Mcp\Tools\BaseTool;
use Core\Mcp\Dependencies\HasDependencies;
use Core\Mcp\Dependencies\ToolDependency;
use Core\Mcp\Dependencies\DependencyType;

class ImportPostsTool extends BaseTool
{
    use HasDependencies;

    public function getDependencies(): array
    {
        return [
            // Required dependency
            new ToolDependency(
                'blog:list-posts',
                DependencyType::REQUIRED
            ),

            // Optional dependency
            new ToolDependency(
                'media:upload',
                DependencyType::OPTIONAL
            ),
        ];
    }

    public function execute(array $params): array
    {
        // Dependencies automatically validated before execution
        // ...
    }
}
```

### Dependency Types

- `DependencyType::REQUIRED` - Tool cannot execute without this
- `DependencyType::OPTIONAL` - Tool works better with this but not required

## Error Handling

### Standard Error Format

```php
public function execute(array $params): array
{
    try {
        // Tool logic...

        return [
            'success' => true,
            'data' => $result,
        ];

    } catch (\Exception $e) {
        return [
            'success' => false,
            'error' => $e->getMessage(),
            'code' => 'TOOL_EXECUTION_FAILED',
        ];
    }
}
```

### Specific Error Codes

```php
// Validation error
return [
    'success' => false,
    'error' => 'Title is required',
    'code' => 'VALIDATION_ERROR',
    'field' => 'title',
];

// Not found
return [
    'success' => false,
    'error' => 'Post not found',
    'code' => 'NOT_FOUND',
    'resource_id' => $params['id'],
];

// Forbidden
return [
    'success' => false,
    'error' => 'Insufficient permissions',
    'code' => 'FORBIDDEN',
    'required_permission' => 'posts.create',
];
```

## Advanced Patterns

### Tool with File Processing

```php
public function execute(array $params): array
{
    $csvPath = $params['csv_path'];

    if (!file_exists($csvPath)) {
        return [
            'success' => false,
            'error' => 'CSV file not found',
            'code' => 'FILE_NOT_FOUND',
        ];
    }

    $imported = 0;
    $errors = [];

    if (($handle = fopen($csvPath, 'r')) !== false) {
        while (($data = fgetcsv($handle)) !== false) {
            try {
                Post::create([
                    'title' => $data[0],
                    'content' => $data[1],
                ]);
                $imported++;
            } catch (\Exception $e) {
                $errors[] = "Row {$imported}: {$e->getMessage()}";
            }
        }
        fclose($handle);
    }

    return [
        'success' => true,
        'imported' => $imported,
        'errors' => $errors,
    ];
}
```

### Tool with Pagination

```php
public function execute(array $params): array
{
    $page = $params['page'] ?? 1;
    $perPage = $params['per_page'] ?? 15;

    $posts = Post::paginate($perPage, ['*'], 'page', $page);

    return [
        'success' => true,
        'posts' => $posts->items(),
        'pagination' => [
            'current_page' => $posts->currentPage(),
            'last_page' => $posts->lastPage(),
            'per_page' => $posts->perPage(),
            'total' => $posts->total(),
        ],
    ];
}
```

### Tool with Progress Tracking

```php
public function execute(array $params): array
{
    $postIds = $params['post_ids'];
    $total = count($postIds);
    $processed = 0;

    foreach ($postIds as $postId) {
        $post = Post::find($postId);

        if ($post) {
            $post->publish();
            $processed++;

            // Emit progress event
            event(new ToolProgress(
                tool: $this->getName(),
                progress: ($processed / $total) * 100,
                message: "Published post {$postId}"
            ));
        }
    }

    return [
        'success' => true,
        'processed' => $processed,
        'total' => $total,
    ];
}
```

## Testing Tools

```php
<?php

namespace Tests\Feature\Mcp;

use Tests\TestCase;
use Mod\Blog\Tools\ListPostsTool;
use Mod\Blog\Models\Post;

class ListPostsToolTest extends TestCase
{
    public function test_lists_all_posts(): void
    {
        Post::factory()->count(5)->create();

        $tool = new ListPostsTool();

        $result = $tool->execute([]);

        $this->assertTrue($result['success']);
        $this->assertCount(5, $result['posts']);
    }

    public function test_filters_by_status(): void
    {
        Post::factory()->count(3)->create(['status' => 'published']);
        Post::factory()->count(2)->create(['status' => 'draft']);

        $tool = new ListPostsTool();

        $result = $tool->execute([
            'status' => 'published',
        ]);

        $this->assertCount(3, $result['posts']);
    }

    public function test_respects_limit(): void
    {
        Post::factory()->count(20)->create();

        $tool = new ListPostsTool();

        $result = $tool->execute([
            'limit' => 5,
        ]);

        $this->assertCount(5, $result['posts']);
    }
}
```

## Best Practices

### 1. Clear Naming

```php
// ✅ Good - descriptive name
'blog:create-post'
'blog:list-published-posts'
'blog:delete-post'

// ❌ Bad - vague name
'blog:action'
'do-thing'
```

### 2. Detailed Descriptions

```php
// ✅ Good - explains what and why
public function getDescription(): string
{
    return 'Create a new blog post with title, content, and optional metadata. '
         . 'Requires workspace context. Validates entitlements before creation.';
}

// ❌ Bad - too brief
public function getDescription(): string
{
    return 'Creates post';
}
```

### 3. Validate Parameters

```php
// ✅ Good - strict validation
public function getParameters(): array
{
    return [
        'title' => [
            'type' => 'string',
            'required' => true,
            'minLength' => 1,
            'maxLength' => 255,
        ],
    ];
}
```

### 4. Return Consistent Format

```php
// ✅ Good - always includes success
return [
    'success' => true,
    'data' => $result,
];

return [
    'success' => false,
    'error' => $message,
    'code' => $code,
];
```

## Learn More

- [Query Database →](/packages/mcp/query-database)
- [Workspace Context →](/packages/mcp/workspace)
- [Tool Analytics →](/packages/mcp/analytics)
