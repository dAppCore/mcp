<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Exceptions\ForbiddenQueryException;

/**
 * Validates SQL queries for security before execution.
 *
 * Implements multiple layers of defence:
 * 1. Keyword blocking - Prevents dangerous SQL operations
 * 2. Structure validation - Detects injection patterns
 * 3. Whitelist matching - Only allows known-safe query patterns
 */
class SqlQueryValidator
{
    /**
     * SQL keywords that are never allowed in queries.
     * These represent write operations or dangerous constructs.
     */
    private const BLOCKED_KEYWORDS = [
        // Data modification
        'INSERT',
        'UPDATE',
        'DELETE',
        'REPLACE',
        'TRUNCATE',
        'DROP',
        'ALTER',
        'CREATE',
        'RENAME',
        // Permission/admin
        'GRANT',
        'REVOKE',
        'FLUSH',
        'KILL',
        'RESET',
        'PURGE',
        // Data export
        'INTO OUTFILE',
        'INTO DUMPFILE',
        'LOAD_FILE',
        'LOAD DATA',
        // Execution
        'EXECUTE',
        'EXEC',
        'PREPARE',
        'DEALLOCATE',
        'CALL',
        // Variables/settings
        'SET ',
    ];

    /**
     * Patterns that indicate injection attempts.
     * These are checked BEFORE comment stripping to catch obfuscation attempts.
     */
    private const DANGEROUS_PATTERNS = [
        // Stacked queries (semicolon followed by anything)
        '/;\s*\S/i',
        // UNION-based injection (with optional comment obfuscation)
        '/\bUNION\b/i',
        '/UNION/i', // Also catch UNION without word boundaries (comment-obfuscated)
        // Hex encoding to bypass filters
        '/0x[0-9a-f]+/i',
        // CHAR() function often used in injection
        '/\bCHAR\s*\(/i',
        // BENCHMARK for time-based attacks
        '/\bBENCHMARK\s*\(/i',
        // SLEEP for time-based attacks
        '/\bSLEEP\s*\(/i',
        // Information schema access (could be allowed with whitelist)
        '/\bINFORMATION_SCHEMA\b/i',
        // System tables
        '/\bmysql\./i',
        '/\bperformance_schema\./i',
        '/\bsys\./i',
        // Subquery in WHERE that could leak data
        '/WHERE\s+.*\(\s*SELECT/i',
        // Comment obfuscation attempts (inline comments between keywords)
        '/\/\*[^*]*\*\/\s*(?:UNION|SELECT|INSERT|UPDATE|DELETE|DROP)/i',
    ];

    /**
     * Default whitelist patterns for safe queries.
     * These are regex patterns that match allowed query structures.
     *
     * WHERE clause restrictions:
     * - Only allows column = value, column != value, column > value, etc.
     * - Supports AND/OR logical operators
     * - Allows LIKE, IN, BETWEEN, IS NULL/NOT NULL operators
     * - No subqueries (no nested SELECT)
     * - No function calls except common safe ones
     */
    private const DEFAULT_WHITELIST = [
        // Simple SELECT from single table with optional WHERE
        '/^\s*SELECT\s+[\w\s,.*`]+\s+FROM\s+`?\w+`?(\s+WHERE\s+[\w\s`.,!=<>\'"%()]+(\s+(AND|OR)\s+[\w\s`.,!=<>\'"%()]+)*)?(\s+ORDER\s+BY\s+[\w\s,`]+(\s+(ASC|DESC))?)?(\s+LIMIT\s+\d+(\s*,\s*\d+)?)?;?\s*$/i',
        // COUNT queries
        '/^\s*SELECT\s+COUNT\s*\(\s*\*?\s*\)\s+FROM\s+`?\w+`?(\s+WHERE\s+[\w\s`.,!=<>\'"%()]+(\s+(AND|OR)\s+[\w\s`.,!=<>\'"%()]+)*)?;?\s*$/i',
        // SELECT with explicit column list
        '/^\s*SELECT\s+`?\w+`?(\s*,\s*`?\w+`?)*\s+FROM\s+`?\w+`?(\s+WHERE\s+[\w\s`.,!=<>\'"%()]+(\s+(AND|OR)\s+[\w\s`.,!=<>\'"%()]+)*)?(\s+ORDER\s+BY\s+[\w\s,`]+)?(\s+LIMIT\s+\d+)?;?\s*$/i',
    ];

    private array $whitelist;

    private bool $useWhitelist;

    public function __construct(
        ?array $whitelist = null,
        bool $useWhitelist = true
    ) {
        $this->whitelist = $whitelist ?? self::DEFAULT_WHITELIST;
        $this->useWhitelist = $useWhitelist;
    }

