# Query Database Tool

The MCP package provides a secure SQL query execution tool with validation, connection management, and EXPLAIN plan analysis.

## Overview

The Query Database tool allows AI agents to:
- Execute SELECT queries safely
- Analyze query performance
- Access multiple database connections
- Prevent destructive operations
- Enforce workspace context

## Basic Usage

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

## Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | SQL SELECT query |
| `bindings` | array | No | Query parameters (prevents SQL injection) |
| `connection` | string | No | Database connection name (default: default) |
| `explain` | bool | No | Include EXPLAIN plan analysis |

## Security Validation

### Allowed Operations

✅ Only SELECT queries are allowed:

```php
// ✅ Allowed
'SELECT * FROM posts'
'SELECT id, title FROM posts WHERE status = ?'
'SELECT COUNT(*) FROM users'

// ❌ Blocked
'DELETE FROM posts'
'UPDATE posts SET status = ?'
'DROP TABLE posts'
'TRUNCATE posts'
```

### Forbidden Keywords

The following are automatically blocked:
- `DROP`
- `TRUNCATE`
- `DELETE`
- `UPDATE`
- `INSERT`
- `ALTER`
- `CREATE`
- `GRANT`
- `REVOKE`

### Required WHERE Clauses

Queries on large tables must include WHERE clauses:

```php
// ✅ Good - has WHERE clause
'SELECT * FROM posts WHERE user_id = ?'

// ⚠️ Warning - no WHERE clause
'SELECT * FROM posts'
// Returns warning if table has > 10,000 rows
```

### Connection Validation

Only whitelisted connections are accessible:

```php
// config/mcp.php
'query_database' => [
    'allowed_connections' => ['mysql', 'pgsql', 'analytics'],
],
```

## EXPLAIN Plan Analysis

Enable query optimization insights:

```php
$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE status = ?',
    'bindings' => ['published'],
    'explain' => true,
]);

// Returns additional 'explain' key:
// [
//   'rows' => [...],
//   'explain' => [
//     'type' => 'ref',
//     'key' => 'idx_status',
//     'rows_examined' => 150,
//     'analysis' => 'Query uses index. Performance: Good',
//     'recommendations' => []
//   ]
// ]
```

### Performance Analysis

The EXPLAIN analyzer provides human-readable insights:

**Good Performance:**
```
"Query uses index. Performance: Good"
```

**Index Missing:**
```
"Warning: Full table scan detected. Consider adding an index on 'status'"
```

**High Row Count:**
```
"Warning: Query examines 50,000 rows. Consider adding WHERE clause to limit results"
```

## Examples

### Basic SELECT

```php
$result = $tool->execute([
    'query' => 'SELECT id, title, created_at FROM posts LIMIT 10',
]);

foreach ($result['rows'] as $row) {
    echo "{$row['title']}\n";
}
```

### With Parameters

```php
$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE user_id = ? AND status = ?',
    'bindings' => [42, 'published'],
]);
```

### Aggregation

```php
$result = $tool->execute([
    'query' => 'SELECT status, COUNT(*) as count FROM posts GROUP BY status',
]);

// Returns: [
//   ['status' => 'draft', 'count' => 15],
//   ['status' => 'published', 'count' => 42],
// ]
```

### Join Query

```php
$result = $tool->execute([
    'query' => '
        SELECT posts.title, users.name
        FROM posts
        JOIN users ON posts.user_id = users.id
        WHERE posts.status = ?
        LIMIT 10
    ',
    'bindings' => ['published'],
]);
```

### Date Filtering

```php
$result = $tool->execute([
    'query' => '
        SELECT *
        FROM posts
        WHERE created_at >= ?
          AND created_at < ?
        ORDER BY created_at DESC
    ',
    'bindings' => ['2024-01-01', '2024-02-01'],
]);
```

## Multiple Connections

Query different databases:

```php
// Main application database
$posts = $tool->execute([
    'query' => 'SELECT * FROM posts',
    'connection' => 'mysql',
]);

// Analytics database
$stats = $tool->execute([
    'query' => 'SELECT * FROM page_views',
    'connection' => 'analytics',
]);

// PostgreSQL database
$data = $tool->execute([
    'query' => 'SELECT * FROM logs',
    'connection' => 'pgsql',
]);
```

## Error Handling

### Forbidden Query

```php
$result = $tool->execute([
    'query' => 'DELETE FROM posts WHERE id = 1',
]);

// Returns:
// [
//   'success' => false,
//   'error' => 'Forbidden query: DELETE operations not allowed',
//   'code' => 'FORBIDDEN_QUERY'
// ]
```

