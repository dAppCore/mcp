# Discovery Scan — 2026-02-21

Automated discovery scan performed for issue #2.

## Issues Created

### Test Coverage (12 issues)
- #4 — test: add tests for ToolRegistry service
- #5 — test: add tests for AuditLogService
- #6 — test: add tests for CircuitBreaker service
- #7 — test: add tests for DataRedactor service
- #8 — test: add tests for McpHealthService
- #9 — test: add tests for McpMetricsService
- #10 — test: add tests for McpWebhookDispatcher
- #11 — test: add tests for OpenApiGenerator
- #12 — test: add tests for ToolRateLimiter
- #13 — test: add tests for AgentSessionService
- #14 — test: add tests for AgentToolRegistry
- #15 — test: add integration tests for QueryDatabase tool

### Refactoring (4 issues)
- #16 — refactor: extract SQL parser from regex to AST-based validation
- #17 — refactor: standardise tool responses with ToolResult DTO
- #18 — refactor: fix PHPStan level 5 type errors across services
- #19 — refactor: extract McpToolsRegistering tool registration from McpAgentServerCommand

### Infrastructure / Chores (4 issues)
- #20 — chore: create missing ToolRegistry YAML server definition files
- #21 — chore: add PHPStan and static analysis to dev dependencies
- #22 — chore: add CI/CD security regression tests
- #31 — chore: add query result streaming for large result sets

### Features (6 issues)
- #23 — feat: add query template system
- #24 — feat: add schema exploration tools (ListTables, DescribeTable, ListIndexes)
- #25 — feat: add data export tool (CSV, JSON)
- #26 — feat: add query result caching
- #32 — feat: add query history tracking per workspace
- #33 — feat: add data validation tool for database quality checks

### Security (3 issues)
- #27 — security: add monitoring and alerting for suspicious query patterns
- #28 — security: review ContentTools for injection and data exposure risks
- #29 — security: review commerce tools for payment data exposure

### Documentation (1 issue)
- #30 — docs: add inline documentation for ContentTools and commerce tools

### Roadmap (1 issue)
- #34 — roadmap: php-mcp production readiness

**Total: 31 issues created**
