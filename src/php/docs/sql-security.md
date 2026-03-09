# Guide: SQL Security

This guide documents the security controls for the Query Database MCP tool, including allowed SQL patterns, forbidden operations, and parameterized query requirements.

## Overview

The MCP Query Database tool provides AI agents with read-only SQL access. Multiple security layers protect against:

- SQL injection attacks
- Data modification/destruction
- Cross-tenant data access
- Resource exhaustion
- Information leakage

## Allowed SQL Patterns

### SELECT-Only Queries

Only `SELECT` statements are permitted. All queries must begin with `SELECT`:

```sql
-- Allowed: Basic SELECT
SELECT * FROM posts WHERE status = 'published';

-- Allowed: Specific columns
SELECT id, title, created_at FROM posts;

-- Allowed: COUNT queries
SELECT COUNT(*) FROM users WHERE active = 1;

-- Allowed: Aggregation
SELECT status, COUNT(*) as count FROM posts GROUP BY status;

-- Allowed: JOIN queries
SELECT posts.title, users.name
FROM posts
JOIN users ON posts.user_id = users.id;

-- Allowed: ORDER BY and LIMIT
SELECT * FROM posts ORDER BY created_at DESC LIMIT 10;

-- Allowed: WHERE with multiple conditions
SELECT * FROM posts
WHERE status = 'published'
  AND user_id = 42
  AND created_at > '2024-01-01';
```

### Supported Operators

WHERE clauses support these operators:

| Operator | Example |
|----------|---------|
| `=` | `WHERE status = 'active'` |
| `!=`, `<>` | `WHERE status != 'deleted'` |
| `>`, `>=` | `WHERE created_at > '2024-01-01'` |
| `<`, `<=` | `WHERE views < 1000` |
| `LIKE` | `WHERE title LIKE '%search%'` |
| `IN` | `WHERE status IN ('draft', 'published')` |
| `BETWEEN` | `WHERE created_at BETWEEN '2024-01-01' AND '2024-12-31'` |
| `IS NULL` | `WHERE deleted_at IS NULL` |
| `IS NOT NULL` | `WHERE email IS NOT NULL` |
| `AND` | `WHERE a = 1 AND b = 2` |
| `OR` | `WHERE status = 'draft' OR status = 'review'` |

## Forbidden Operations

### Data Modification (Blocked)

```sql
-- BLOCKED: INSERT
INSERT INTO users (name) VALUES ('attacker');

-- BLOCKED: UPDATE
UPDATE users SET role = 'admin' WHERE id = 1;

-- BLOCKED: DELETE
DELETE FROM users WHERE id = 1;

-- BLOCKED: REPLACE
REPLACE INTO users (id, name) VALUES (1, 'changed');
```

### Schema Modification (Blocked)

```sql
-- BLOCKED: DROP
DROP TABLE users;
DROP DATABASE production;

-- BLOCKED: TRUNCATE
TRUNCATE TABLE logs;

-- BLOCKED: ALTER
ALTER TABLE users ADD COLUMN backdoor TEXT;

-- BLOCKED: CREATE
CREATE TABLE malicious_table (...);

-- BLOCKED: RENAME
RENAME TABLE users TO users_backup;
```

### Permission Operations (Blocked)

```sql
-- BLOCKED: GRANT
GRANT ALL ON *.* TO 'attacker'@'%';

-- BLOCKED: REVOKE
REVOKE SELECT ON database.* FROM 'user'@'%';

-- BLOCKED: FLUSH
FLUSH PRIVILEGES;
```

### System Operations (Blocked)

```sql
-- BLOCKED: File operations
SELECT * FROM posts INTO OUTFILE '/tmp/data.csv';
SELECT LOAD_FILE('/etc/passwd');
LOAD DATA INFILE '/etc/passwd' INTO TABLE users;

-- BLOCKED: Execution
EXECUTE prepared_statement;
CALL stored_procedure();
PREPARE stmt FROM 'SELECT ...';

-- BLOCKED: Variables
SET @var = (SELECT password FROM users);
SET GLOBAL max_connections = 1;
```

### Complete Blocked Keywords List

