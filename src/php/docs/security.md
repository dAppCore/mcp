# MCP Security

Security features for protecting database access and preventing SQL injection in MCP tools.

## SQL Query Validation

### Validation Rules

The `SqlQueryValidator` enforces strict rules on all queries:

**Allowed:**
- `SELECT` statements only
- Table/column qualifiers
- WHERE clauses
- JOINs
- ORDER BY, GROUP BY
- LIMIT clauses
- Subqueries (SELECT only)

**Forbidden:**
- `INSERT`, `UPDATE`, `DELETE`, `DROP`, `CREATE`, `ALTER`
- `TRUNCATE`, `GRANT`, `REVOKE`
- Database modification operations
- System table access
- Multiple statements (`;` separated)

### Usage

```php
use Core\Mcp\Services\SqlQueryValidator;

$validator = app(SqlQueryValidator::class);

// Valid query
$result = $validator->validate('SELECT * FROM posts WHERE id = ?');
// Returns: ['valid' => true]

// Invalid query
$result = $validator->validate('DROP TABLE users');
// Returns: ['valid' => false, 'error' => 'Only SELECT queries are allowed']
```

### Forbidden Patterns

```php
// ❌ Data modification
DELETE FROM users WHERE id = 1
UPDATE posts SET status = 'published'
INSERT INTO logs VALUES (...)

// ❌ Schema changes
DROP TABLE posts
ALTER TABLE users ADD COLUMN...
CREATE INDEX...

// ❌ Permission changes
GRANT ALL ON *.* TO user
REVOKE SELECT ON posts FROM user

// ❌ Multiple statements
SELECT * FROM posts; DROP TABLE users;

// ❌ System tables
SELECT * FROM information_schema.tables
SELECT * FROM mysql.user
```

### Parameterized Queries

Always use bindings to prevent SQL injection:

```php
// ✅ Good - parameterized
$tool->execute([
    'query' => 'SELECT * FROM posts WHERE user_id = ? AND status = ?',
    'bindings' => [$userId, 'published'],
]);

// ❌ Bad - SQL injection risk
$tool->execute([
    'query' => "SELECT * FROM posts WHERE user_id = {$userId}",
]);
```

## Workspace Context Security

### Automatic Scoping

Queries are automatically scoped to the current workspace:

```php
use Core\Mcp\Context\WorkspaceContext;

// Get workspace context from request
$context = WorkspaceContext::fromRequest($request);

// Queries automatically filtered by workspace_id
$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE status = ?',
    'bindings' => ['published'],
], $context);

// Internally becomes:
// SELECT * FROM posts WHERE status = ? AND workspace_id = ?
```

### Validation

Tools validate workspace context before execution:

```php
use Core\Mcp\Tools\Concerns\RequiresWorkspaceContext;

class MyTool
{
    use RequiresWorkspaceContext;

    public function execute(array $params)
    {
        // Throws MissingWorkspaceContextException if context missing
        $this->validateWorkspaceContext();

        // Safe to proceed
        $workspace = $this->workspaceContext->workspace;
    }
}
```

### Bypassing (Admin Only)

```php
// Requires admin permission
$result = $tool->execute([
    'query' => 'SELECT * FROM posts',
    'bypass_workspace_scope' => true, // Admin only
]);
```

## Connection Security

### Allowed Connections

Only specific connections can be queried:

```php
// config/mcp.php
return [
    'database' => [
        'allowed_connections' => [
            'mysql',       // Primary database
            'analytics',   // Read-only analytics
            'logs',        // Application logs
        ],
        'default_connection' => 'mysql',
    ],
];
```

### Read-Only Connections

Use read-only database users for MCP:

```php
// config/database.php
'connections' => [
    'mcp_readonly' => [
        'driver' => 'mysql',
        'host' => env('DB_HOST'),
        'database' => env('DB_DATABASE'),
        'username' => env('MCP_DB_USER'), // Read-only user
        'password' => env('MCP_DB_PASSWORD'),
        'charset' => 'utf8mb4',
    ],
],
```

**Database Setup:**

