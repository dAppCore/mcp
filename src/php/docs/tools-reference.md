# API Reference: MCP Tools

Complete reference for all MCP tools including parameters, response formats, and error handling.

## Database Tools

### query_database

Execute read-only SQL queries against the database.

**Description:** Execute a read-only SQL SELECT query against the database

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | Yes | SQL SELECT query to execute. Only read-only SELECT queries are permitted. |
| `explain` | boolean | No | If true, runs EXPLAIN on the query instead of executing it. Useful for query optimization. Default: `false` |

**Example Request:**

```json
{
    "tool": "query_database",
    "arguments": {
        "query": "SELECT id, title, status FROM posts WHERE status = 'published' LIMIT 10"
    }
}
```

**Success Response:**

```json
[
    {"id": 1, "title": "First Post", "status": "published"},
    {"id": 2, "title": "Second Post", "status": "published"}
]
```

**With EXPLAIN:**

```json
{
    "tool": "query_database",
    "arguments": {
        "query": "SELECT * FROM posts WHERE status = 'published'",
        "explain": true
    }
}
```

**EXPLAIN Response:**

```json
{
    "explain": [
        {
            "id": 1,
            "select_type": "SIMPLE",
            "table": "posts",
            "type": "ref",
            "key": "idx_status",
            "rows": 150,
            "Extra": "Using index"
        }
    ],
    "query": "SELECT * FROM posts WHERE status = 'published' LIMIT 1000",
    "interpretation": [
        {
            "table": "posts",
            "analysis": [
                "GOOD: Using index: idx_status"
            ]
        }
    ]
}
```

**Error Response - Forbidden Query:**

```json
{
    "error": "Query rejected: Disallowed SQL keyword 'DELETE' detected"
}
```

**Error Response - Invalid Structure:**

```json
{
    "error": "Query rejected: Query must begin with SELECT"
}
```

**Security Notes:**
- Only SELECT queries are allowed
- Blocked keywords: INSERT, UPDATE, DELETE, DROP, TRUNCATE, ALTER, CREATE, GRANT, REVOKE
- UNION queries are blocked
- System tables (information_schema, mysql.*) are blocked
- Automatic LIMIT applied if not specified
- Use read-only database connection

---

### list_tables

List all database tables in the application.

**Description:** List all database tables

**Parameters:** None

**Example Request:**

```json
{
    "tool": "list_tables",
    "arguments": {}
}
```

**Success Response:**

```json
[
    "users",
    "posts",
    "comments",
    "tags",
    "categories",
    "media",
    "migrations",
    "jobs"
]
```

**Security Notes:**
- Returns table names only, not structure
- Some tables may be filtered based on configuration

---

## Commerce Tools

### get_billing_status

Get billing status for the authenticated workspace.

**Description:** Get billing status for your workspace including subscription, current plan, and billing period

**Parameters:** None (workspace from authentication context)

**Requires:** Workspace Context

**Example Request:**

```json
{
    "tool": "get_billing_status",
    "arguments": {}
}
```

**Success Response:**