```php
// Data modification
'INSERT', 'UPDATE', 'DELETE', 'REPLACE', 'TRUNCATE'

// Schema changes
'DROP', 'ALTER', 'CREATE', 'RENAME'

// Permissions
'GRANT', 'REVOKE', 'FLUSH'

// System
'KILL', 'RESET', 'PURGE'

// File operations
'INTO OUTFILE', 'INTO DUMPFILE', 'LOAD_FILE', 'LOAD DATA'

// Execution
'EXECUTE', 'EXEC', 'PREPARE', 'DEALLOCATE', 'CALL'

// Variables
'SET '
```

## SQL Injection Prevention

### Dangerous Patterns (Detected and Blocked)

The validator detects and blocks common injection patterns:

#### Stacked Queries

```sql
-- BLOCKED: Multiple statements
SELECT * FROM posts; DROP TABLE users;
SELECT * FROM posts; DELETE FROM logs;
```

#### UNION Injection

```sql
-- BLOCKED: UNION attacks
SELECT * FROM posts WHERE id = 1 UNION SELECT password FROM users;
SELECT * FROM posts UNION ALL SELECT * FROM secrets;
```

#### Comment Obfuscation

```sql
-- BLOCKED: Comments hiding keywords
SELECT * FROM posts WHERE id = 1 /**/UNION/**/SELECT password FROM users;
SELECT * FROM posts; -- DROP TABLE users
SELECT * FROM posts # DELETE FROM logs
```

#### Hex Encoding

```sql
-- BLOCKED: Hex-encoded strings
SELECT * FROM posts WHERE id = 0x313B44524F50205441424C4520757365727320;
```

#### Time-Based Attacks

```sql
-- BLOCKED: Timing attacks
SELECT * FROM posts WHERE id = 1 AND SLEEP(10);
SELECT * FROM posts WHERE BENCHMARK(10000000, SHA1('test'));
```

#### System Table Access

```sql
-- BLOCKED: Information schema
SELECT * FROM information_schema.tables;
SELECT * FROM information_schema.columns WHERE table_name = 'users';

-- BLOCKED: MySQL system tables
SELECT * FROM mysql.user;
SELECT * FROM performance_schema.threads;
SELECT * FROM sys.session;
```

#### Subquery in WHERE

```sql
-- BLOCKED: Potential data exfiltration
SELECT * FROM posts WHERE id = (SELECT user_id FROM admins LIMIT 1);
```

### Detection Patterns

The validator uses these regex patterns to detect attacks:

```php
// Stacked queries
'/;\s*\S/i'

// UNION injection
'/\bUNION\b/i'

// Hex encoding
'/0x[0-9a-f]+/i'

// Dangerous functions
'/\bCHAR\s*\(/i'
'/\bBENCHMARK\s*\(/i'
'/\bSLEEP\s*\(/i'

// System tables
'/\bINFORMATION_SCHEMA\b/i'
'/\bmysql\./i'
'/\bperformance_schema\./i'
'/\bsys\./i'

// Subquery in WHERE
'/WHERE\s+.*\(\s*SELECT/i'

// Comment obfuscation
'/\/\*[^*]*\*\/\s*(?:UNION|SELECT|INSERT|UPDATE|DELETE|DROP)/i'
```

## Parameterized Queries

**Always use parameter bindings** instead of string interpolation:

### Correct Usage

```php
// SAFE: Parameterized query
$result = $tool->execute([
    'query' => 'SELECT * FROM posts WHERE user_id = ? AND status = ?',
    'bindings' => [$userId, 'published'],
]);

// SAFE: Multiple parameters
$result = $tool->execute([
    'query' => 'SELECT * FROM orders WHERE created_at BETWEEN ? AND ? AND total > ?',
    'bindings' => ['2024-01-01', '2024-12-31', 100.00],
]);
```

### Incorrect Usage (Vulnerable)

```php
// VULNERABLE: String interpolation
$result = $tool->execute([
    'query' => "SELECT * FROM posts WHERE user_id = {$userId}",
]);

// VULNERABLE: Concatenation
$query = "SELECT * FROM posts WHERE status = '" . $status . "'";
$result = $tool->execute(['query' => $query]);

// VULNERABLE: sprintf
$query = sprintf("SELECT * FROM posts WHERE id = %d", $id);
$result = $tool->execute(['query' => $query]);
```

### Why Bindings Matter

