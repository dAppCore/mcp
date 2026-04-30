# Core-MCP TODO

> See [FINDINGS.md](FINDINGS.md) for the full Phase 0 environment assessment report.

## Phase 0 — Environment Blockers (February 2026)

- [ ] **Resolve `host-uk/core` dependency access**
  - Package is not available via Packagist; private registry credentials needed
  - Blocks: `composer install`, all tests, lint, and static analysis
  - **Action:** Configure private composer repository or provide mock stubs

- [ ] **Generate `composer.lock`** after successful install
  - Currently absent — dependency versions are unpinned
  - Reproducibility risk in CI/CD

- [ ] **Establish numeric test baseline**
  - Run `vendor/bin/pest` and record pass/fail counts
  - Targeted after dependency access is resolved

- [ ] **Run PHPStan analysis**
  - `vendor/bin/phpstan analyse --memory-limit=512M`
  - No `phpstan.neon` config file present — needs creating

- [ ] **Run lint baseline**
  - `vendor/bin/pint --test`
  - No `pint.json` config file present — needs creating

## Phase 1 — Critical Architecture (Prerequisite)

- [ ] **Refactor: Extract tools from `McpAgentServerCommand`**
  - MCP tools are registered inside the command, not via DI container
  - Implement `Boot::onMcpTools()` handler (currently a stub)
  - Enables unit testing of individual tools in isolation
  - **Files:** `src/Mcp/Boot.php`, `src/Mcp/Console/Commands/McpAgentServerCommand.php`
  - **Estimated effort:** 4-6 hours

- [ ] **Test Coverage: QueryDatabase Tool** — primary public surface has zero tests
  - Test SELECT execution, EXPLAIN analysis, connection validation
  - Test blocked keywords and injection prevention end-to-end
  - Test tier-based row limit truncation
  - Test timeout handling
  - **Files:** `tests/Unit/QueryDatabaseTest.php`
  - **Estimated effort:** 3-4 hours

- [ ] **Test Coverage: AuditLogService** — security-critical, no tests exist
  - Test HMAC tamper-evident logging
  - Test log verification command (`mcp:verify-audit-log`)
  - **Files:** `src/Mcp/Tests/Unit/AuditLogServiceTest.php`
  - **Estimated effort:** 2-3 hours

- [ ] **Add integration tests** — `tests/Feature/` is currently empty
  - Test full HTTP → tool → response flow
  - Test authentication and quota enforcement via middleware stack
  - **Estimated effort:** 4-5 hours

## Testing & Quality Assurance

### High Priority

- [x] **Test Coverage: SQL Query Validator** - Test injection prevention
  - [x] Test all forbidden SQL keywords (DROP, INSERT, UPDATE, DELETE, etc.)
  - [x] Test SQL injection attempts (UNION, boolean blinds, etc.)
  - [x] Test parameterized query validation
  - [x] Test subquery restrictions
  - [x] Test multi-statement detection
  - **Completed:** 29 January 2026
  - **File:** `tests/Unit/SqlQueryValidatorTest.php`

- [x] **Test Coverage: Workspace Context** - Test isolation and validation
  - [x] Test WorkspaceContext resolution from headers
  - [x] Test automatic workspace scoping in queries
  - [x] Test MissingWorkspaceContextException
  - [x] Test workspace boundary enforcement
  - [x] Test cross-workspace query prevention
  - **Completed:** 29 January 2026
  - **File:** `src/Mcp/Tests/Unit/WorkspaceContextSecurityTest.php`

- [x] **Test Coverage: Tool Analytics** - Test metrics tracking
  - [x] Test ToolAnalyticsService recording
  - [x] Test ToolStats DTO calculations
  - [x] Test error rate calculations
  - [x] Test daily trend aggregation
  - [x] Test reporting functions (popular tools, error-prone tools, workspace stats)
  - [x] Test tool combination tracking
  - **Completed:** 29 January 2026
  - **File:** `src/Mcp/Tests/Unit/ToolAnalyticsServiceTest.php`