    /**
     * Validate a SQL query for safety.
     *
     * @throws ForbiddenQueryException If the query fails validation
     */
    public function validate(string $query): void
    {
        // Check for dangerous patterns on the ORIGINAL query first
        // This catches attempts to obfuscate keywords with comments
        $this->checkDangerousPatterns($query);

        // Now normalise and continue validation
        $query = $this->normaliseQuery($query);

        $this->checkBlockedKeywords($query);
        $this->checkQueryStructure($query);

        if ($this->useWhitelist) {
            $this->checkWhitelist($query);
        }
    }

    /**
     * Check if a query is valid without throwing.
     */
    public function isValid(string $query): bool
    {
        try {
            $this->validate($query);

            return true;
        } catch (ForbiddenQueryException) {
            return false;
        }
    }

    /**
     * Add a pattern to the whitelist.
     */
    public function addWhitelistPattern(string $pattern): self
    {
        $this->whitelist[] = $pattern;

        return $this;
    }

    /**
     * Replace the entire whitelist.
     */
    public function setWhitelist(array $patterns): self
    {
        $this->whitelist = $patterns;

        return $this;
    }

    /**
     * Enable or disable whitelist checking.
     */
    public function setUseWhitelist(bool $use): self
    {
        $this->useWhitelist = $use;

        return $this;
    }

    /**
     * Normalise the query for consistent validation.
     */
    private function normaliseQuery(string $query): string
    {
        // Remove SQL comments
        $query = $this->stripComments($query);

        // Normalise whitespace
        $query = preg_replace('/\s+/', ' ', $query);

        return trim($query);
    }

    /**
     * Strip SQL comments which could be used to bypass filters.
     */
    private function stripComments(string $query): string
    {
        // Remove -- style comments
        $query = preg_replace('/--.*$/m', '', $query);

        // Remove # style comments
        $query = preg_replace('/#.*$/m', '', $query);

        // Remove /* */ style comments (including multi-line)
        $query = preg_replace('/\/\*.*?\*\//s', '', $query);

        // Remove /*! MySQL-specific comments that execute code
        $query = preg_replace('/\/\*!.*?\*\//s', '', $query);

        return $query;
    }

    /**
     * Check for blocked SQL keywords.
     *
     * @throws ForbiddenQueryException
     */
    private function checkBlockedKeywords(string $query): void
    {
        $upperQuery = strtoupper($query);

        foreach (self::BLOCKED_KEYWORDS as $keyword) {
            // Use word boundary check for most keywords
            $pattern = '/\b'.preg_quote($keyword, '/').'\b/i';

            if (preg_match($pattern, $query)) {
                throw ForbiddenQueryException::disallowedKeyword($query, $keyword);
            }
        }
    }

    /**
     * Check for dangerous patterns that indicate injection.
     *
     * @throws ForbiddenQueryException
     */
    private function checkDangerousPatterns(string $query): void
    {
        foreach (self::DANGEROUS_PATTERNS as $pattern) {
            if (preg_match($pattern, $query)) {
                throw ForbiddenQueryException::invalidStructure(
                    $query,
                    'Query contains potentially malicious pattern'
                );
            }
        }
    }

    /**
     * Check basic query structure.
     *
     * @throws ForbiddenQueryException
     */
    private function checkQueryStructure(string $query): void
    {
        // Must start with SELECT
        if (! preg_match('/^\s*SELECT\b/i', $query)) {
            throw ForbiddenQueryException::invalidStructure(
                $query,
                'Query must begin with SELECT'
            );
        }

        // Check for multiple statements (stacked queries)
        // After stripping comments, there should be at most one semicolon at the end
        $semicolonCount = substr_count($query, ';');
        if ($semicolonCount > 1) {
            throw ForbiddenQueryException::invalidStructure(
                $query,
                'Multiple statements detected'
            );
        }

        if ($semicolonCount === 1 && ! preg_match('/;\s*$/', $query)) {
            throw ForbiddenQueryException::invalidStructure(
                $query,
                'Semicolon only allowed at end of query'
            );
        }
    }

    /**
     * Check if query matches at least one whitelist pattern.
     *
     * @throws ForbiddenQueryException
     */
    private function checkWhitelist(string $query): void
    {
        foreach ($this->whitelist as $pattern) {
            if (preg_match($pattern, $query)) {
                return; // Query matches a whitelisted pattern
            }
        }

        throw ForbiddenQueryException::notWhitelisted($query);
    }
}