With bindings, malicious input is escaped automatically:

```php
// User input
$userInput = "'; DROP TABLE users; --";

// With bindings: SAFE (input is escaped)
$tool->execute([
    'query' => 'SELECT * FROM posts WHERE title = ?',
    'bindings' => [$userInput],
]);
// Executed as: SELECT * FROM posts WHERE title = '\'; DROP TABLE users; --'

// Without bindings: VULNERABLE
$tool->execute([
    'query' => "SELECT * FROM posts WHERE title = '$userInput'",
]);
// Executed as: SELECT * FROM posts WHERE title = ''; DROP TABLE users; --'
```

## Whitelist-Based Validation

The validator uses a whitelist approach, only allowing queries matching known-safe patterns:

### Default Whitelist Patterns

```php
// Simple SELECT with optional WHERE
'/^\s*SELECT\s+[\w\s,.*`]+\s+FROM\s+`?\w+`?
  (\s+WHERE\s+[\w\s`.,!=<>\'"%()]+)*
  (\s+ORDER\s+BY\s+[\w\s,`]+)?
  (\s+LIMIT\s+\d+)?;?\s*$/i'

// COUNT queries
'/^\s*SELECT\s+COUNT\s*\(\s*\*?\s*\)
  \s+FROM\s+`?\w+`?
  (\s+WHERE\s+[\w\s`.,!=<>\'"%()]+)*;?\s*$/i'

// Explicit column list
'/^\s*SELECT\s+`?\w+`?(\s*,\s*`?\w+`?)*
  \s+FROM\s+`?\w+`?
  (\s+WHERE\s+[\w\s`.,!=<>\'"%()]+)*
  (\s+ORDER\s+BY\s+[\w\s,`]+)?
  (\s+LIMIT\s+\d+)?;?\s*$/i'
```

### Adding Custom Patterns

```php
// config/mcp.php
'database' => [
    'use_whitelist' => true,
    'whitelist_patterns' => [
        // Allow specific JOIN pattern
        '/^\s*SELECT\s+[\w\s,.*`]+\s+FROM\s+posts\s+JOIN\s+users\s+ON\s+posts\.user_id\s*=\s*users\.id/i',
    ],
],
```

## Connection Security

### Allowed Connections

Only whitelisted database connections can be queried:

```php
// config/mcp.php
'database' => [
    'allowed_connections' => [
        'mysql',      // Primary database
        'analytics',  // Read-only analytics
        'logs',       // Application logs
    ],
    'connection' => 'mcp_readonly', // Default MCP connection
],
```

### Read-Only Database User

Create a dedicated read-only user for MCP:

```sql
-- Create read-only user
CREATE USER 'mcp_readonly'@'%' IDENTIFIED BY 'secure_password';

-- Grant SELECT only
GRANT SELECT ON app_database.* TO 'mcp_readonly'@'%';

-- Explicitly deny write operations
REVOKE INSERT, UPDATE, DELETE, DROP, CREATE, ALTER
ON app_database.* FROM 'mcp_readonly'@'%';

FLUSH PRIVILEGES;
```

Configure in Laravel:

```php
// config/database.php
'connections' => [
    'mcp_readonly' => [
        'driver' => 'mysql',
        'host' => env('DB_HOST'),
        'database' => env('DB_DATABASE'),
        'username' => env('MCP_DB_USER', 'mcp_readonly'),
        'password' => env('MCP_DB_PASSWORD'),
        'charset' => 'utf8mb4',
        'collation' => 'utf8mb4_unicode_ci',
        'strict' => true,
    ],
],
```

## Blocked Tables

Configure tables that cannot be queried:

```php
// config/mcp.php
'database' => [
    'blocked_tables' => [
        'users',              // User credentials
        'password_resets',    // Password tokens
        'sessions',           // Session data
        'api_keys',           // API credentials
        'oauth_access_tokens', // OAuth tokens
        'personal_access_tokens', // Sanctum tokens
        'failed_jobs',        // Job queue data
    ],
],
```

The validator checks for table references in multiple formats:

```php
// All these are blocked for 'users' table:
'SELECT * FROM users'
'SELECT * FROM `users`'
'SELECT posts.*, users.name FROM posts JOIN users...'
'SELECT users.email FROM ...'
```

## Row Limits

Automatic row limits prevent data exfiltration:

```php
// config/mcp.php
'database' => [
    'max_rows' => 1000, // Maximum rows per query
],
```

If query doesn't include LIMIT, one is added automatically:

```php
// Query without LIMIT
$tool->execute(['query' => 'SELECT * FROM posts']);
// Becomes: SELECT * FROM posts LIMIT 1000

// Query with smaller LIMIT (preserved)
$tool->execute(['query' => 'SELECT * FROM posts LIMIT 10']);
// Stays: SELECT * FROM posts LIMIT 10
```

## Error Handling

### Forbidden Query Response

```json
{
    "error": "Query rejected: Disallowed SQL keyword 'DELETE' detected"
}
```

### Invalid Structure Response

```json
{
    "error": "Query rejected: Query must begin with SELECT"
}
```

### Not Whitelisted Response

```json
{
    "error": "Query rejected: Query does not match any allowed pattern"
}
```

### Sanitized SQL Errors

Database errors are sanitized to prevent information leakage:

```php
// Original error (logged for debugging)
"SQLSTATE[42S02]: Table 'production.secret_table' doesn't exist at 192.168.1.100"

// Sanitized response (returned to client)
"Query execution failed: Table '[path]' doesn't exist at [ip]"
```

## Configuration Reference

```php
// config/mcp.php
return [
    'database' => [
        // Database connection for MCP queries
        'connection' => env('MCP_DB_CONNECTION', 'mcp_readonly'),

        // Use whitelist validation (recommended: true)
        'use_whitelist' => true,

        // Custom whitelist patterns (regex)
        'whitelist_patterns' => [],

        // Tables that cannot be queried
        'blocked_tables' => [
            'users',
            'password_resets',
            'sessions',
            'api_keys',
        ],

        // Maximum rows per query
        'max_rows' => 1000,

        // Query execution timeout (milliseconds)
        'timeout' => 5000,

        // Enable EXPLAIN analysis
        'enable_explain' => true,
    ],
];
```

## Testing Security

```php
use Tests\TestCase;
use Core\Mod\Mcp\Services\SqlQueryValidator;
use Core\Mod\Mcp\Exceptions\ForbiddenQueryException;

class SqlSecurityTest extends TestCase
{
    private SqlQueryValidator $validator;

    protected function setUp(): void
    {
        parent::setUp();
        $this->validator = new SqlQueryValidator();
    }

    public function test_blocks_delete(): void
    {
        $this->expectException(ForbiddenQueryException::class);
        $this->validator->validate('DELETE FROM users');
    }

    public function test_blocks_union_injection(): void
    {
        $this->expectException(ForbiddenQueryException::class);
        $this->validator->validate("SELECT * FROM posts UNION SELECT password FROM users");
    }

    public function test_blocks_stacked_queries(): void
    {
        $this->expectException(ForbiddenQueryException::class);
        $this->validator->validate("SELECT * FROM posts; DROP TABLE users");
    }

    public function test_blocks_system_tables(): void
    {
        $this->expectException(ForbiddenQueryException::class);
        $this->validator->validate("SELECT * FROM information_schema.tables");
    }

    public function test_allows_safe_select(): void
    {
        $this->validator->validate("SELECT id, title FROM posts WHERE status = 'published'");
        $this->assertTrue(true); // No exception = pass
    }

    public function test_allows_count(): void
    {
        $this->validator->validate("SELECT COUNT(*) FROM posts");
        $this->assertTrue(true);
    }
}
```

## Best Practices Summary

1. **Always use parameterized queries** - Never interpolate values into SQL strings
2. **Use a read-only database user** - Database-level protection against modifications
3. **Configure blocked tables** - Prevent access to sensitive data
4. **Enable whitelist validation** - Only allow known-safe query patterns
5. **Set appropriate row limits** - Prevent large data exports
6. **Review logs regularly** - Monitor for suspicious query patterns
7. **Test security controls** - Include injection tests in your test suite

## Learn More

- [Query Database Tool](/packages/mcp/query-database) - Tool usage
- [Workspace Context](/packages/mcp/workspace) - Multi-tenant isolation
- [Creating MCP Tools](/packages/mcp/creating-mcp-tools) - Tool development