- [x] **Test Coverage: Quota System** - Test limits and enforcement
  - [x] Test McpQuotaService tier limits (free, starter, pro, business, enterprise)
  - [x] Test quota exceeded detection
  - [x] Test quota reset timing
  - [x] Test workspace-scoped quotas
  - [x] Test CheckMcpQuota middleware (429 responses, headers)
  - [x] Test edge cases (concurrent requests, cache invalidation, month boundaries)
  - **Completed:** 29 January 2026
  - **File:** `src/Mcp/Tests/Unit/McpQuotaServiceTest.php`

### Medium Priority

- [ ] **Test Coverage: Tool Dependencies** - Test dependency validation
  - [ ] Test ToolDependencyService resolution
  - [ ] Test MissingDependencyException
  - [ ] Test circular dependency detection
  - [ ] Test version compatibility checking
  - **Estimated effort:** 2-3 hours

- [ ] **Test Coverage: Query Database Tool** - Test complete workflow
  - [ ] Test SELECT query execution
  - [ ] Test EXPLAIN plan analysis
  - [ ] Test connection validation
  - [ ] Test result formatting
  - [ ] Test error handling
  - **Estimated effort:** 3-4 hours

### Low Priority

- [ ] **Test Coverage: Tool Registry** - Test tool registration
  - [ ] Test AgentToolRegistry with multiple tools
  - [ ] Test tool discovery
  - [ ] Test tool metadata
  - **Estimated effort:** 2-3 hours

## Security (Critical)

### High Priority - Security Fixes Needed

- [x] **COMPLETED: Database Connection Fallback** - Throw exception instead of fallback
  - [x] Fixed to throw ForbiddenConnectionException
  - [x] No silent fallback to default connection
  - [x] Prevents accidental production data exposure
  - **Completed:** January 2026

- [x] **COMPLETED: SQL Validator Regex Strengthening** - Stricter WHERE clause validation
  - [x] Replaced permissive `.+` with restrictive character classes
  - [x] Added explicit structure validation
  - [x] Better detection of injection attempts
  - **Completed:** January 2026

### Medium Priority - Additional Security

- [x] **COMPLETED: Query Result Size Limits** - Prevent data exfiltration
  - [x] Add max_rows configuration per tier (free: 100, starter: 500, professional: 1000, enterprise: 5000, unlimited: 10000)
  - [x] Enforce result set limits via QueryExecutionService
  - [x] Return truncation warnings in response metadata
  - [x] Tests in QueryExecutionServiceTest.php
  - **Completed:** 29 January 2026
  - **Files:** `src/Mcp/Services/QueryExecutionService.php`, `src/Mcp/Exceptions/ResultSizeLimitException.php`

- [x] **COMPLETED: Query Timeout Enforcement** - Prevent resource exhaustion
  - [x] Add per-query timeout configuration per tier (free: 5s, starter: 10s, professional: 30s, enterprise: 60s, unlimited: 120s)
  - [x] Database-specific timeout application (MySQL/MariaDB, PostgreSQL, SQLite)
  - [x] Throw QueryTimeoutException on timeout
  - [x] Log timeout attempts via QueryAuditService
  - **Completed:** 29 January 2026
  - **Files:** `src/Mcp/Services/QueryExecutionService.php`, `src/Mcp/Exceptions/QueryTimeoutException.php`

- [x] **COMPLETED: Audit Logging for Queries** - Complete query audit trail
  - [x] Log all query attempts (success, blocked, timeout, error, truncated)
  - [x] Include user, workspace, query, bindings count, duration, row count
  - [x] Sanitise queries and error messages for security
  - [x] Security channel logging for blocked queries
  - [x] Session and tier context tracking
  - **Completed:** 29 January 2026
  - **Files:** `src/Mcp/Services/QueryAuditService.php`, `src/Mcp/Tests/Unit/QueryAuditServiceTest.php`

## Features & Enhancements

### High Priority

- [x] **COMPLETED: EXPLAIN Plan Analysis** - Query optimization insights
  - [x] Added `explain` parameter to QueryDatabase tool
  - [x] Returns human-readable performance analysis
  - [x] Shows index usage and optimization opportunities
  - **Completed:** January 2026