```sql
-- Create read-only user
CREATE USER 'mcp_readonly'@'%' IDENTIFIED BY 'secure_password';

-- Grant SELECT only
GRANT SELECT ON app_database.* TO 'mcp_readonly'@'%';

-- Explicitly deny modifications
REVOKE INSERT, UPDATE, DELETE, DROP, CREATE, ALTER ON app_database.* FROM 'mcp_readonly'@'%';

FLUSH PRIVILEGES;
```

### Connection Validation

```php
use Core\Mcp\Services\ConnectionValidator;

$validator = app(ConnectionValidator::class);

// Check if connection is allowed
if (!$validator->isAllowed('mysql')) {
    throw new ForbiddenConnectionException();
}

// Check if connection exists
if (!$validator->exists('mysql')) {
    throw new InvalidConnectionException();
}
```

## Rate Limiting

Prevent abuse with rate limits:

```php
use Core\Mcp\Middleware\CheckMcpQuota;

Route::middleware([CheckMcpQuota::class])
    ->post('/mcp/query', [McpApiController::class, 'query']);
```

**Limits:**

| Tier | Requests/Hour | Queries/Day |
|------|--------------|-------------|
| Free | 60 | 500 |
| Pro | 600 | 10,000 |
| Enterprise | Unlimited | Unlimited |

### Quota Enforcement

```php
use Core\Mcp\Services\McpQuotaService;

$quota = app(McpQuotaService::class);

// Check if within quota
if (!$quota->withinLimit($workspace)) {
    throw new QuotaExceededException();
}

// Record usage
$quota->recordUsage($workspace, 'query_database');
```

## Query Logging

All queries are logged for audit:

```php
// storage/logs/mcp-queries.log
[2026-01-26 12:00:00] Query executed
  Workspace: acme-corp
  User: john@example.com
  Query: SELECT * FROM posts WHERE status = ?
  Bindings: ["published"]
  Rows: 42
  Duration: 5.23ms
```

### Log Configuration

```php
// config/logging.php
'channels' => [
    'mcp' => [
        'driver' => 'daily',
        'path' => storage_path('logs/mcp-queries.log'),
        'level' => 'info',
        'days' => 90, // Retain for 90 days
    ],
],
```

## Best Practices

### 1. Always Use Bindings

```php
// ✅ Good - parameterized
'query' => 'SELECT * FROM posts WHERE id = ?',
'bindings' => [$id],

// ❌ Bad - SQL injection risk
'query' => "SELECT * FROM posts WHERE id = {$id}",
```

### 2. Limit Result Sets

```php
// ✅ Good - limited results
'query' => 'SELECT * FROM posts LIMIT 100',

// ❌ Bad - unbounded query
'query' => 'SELECT * FROM posts',
```

### 3. Use Read-Only Connections

```php
// ✅ Good - read-only user
'connection' => 'mcp_readonly',

// ❌ Bad - admin connection
'connection' => 'mysql_admin',
```

### 4. Validate Workspace Context

```php
// ✅ Good - validate context
$this->validateWorkspaceContext();

// ❌ Bad - no validation
// (workspace boundary bypass risk)
```

## Testing

```php
use Tests\TestCase;
use Core\Mcp\Services\SqlQueryValidator;

class SecurityTest extends TestCase
{
    public function test_blocks_destructive_queries(): void
    {
        $validator = app(SqlQueryValidator::class);

        $result = $validator->validate('DROP TABLE users');

        $this->assertFalse($result['valid']);
        $this->assertStringContainsString('Only SELECT', $result['error']);
    }

    public function test_allows_select_queries(): void
    {
        $validator = app(SqlQueryValidator::class);

        $result = $validator->validate('SELECT * FROM posts WHERE id = ?');

        $this->assertTrue($result['valid']);
    }

    public function test_enforces_workspace_scope(): void
    {
        $workspace = Workspace::factory()->create();
        $context = new WorkspaceContext($workspace);

        $result = $tool->execute([
            'query' => 'SELECT * FROM posts',
        ], $context);

        // Should only return workspace's posts
        $this->assertEquals($workspace->id, $result['rows'][0]['workspace_id']);
    }
}
```

## Learn More

- [Query Database →](/packages/mcp/query-database)
- [Workspace Context →](/packages/mcp/workspace)
- [Quotas →](/packages/mcp/quotas)