### Invalid Connection

```php
$result = $tool->execute([
    'query' => 'SELECT * FROM posts',
    'connection' => 'unknown',
]);

// Returns:
// [
//   'success' => false,
//   'error' => 'Connection "unknown" not allowed',
//   'code' => 'INVALID_CONNECTION'
// ]
```

### SQL Error

```php
$result = $tool->execute([
    'query' => 'SELECT * FROM nonexistent_table',
]);

// Returns:
// [
//   'success' => false,
//   'error' => 'Table "nonexistent_table" doesn\'t exist',
//   'code' => 'SQL_ERROR'
// ]
```

## Configuration

```php
// config/mcp.php
'query_database' => [
    // Allowed database connections
    'allowed_connections' => [
        'mysql',
        'pgsql',
        'analytics',
    ],

    // Forbidden SQL keywords
    'forbidden_keywords' => [
        'DROP', 'TRUNCATE', 'DELETE', 'UPDATE', 'INSERT',
        'ALTER', 'CREATE', 'GRANT', 'REVOKE',
    ],

    // Maximum execution time (milliseconds)
    'max_execution_time' => 5000,

    // Enable EXPLAIN plan analysis
    'enable_explain' => true,

    // Warn on queries without WHERE clause for tables larger than:
    'warn_no_where_threshold' => 10000,
],
```

## Workspace Context

Queries are automatically scoped to the current workspace:

```php
// When workspace context is set
$result = $tool->execute([
    'query' => 'SELECT * FROM posts',
]);

// Equivalent to:
// 'SELECT * FROM posts WHERE workspace_id = ?'
// with workspace_id automatically added
```

Disable automatic scoping:

```php
$result = $tool->execute([
    'query' => 'SELECT * FROM global_settings',
    'ignore_workspace_scope' => true,
]);
```

## Best Practices

### 1. Always Use Bindings

```php
// ✅ Good - prevents SQL injection
$tool->execute([
    'query' => 'SELECT * FROM posts WHERE user_id = ?',
    'bindings' => [$userId],
]);

// ❌ Bad - vulnerable to SQL injection
$tool->execute([
    'query' => "SELECT * FROM posts WHERE user_id = {$userId}",
]);
```

### 2. Limit Results

```php
// ✅ Good - limits results
'SELECT * FROM posts LIMIT 100'

// ❌ Bad - could return millions of rows
'SELECT * FROM posts'
```

### 3. Use EXPLAIN for Optimization

```php
// ✅ Good - analyze slow queries
$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE status = ?',
    'bindings' => ['published'],
    'explain' => true,
]);

if (isset($result['explain']['recommendations'])) {
    foreach ($result['explain']['recommendations'] as $rec) {
        error_log("Query optimization: {$rec}");
    }
}
```

### 4. Handle Errors Gracefully

```php
// ✅ Good - check for errors
$result = $tool->execute([...]);

if (!($result['success'] ?? true)) {
    return [
        'error' => $result['error'],
        'code' => $result['code'],
    ];
}

return $result['rows'];
```

## Testing

```php
<?php

namespace Tests\Feature\Mcp;

use Tests\TestCase;
use Core\Mcp\Tools\QueryDatabase;

class QueryDatabaseTest extends TestCase
{
    public function test_executes_select_query(): void
    {
        Post::factory()->create(['title' => 'Test Post']);

        $tool = new QueryDatabase();

        $result = $tool->execute([
            'query' => 'SELECT * FROM posts WHERE title = ?',
            'bindings' => ['Test Post'],
        ]);

        $this->assertTrue($result['success'] ?? true);
        $this->assertCount(1, $result['rows']);
    }

    public function test_blocks_delete_query(): void
    {
        $tool = new QueryDatabase();

        $result = $tool->execute([
            'query' => 'DELETE FROM posts WHERE id = 1',
        ]);

        $this->assertFalse($result['success']);
        $this->assertEquals('FORBIDDEN_QUERY', $result['code']);
    }

    public function test_validates_connection(): void
    {
        $tool = new QueryDatabase();

        $result = $tool->execute([
            'query' => 'SELECT 1',
            'connection' => 'invalid',
        ]);

        $this->assertFalse($result['success']);
        $this->assertEquals('INVALID_CONNECTION', $result['code']);
    }
}
```

## Learn More

- [SQL Security →](/packages/mcp/security)
- [Workspace Context →](/packages/mcp/workspace)
- [Tool Analytics →](/packages/mcp/analytics)