```json
{
    "workspace": {
        "id": 42,
        "name": "Acme Corp"
    },
    "subscription": {
        "id": 123,
        "status": "active",
        "gateway": "stripe",
        "billing_cycle": "monthly",
        "current_period_start": "2024-01-01T00:00:00+00:00",
        "current_period_end": "2024-02-01T00:00:00+00:00",
        "days_until_renewal": 15,
        "cancel_at_period_end": false,
        "on_trial": false,
        "trial_ends_at": null
    },
    "packages": [
        {
            "code": "professional",
            "name": "Professional Plan",
            "status": "active",
            "expires_at": null
        }
    ]
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `workspace.id` | integer | Workspace ID |
| `workspace.name` | string | Workspace name |
| `subscription.status` | string | active, trialing, past_due, canceled |
| `subscription.billing_cycle` | string | monthly, yearly |
| `subscription.days_until_renewal` | integer | Days until next billing |
| `subscription.on_trial` | boolean | Currently in trial period |
| `packages` | array | Active feature packages |

**Error Response - No Workspace Context:**

```json
{
    "error": "MCP tool 'get_billing_status' requires workspace context. Authenticate with an API key or user session."
}
```

---

### list_invoices

List invoices for the authenticated workspace.

**Description:** List invoices for your workspace with optional status filter

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `status` | string | No | Filter by status: paid, pending, overdue, void |
| `limit` | integer | No | Maximum invoices to return. Default: 10, Max: 50 |

**Requires:** Workspace Context

**Example Request:**

```json
{
    "tool": "list_invoices",
    "arguments": {
        "status": "paid",
        "limit": 5
    }
}
```

**Success Response:**

```json
{
    "workspace_id": 42,
    "count": 5,
    "invoices": [
        {
            "id": 1001,
            "invoice_number": "INV-2024-001",
            "status": "paid",
            "subtotal": 99.00,
            "discount_amount": 0.00,
            "tax_amount": 19.80,
            "total": 118.80,
            "amount_paid": 118.80,
            "amount_due": 0.00,
            "currency": "GBP",
            "issue_date": "2024-01-01",
            "due_date": "2024-01-15",
            "paid_at": "2024-01-10T14:30:00+00:00",
            "is_overdue": false,
            "order_number": "ORD-2024-001"
        }
    ]
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `invoice_number` | string | Unique invoice identifier |
| `status` | string | paid, pending, overdue, void |
| `total` | number | Total amount including tax |
| `amount_due` | number | Remaining amount to pay |
| `is_overdue` | boolean | Past due date with unpaid balance |

---

### upgrade_plan

Preview or execute a plan upgrade/downgrade.

**Description:** Preview or execute a plan upgrade/downgrade for your workspace subscription

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `package_code` | string | Yes | Code of the new package (e.g., agency, enterprise) |
| `preview` | boolean | No | If true, only preview without executing. Default: `true` |
| `immediate` | boolean | No | If true, apply immediately; false schedules for period end. Default: `true` |

**Requires:** Workspace Context

**Example Request - Preview:**

```json
{
    "tool": "upgrade_plan",
    "arguments": {
        "package_code": "enterprise",
        "preview": true
    }
}
```

**Preview Response:**

```json
{
    "preview": true,
    "current_package": "professional",
    "new_package": "enterprise",
    "proration": {
        "is_upgrade": true,
        "is_downgrade": false,
        "current_plan_price": 99.00,
        "new_plan_price": 299.00,
        "credit_amount": 49.50,
        "prorated_new_cost": 149.50,
        "net_amount": 100.00,
        "requires_payment": true,
        "days_remaining": 15,
        "currency": "GBP"
    }
}
```

**Execute Response:**

```json
{
    "success": true,
    "immediate": true,
    "current_package": "professional",
    "new_package": "enterprise",
    "proration": {
        "is_upgrade": true,
        "net_amount": 100.00
    },
    "subscription_status": "active"
}
```

**Error Response - Package Not Found:**

```json
{
    "error": "Package not found",
    "available_packages": ["starter", "professional", "agency", "enterprise"]
}
```

---

### create_coupon

Create a new discount coupon code.

**Description:** Create a new discount coupon code

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `code` | string | Yes | Unique coupon code (uppercase letters, numbers, hyphens, underscores) |
| `name` | string | Yes | Display name for the coupon |
| `type` | string | No | Discount type: percentage or fixed_amount. Default: percentage |
| `value` | number | Yes | Discount value (1-100 for percentage, or fixed amount) |
| `duration` | string | No | How long discount applies: once, repeating, forever. Default: once |
| `max_uses` | integer | No | Maximum total uses (null for unlimited) |
| `valid_until` | string | No | Expiry date in YYYY-MM-DD format |

**Example Request:**

```json
{
    "tool": "create_coupon",
    "arguments": {
        "code": "SUMMER25",
        "name": "Summer Sale 2024",
        "type": "percentage",
        "value": 25,
        "duration": "once",
        "max_uses": 100,
        "valid_until": "2024-08-31"
    }
}
```

**Success Response:**

```json
{
    "success": true,
    "coupon": {
        "id": 42,
        "code": "SUMMER25",
        "name": "Summer Sale 2024",
        "type": "percentage",
        "value": 25.0,
        "duration": "once",
        "max_uses": 100,
        "valid_until": "2024-08-31",
        "is_active": true
    }
}
```

**Error Response - Invalid Code:**

```json
{
    "error": "Invalid code format. Use only uppercase letters, numbers, hyphens, and underscores."
}
```

**Error Response - Duplicate Code:**

```json
{
    "error": "A coupon with this code already exists."
}
```

**Error Response - Invalid Percentage:**

```json
{
    "error": "Percentage value must be between 1 and 100."
}
```

---

## System Tools

### list_sites

List all sites managed by the platform.

**Description:** List all sites managed by Host Hub

**Parameters:** None

**Example Request:**

```json
{
    "tool": "list_sites",
    "arguments": {}
}
```

**Success Response:**

```json
[
    {
        "name": "BioHost",
        "domain": "link.host.uk.com",
        "type": "WordPress"
    },
    {
        "name": "SocialHost",
        "domain": "social.host.uk.com",
        "type": "Laravel"
    },
    {
        "name": "AnalyticsHost",
        "domain": "analytics.host.uk.com",
        "type": "Node.js"
    }
]
```

---

### list_routes

List all web routes in the application.

**Description:** List all web routes in the application

**Parameters:** None

**Example Request:**

```json
{
    "tool": "list_routes",
    "arguments": {}
}
```

**Success Response:**

```json
[
    {
        "uri": "/",
        "methods": ["GET", "HEAD"],
        "name": "home"
    },
    {
        "uri": "/login",
        "methods": ["GET", "HEAD"],
        "name": "login"
    },
    {
        "uri": "/api/posts",
        "methods": ["GET", "HEAD"],
        "name": "api.posts.index"
    },
    {
        "uri": "/api/posts/{post}",
        "methods": ["GET", "HEAD"],
        "name": "api.posts.show"
    }
]
```

---

### get_stats

Get current system statistics.

**Description:** Get current system statistics for Host Hub

**Parameters:** None

**Example Request:**

```json
{
    "tool": "get_stats",
    "arguments": {}
}
```

**Success Response:**

```json
{
    "total_sites": 6,
    "active_users": 128,
    "page_views_30d": 12500,
    "server_load": "23%"
}
```

---

## Common Error Responses

### Missing Workspace Context

Tools requiring workspace context return this when no API key or session is provided:

```json
{
    "error": "MCP tool 'tool_name' requires workspace context. Authenticate with an API key or user session."
}
```

**HTTP Status:** 403

### Missing Dependency

When a tool's dependencies aren't satisfied:

```json
{
    "error": "dependency_not_met",
    "message": "Dependencies not satisfied for tool 'update_task'",
    "missing": [
        {
            "type": "tool_called",
            "key": "create_plan",
            "description": "A plan must be created before updating tasks"
        }
    ],
    "suggested_order": ["create_plan", "update_task"]
}
```

**HTTP Status:** 422

### Quota Exceeded

When workspace has exceeded their tool usage quota:

```json
{
    "error": "quota_exceeded",
    "message": "Daily tool quota exceeded for this workspace",
    "current_usage": 1000,
    "limit": 1000,
    "resets_at": "2024-01-16T00:00:00+00:00"
}
```

**HTTP Status:** 429

### Validation Error

When parameters fail validation:

```json
{
    "error": "Validation failed",
    "code": "VALIDATION_ERROR",
    "details": {
        "query": ["The query field is required"]
    }
}
```

**HTTP Status:** 422

### Internal Error

When an unexpected error occurs:

```json
{
    "error": "An unexpected error occurred. Please try again.",
    "code": "INTERNAL_ERROR"
}
```

**HTTP Status:** 500

---

## Authentication

### API Key Authentication

Include your API key in the Authorization header:

```bash
curl -X POST https://api.example.com/mcp/tools/call \
    -H "Authorization: Bearer sk_live_xxxxx" \
    -H "Content-Type: application/json" \
    -d '{"tool": "get_billing_status", "arguments": {}}'
```

### Session Authentication

For browser-based access, use session cookies:

```javascript
fetch('/mcp/tools/call', {
    method: 'POST',
    credentials: 'include',
    headers: {
        'Content-Type': 'application/json',
        'X-CSRF-TOKEN': document.querySelector('meta[name="csrf-token"]').content
    },
    body: JSON.stringify({
        tool: 'list_invoices',
        arguments: { limit: 10 }
    })
});
```

### MCP Session ID

For tracking dependencies across tool calls, include a session ID:

```bash
curl -X POST https://api.example.com/mcp/tools/call \
    -H "Authorization: Bearer sk_live_xxxxx" \
    -H "X-MCP-Session-ID: session_abc123" \
    -H "Content-Type: application/json" \
    -d '{"tool": "update_task", "arguments": {...}}'
```

---

## Tool Categories

### Query Tools
- `query_database` - Execute SQL queries
- `list_tables` - List database tables

### Commerce Tools
- `get_billing_status` - Get subscription status
- `list_invoices` - List workspace invoices
- `upgrade_plan` - Change subscription plan
- `create_coupon` - Create discount codes

### System Tools
- `list_sites` - List managed sites
- `list_routes` - List application routes
- `get_stats` - Get system statistics

---

## Response Format

All tools return JSON responses. Success responses vary by tool, but error responses follow a consistent format:

```json
{
    "error": "Human-readable error message",
    "code": "ERROR_CODE",
    "details": {}  // Optional additional information
}
```

**Common Error Codes:**

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Invalid parameters |
| `FORBIDDEN_QUERY` | SQL query blocked by security |
| `MISSING_WORKSPACE_CONTEXT` | Workspace authentication required |
| `QUOTA_EXCEEDED` | Usage limit reached |
| `NOT_FOUND` | Resource not found |
| `DEPENDENCY_NOT_MET` | Tool prerequisites not satisfied |
| `INTERNAL_ERROR` | Unexpected server error |

---

## Learn More

- [Creating MCP Tools](/packages/mcp/creating-mcp-tools) - Build custom tools
- [SQL Security](/packages/mcp/sql-security) - Query security rules
- [Workspace Context](/packages/mcp/workspace) - Multi-tenant isolation
- [Quotas](/packages/mcp/quotas) - Usage limits
- [Analytics](/packages/mcp/analytics) - Usage tracking
