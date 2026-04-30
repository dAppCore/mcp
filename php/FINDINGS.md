# Phase 0 Findings: Environment Assessment

**Date:** 2026-02-21
**Branch:** feat/phase-0-assessment
**Issue:** #1

---

## 1. Environment Assessment

### Composer Install

**Result:** FAILED

```
Your requirements could not be resolved to an installable set of packages.
Problem 1
  - Root composer.json requires host-uk/core, it could not be found in any version
```

`host-uk/core` is a private proprietary package (the Core PHP framework). It is not
published to Packagist or a configured private registry accessible in this environment.

**Impact:** Tests, lint, and static analysis cannot be executed without the vendor
directory. All tooling assessment below is based on static code review.

---

## 2. Test Baseline

**Status:** Unable to run (no vendor directory)

**Configured test runner:** Pest (`vendor/bin/pest`)

**Test file inventory:**

| File | Suite | Status |
|------|-------|--------|
| `tests/Unit/SqlQueryValidatorTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/WorkspaceContextSecurityTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/ToolAnalyticsServiceTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/McpQuotaServiceTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/QueryAuditServiceTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/QueryExecutionServiceTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/ToolDependencyServiceTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/ToolVersionServiceTest.php` | Unit | Present |
| `src/Mcp/Tests/Unit/ValidateWorkspaceContextMiddlewareTest.php` | Unit | Present |
| `src/Mcp/Tests/UseCase/ApiKeyManagerBasic.php` | UseCase | Present |

**Notable gaps:**
- No test for `QueryDatabase` tool (the primary entry point)
- No test for `ToolRegistry` / `AgentToolRegistry` service
- No test for `McpAgentServerCommand` (stdio MCP server)
- No test for `AuditLogService` (tamper-evidence verification)
- No test for `CircuitBreaker` service
- No integration tests at all (`tests/Feature/` is empty)

---

## 3. Lint Baseline

**Tool:** `vendor/bin/pint`
**Status:** Unable to run (no vendor directory)

**Static observation:** All reviewed files contain `declare(strict_types=1)` at the top
and follow PSR-12 conventions. Consistent UK English spelling observed throughout
(colour, organisation, licence, sanitise, normalise).

---

## 4. Static Analysis Baseline

**Tool:** `vendor/bin/phpstan` (level unknown — not declared in composer.json)
**Status:** Unable to run (no vendor directory)

**Observations from code review:**

### Type Safety
- All public methods have complete parameter and return type hints
- Private methods are consistently typed
- `SqlQueryValidator::$whitelist` is `array` — could be `array<int, string>` for PHPStan level 5+
- `Boot::$listens` uses `array<class-string, string>` — correct

### Potential PHPStan Issues (estimated level 5)
1. `QueryDatabase::getWorkspaceId()` calls `workspace()` global helper — not declared in stubs
2. `QueryDatabase::getUserId()` calls `auth()->id()` — return type is `int|string|null`, cast to int without null check
3. `QueryDatabase::interpretExplain()` accesses `$rowArray['type']` on `array<string, mixed>` — likely needs type narrowing
4. `QueryDatabase::handleExplain()` passes `$explainResults` (array of stdClass) to `interpretExplain()` typed as `array` — needs `array<int, object>`
5. `Boot::onMcpTools()` has empty body — PHPStan will warn about unused parameter

---

## 5. Architecture Review

### Package Structure

```
src/Mcp/              # Core\Mcp namespace (103 PHP files)
├── Boot.php          # ServiceProvider + event-driven registration
├── Console/          # 5 Artisan commands
├── Context/          # WorkspaceContext value object
├── Controllers/      # McpApiController (REST)
├── Dependencies/     # Tool dependency system (interface + DTO + enum)
├── DTO/              # ToolStats data transfer object
├── Events/           # ToolExecuted domain event
├── Exceptions/       # 6 custom exceptions (typed hierarchy)
├── Lang/en_GB/       # UK English translations
├── Listeners/        # RecordToolExecution event listener
├── Middleware/        # 5 middleware (auth, quota, workspace context, deps)
├── Migrations/       # 5 database migrations
├── Models/           # 8 Eloquent models
├── Resources/        # AppConfig, ContentResource, DatabaseSchema
├── Routes/           # admin.php route file
├── Services/         # 18 business logic services
├── Tests/            # Unit tests co-located with package
├── Tools/            # 10 MCP tool implementations
└── View/             # 12 Blade templates + 9 Livewire components

src/Website/Mcp/      # Core\Website\Mcp namespace
└── ...               # Web-facing UI module
```

### Key Architectural Patterns