- [ ] **Feature: Query Templates** - Reusable parameterized queries
  - [ ] Create query template system
  - [ ] Support named parameters
  - [ ] Add template validation
  - [ ] Store templates per workspace
  - [ ] Test with complex queries
  - **Estimated effort:** 5-6 hours
  - **Files:** `src/Mod/Mcp/Templates/`

- [ ] **Feature: Schema Exploration Tools** - Database metadata access
  - [ ] Add ListTables tool
  - [ ] Add DescribeTable tool
  - [ ] Add ListIndexes tool
  - [ ] Respect information_schema restrictions
  - [ ] Test with multiple database types
  - **Estimated effort:** 4-5 hours
  - **Files:** `src/Mod/Mcp/Tools/Schema/`

### Medium Priority

- [ ] **Enhancement: Query Result Caching** - Cache frequent queries
  - [ ] Implement result caching with TTL
  - [ ] Add cache key generation
  - [ ] Support cache invalidation
  - [ ] Test cache hit rates
  - **Estimated effort:** 3-4 hours

- [ ] **Enhancement: Query History** - Track agent queries
  - [ ] Store query history per workspace
  - [ ] Add query rerun capability
  - [ ] Create history browser UI
  - [ ] Add favorite queries
  - **Estimated effort:** 4-5 hours
  - **Files:** `src/Mod/Mcp/History/`

- [ ] **Enhancement: Advanced Analytics** - Deeper insights
  - [ ] Add query complexity scoring
  - [ ] Track table access patterns
  - [ ] Identify slow query patterns
  - [ ] Create optimization recommendations
  - **Estimated effort:** 5-6 hours
  - **Files:** `src/Mod/Mcp/Analytics/`

### Low Priority

- [ ] **Enhancement: Multi-Database Support** - Query multiple databases
  - [ ] Support cross-database queries
  - [ ] Add database selection parameter
  - [ ] Test with MySQL, PostgreSQL, SQLite
  - **Estimated effort:** 4-5 hours

- [ ] **Enhancement: Query Builder UI** - Visual query construction
  - [ ] Create Livewire query builder component
  - [ ] Add table/column selection
  - [ ] Support WHERE clause builder
  - [ ] Generate safe SQL
  - **Estimated effort:** 8-10 hours
  - **Files:** `src/Mod/Mcp/QueryBuilder/`

## Tool Development

### High Priority

- [ ] **Tool: Create/Update Records** - Controlled data modification
  - [ ] Create InsertRecord tool with strict validation
  - [ ] Create UpdateRecord tool with WHERE requirements
  - [ ] Implement record-level permissions
  - [ ] Require explicit confirmation for modifications
  - [ ] Test with workspace scoping
  - **Estimated effort:** 6-8 hours
  - **Files:** `src/Mod/Mcp/Tools/Modify/`
  - **Note:** Requires careful security review

- [ ] **Tool: Export Data** - Export query results
  - [ ] Add ExportResults tool
  - [ ] Support CSV, JSON, Excel formats
  - [ ] Add row limits per tier
  - [ ] Implement streaming for large exports
  - **Estimated effort:** 4-5 hours
  - **Files:** `src/Mod/Mcp/Tools/Export/`

### Medium Priority

- [ ] **Tool: Analyze Performance** - Database health insights
  - [ ] Add TableStats tool (row count, size, etc.)
  - [ ] Add SlowQueries tool
  - [ ] Add IndexUsage tool
  - [ ] Create performance dashboard
  - **Estimated effort:** 5-6 hours
  - **Files:** `src/Mod/Mcp/Tools/Performance/`

- [ ] **Tool: Data Validation** - Validate data quality
  - [ ] Add ValidateData tool
  - [ ] Check for NULL values, duplicates
  - [ ] Validate foreign key integrity
  - [ ] Generate data quality report
  - **Estimated effort:** 4-5 hours
  - **Files:** `src/Mod/Mcp/Tools/Validation/`

## Documentation

- [x] **Guide: Creating MCP Tools** - Comprehensive tutorial
  - [x] Document tool interface
  - [x] Show parameter validation
  - [x] Explain workspace context
  - [x] Add dependency examples
  - [x] Include security best practices
  - **Completed:** January 2026
  - **File:** `docs/packages/mcp/creating-mcp-tools.md`

- [x] **Guide: SQL Security** - Safe query patterns
  - [x] Document allowed SQL patterns
  - [x] Show parameterized query examples
  - [x] Explain validation rules
  - [x] List forbidden operations
  - **Completed:** January 2026
  - **File:** `docs/packages/mcp/sql-security.md`

- [x] **API Reference: All MCP Tools** - Complete tool catalog
  - [x] Document each tool's parameters
  - [x] Add usage examples
  - [x] Show response formats
  - [x] Include error cases
  - **Completed:** January 2026
  - **File:** `docs/packages/mcp/tools-reference.md`

## Code Quality

- [ ] **Refactor: Extract SQL Parser** - Better query validation
  - [ ] Create proper SQL parser
  - [ ] Replace regex with AST parsing
  - [ ] Support dialect-specific syntax
  - [ ] Add comprehensive tests
  - **Estimated effort:** 8-10 hours

- [ ] **Refactor: Standardize Tool Responses** - Consistent API
  - [ ] Create ToolResult DTO
  - [ ] Standardize error responses
  - [ ] Add response metadata
  - [ ] Update all tools
  - **Estimated effort:** 3-4 hours

- [ ] **PHPStan: Fix Level 5 Errors** - Improve type safety
  - [ ] Fix property type declarations
  - [ ] Add missing return types
  - [ ] Fix array shape types
  - **Estimated effort:** 2-3 hours

## Performance

- [ ] **Optimization: Query Result Streaming** - Handle large results
  - [ ] Implement cursor-based result streaming
  - [ ] Add chunked response delivery
  - [ ] Test with millions of rows
  - **Estimated effort:** 3-4 hours

- [ ] **Optimization: Connection Pooling** - Reuse database connections
  - [ ] Implement connection pool
  - [ ] Add connection health checks
  - [ ] Test connection lifecycle
  - **Estimated effort:** 3-4 hours

## Infrastructure

- [ ] **Monitoring: Alert on Suspicious Queries** - Security monitoring
  - [ ] Detect unusual query patterns
  - [ ] Alert on potential injection attempts
  - [ ] Track query anomalies
  - [ ] Create security dashboard
  - **Estimated effort:** 4-5 hours

- [ ] **CI/CD: Add Security Regression Tests** - Prevent vulnerabilities
  - [ ] Test SQL injection prevention
  - [ ] Test workspace isolation
  - [ ] Test quota enforcement
  - [ ] Fail CI on security issues
  - **Estimated effort:** 3-4 hours

---

## Completed (January 2026)

- [x] **Test Coverage: Workspace Context** - Comprehensive Pest tests for workspace isolation and context injection (P2-014)
- [x] **Test Coverage: SQL Query Validator** - Comprehensive Pest tests for SQL injection prevention (P2-013)
- [x] **Security: Database Connection Validation** - Throws exception for invalid connections
- [x] **Security: SQL Validator Strengthening** - Stricter WHERE clause patterns
- [x] **Security: Query Result Size Limits** - Tier-based max_rows with truncation warnings (P1-007)
- [x] **Security: Query Timeout Enforcement** - Per-query timeout with database-specific implementation (P1-008)
- [x] **Security: Audit Logging for Queries** - Comprehensive logging of all query attempts (P1-009)
- [x] **Feature: EXPLAIN Plan Analysis** - Query optimization insights
- [x] **Tool Analytics System** - Complete usage tracking and metrics
- [x] **Quota System** - Tier-based limits with enforcement
- [x] **Workspace Context** - Automatic query scoping and validation
- [x] **Documentation: Creating MCP Tools Guide** - Complete tutorial with workspace context, dependencies, security
- [x] **Documentation: SQL Security Guide** - Allowed patterns, forbidden operations, injection prevention
- [x] **Documentation: MCP Tools API Reference** - All tools with parameters, examples, error handling

*See `changelog/2026/jan/` for completed features and security fixes.*