**1. Event-Driven Module Registration**
`Boot.php` uses a `$listens` static array to subscribe to framework lifecycle events
(`AdminPanelBooting`, `ConsoleBooting`, `McpToolsRegistering`). This enables lazy-loading
of admin UI, commands, and tool registrations without booting the full framework.

**2. Tool Contract**
Tools extend `Laravel\Mcp\Server\Tool` and implement:
- `$description` property
- `schema(JsonSchema $schema): array` — declares MCP input schema
- `handle(Request $request): Response` — executes tool logic

**3. Defence-in-Depth SQL Validation** (`SqlQueryValidator`)
Four sequential layers:
1. Dangerous pattern check on raw query (before comment stripping)
2. Comment stripping (removes `--`, `#`, `/* */`, `/*!` obfuscation)
3. Blocked keyword check (write/admin/export operations)
4. Whitelist regex matching (only known-safe SELECT structures pass)

**4. Workspace Tenant Isolation**
`RequiresWorkspaceContext` trait + `ValidateWorkspaceContext` middleware enforce per-request
tenant scoping. `MissingWorkspaceContextException` is thrown for unauthenticated context.

**5. Tier-Based Resource Limits**
`McpQuotaService` and `QueryExecutionService` apply different limits per subscription tier:
- free / starter / pro / business / enterprise / unlimited
- Limits cover: row count, query timeout, daily request quota

**6. Singleton Service Container**
All 8 core services registered as singletons in `Boot::register()`. Each is independently
testable and injected via Laravel's container.

### Notable Issues

**Issue A — `onMcpTools` is a stub**
`Boot::onMcpTools()` contains only a comment:
```php
public function onMcpTools(McpToolsRegistering $event): void
{
    // MCP tool handlers will be registered here once extracted
    // from the monolithic McpAgentServerCommand
}
```
This means MCP tools are registered inside `McpAgentServerCommand` rather than being
injected via the service container. Refactoring this is P1 work.

**Issue B — `McpAgentServerCommand` is monolithic**
The stdio MCP server command handles tool registration, JSON-RPC dispatch, and tool
execution in a single command class. This makes it untestable in isolation.

**Issue C — `ListTables` tool exists but Schema Exploration is listed as TODO**
`src/Mcp/Tools/ListTables.php` exists but the TODO.md item "Schema Exploration Tools"
lists adding `ListTables` as pending. This is already implemented.

**Issue D — No `composer.lock`**
No `composer.lock` file is present. Dependency versions are not pinned, which creates
reproducibility risk in CI/CD.

**Issue E — `phpunit.xml` references `vendor/phpunit/phpunit`**
The test runner is configured for PHPUnit XML format but Pest is the stated test runner.
This is compatible (Pest uses PHPUnit under the hood) but the XML namespace warning will
appear until `composer.lock` is generated.

**Issue F — `tests/Feature/` is empty**
No feature/integration tests exist. All tests are unit tests that mock the database.
End-to-end request-response flows have no test coverage.

---

## 6. Security Observations

| Finding | Severity | Status |
|---------|----------|--------|
| SQL injection prevention (multi-layer) | GOOD | Implemented |
| Read-only connection enforcement | GOOD | Implemented |
| Workspace tenant isolation | GOOD | Implemented |
| Audit trail with HMAC verification | GOOD | Implemented |
| Tier-based resource limits | GOOD | Implemented |
| Circuit breaker for external calls | GOOD | Implemented |
| Tool registration outside DI container | MEDIUM | Issue A above |
| No integration tests for auth flow | MEDIUM | Issue F above |
| Missing `composer.lock` | LOW | Issue D above |
| `INFORMATION_SCHEMA` access blocked | GOOD | Implemented |
| System table access blocked | GOOD | Implemented |

---

## 7. Phased Work Recommendations

### Phase 1 — Unblock Testing (Prerequisite)

1. Resolve `host-uk/core` dependency access (private registry credentials or mock stubs)
2. Generate `composer.lock` after successful install
3. Run `vendor/bin/pest` to establish a numerical test baseline

### Phase 2 — Critical Gaps

1. **Extract tools from `McpAgentServerCommand`** into the `McpToolsRegistering` event
   handler in `Boot::onMcpTools()` — makes the command testable
2. **Write `QueryDatabase` tool tests** — primary public surface has zero test coverage
3. **Write `AuditLogService` tests** — tamper-evident logging is security-critical
4. **Write integration tests** for the full HTTP → tool → response flow

### Phase 3 — Code Quality

1. Fix estimated PHPStan level 5 type errors (see §4)
2. Add `phpstan.neon` configuration file (currently absent)
3. Add `pint.json` configuration file (currently absent)
4. Resolve TODO.md items marked medium priority

### Phase 4 — Features

Refer to `TODO.md` for the full backlog.
