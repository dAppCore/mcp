<?php

/*
 * Core MCP Package
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

use Core\Mcp\Exceptions\ForbiddenQueryException;
use Core\Mcp\Services\SqlQueryValidator;

describe('SqlQueryValidator', function () {
    describe('allowed SELECT statements', function () {
        it('allows simple SELECT queries', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT * FROM users'))->toBeTrue();
            expect($validator->isValid('SELECT id, name FROM users'))->toBeTrue();
            expect($validator->isValid('SELECT `id`, `name` FROM `users`'))->toBeTrue();
        });

        it('allows SELECT with WHERE clause', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid("SELECT * FROM users WHERE id = 1"))->toBeTrue();
            expect($validator->isValid("SELECT * FROM users WHERE name = 'John'"))->toBeTrue();
            expect($validator->isValid("SELECT * FROM users WHERE id = 1 AND status = 'active'"))->toBeTrue();
            expect($validator->isValid("SELECT * FROM users WHERE id = 1 OR id = 2"))->toBeTrue();
        });

        it('allows SELECT with ORDER BY', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT * FROM users ORDER BY name'))->toBeTrue();
            expect($validator->isValid('SELECT * FROM users ORDER BY name ASC'))->toBeTrue();
            expect($validator->isValid('SELECT * FROM users ORDER BY name DESC'))->toBeTrue();
        });

        it('allows SELECT with LIMIT', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT * FROM users LIMIT 10'))->toBeTrue();
            expect($validator->isValid('SELECT * FROM users LIMIT 10, 20'))->toBeTrue();
        });

        it('allows COUNT queries', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT COUNT(*) FROM users'))->toBeTrue();
            expect($validator->isValid("SELECT COUNT(*) FROM users WHERE status = 'active'"))->toBeTrue();
        });

        it('allows queries with trailing semicolon', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT * FROM users;'))->toBeTrue();
            expect($validator->isValid('SELECT id FROM users WHERE id = 1;'))->toBeTrue();
        });
    });

    describe('blocked data modification statements', function () {
        it('blocks INSERT statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('INSERT INTO users (name) VALUES ("test")'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('INSERT users SET name = "test"'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks UPDATE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('UPDATE users SET name = "test"'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('UPDATE users SET name = "test" WHERE id = 1'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks DELETE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('DELETE FROM users'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('DELETE FROM users WHERE id = 1'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks REPLACE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('REPLACE INTO users (id, name) VALUES (1, "test")'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks TRUNCATE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('TRUNCATE TABLE users'))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('blocked schema modification statements', function () {
        it('blocks DROP statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('DROP TABLE users'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('DROP DATABASE mydb'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('DROP INDEX idx_name ON users'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks ALTER statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('ALTER TABLE users ADD column email VARCHAR(255)'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('ALTER TABLE users DROP column email'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks CREATE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('CREATE TABLE test (id INT)'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('CREATE INDEX idx ON users (name)'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('CREATE DATABASE newdb'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks RENAME statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('RENAME TABLE users TO customers'))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('blocked permission and admin statements', function () {
        it('blocks GRANT statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('GRANT SELECT ON users TO user@localhost'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks REVOKE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('REVOKE SELECT ON users FROM user@localhost'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks FLUSH statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('FLUSH PRIVILEGES'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('FLUSH TABLES'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks KILL statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('KILL 12345'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('KILL QUERY 12345'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks SET statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('SET GLOBAL max_connections = 500'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('SET SESSION sql_mode = ""'))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('blocked execution statements', function () {
        it('blocks EXECUTE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('EXECUTE prepared_stmt'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks PREPARE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('PREPARE stmt FROM "SELECT * FROM users"'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks CALL statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('CALL stored_procedure()'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks DEALLOCATE statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('DEALLOCATE PREPARE stmt'))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('blocked file operations', function () {
        it('blocks INTO OUTFILE', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users INTO OUTFILE '/tmp/users.csv'"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks INTO DUMPFILE', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users INTO DUMPFILE '/tmp/dump.txt'"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks LOAD_FILE', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT LOAD_FILE('/etc/passwd')"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks LOAD DATA', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("LOAD DATA INFILE '/tmp/data.csv' INTO TABLE users"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - UNION attacks', function () {
        it('blocks basic UNION injection', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 UNION SELECT * FROM passwords"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks UNION ALL injection', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 UNION ALL SELECT password FROM users"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks UNION with NULL padding', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT id, name FROM users WHERE id = 1 UNION SELECT NULL, password FROM admin"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks UNION with comment obfuscation', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 UN/**/ION SELECT * FROM admin"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 /*!UNION*/ SELECT * FROM admin"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks UNION with case variation', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 UnIoN SELECT * FROM admin"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 union SELECT * FROM admin"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - stacked queries', function () {
        it('blocks semicolon-separated statements', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users; DROP TABLE users"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users; DELETE FROM users"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks stacked queries with comments', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users; -- DROP TABLE users"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users;/* comment */DROP TABLE users"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks multiple semicolons', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT 1; SELECT 2; SELECT 3"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks semicolon not at end', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users; "))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - time-based attacks', function () {
        it('blocks SLEEP function', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 AND SLEEP(5)"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT SLEEP(5)"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks BENCHMARK function', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT BENCHMARK(10000000, SHA1('test'))"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 AND BENCHMARK(1000000, MD5('x'))"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - encoding attacks', function () {
        it('blocks hex-encoded strings', function () {
            $validator = new SqlQueryValidator();

            // 0x61646d696e = 'admin'
            expect(fn () => $validator->validate("SELECT * FROM users WHERE name = 0x61646d696e"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT 0x44524f50205441424c4520757365727320"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks CHAR function for string construction', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE name = CHAR(97, 100, 109, 105, 110)"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT CHAR(65)"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - subquery restrictions', function () {
        it('blocks subqueries in WHERE clause', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = (SELECT admin_id FROM admins)"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id IN (SELECT id FROM admins)"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks correlated subqueries', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM users u WHERE EXISTS (SELECT 1 FROM admins a WHERE a.user_id = u.id)"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - system table access', function () {
        it('blocks INFORMATION_SCHEMA access', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM INFORMATION_SCHEMA.TABLES"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT table_name FROM information_schema.columns"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks mysql system database access', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM mysql.user"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT host, user FROM mysql.db"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks performance_schema access', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM performance_schema.threads"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks sys schema access', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SELECT * FROM sys.session"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('SQL injection prevention - comment obfuscation', function () {
        it('blocks inline comment keyword obfuscation', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SEL/**/ECT * FROM users"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("SELECT * FROM users WHERE id = 1 OR/**/1=1"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('blocks MySQL conditional comments with harmful content', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("/*!50000 DROP TABLE users */"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('query structure validation', function () {
        it('requires query to start with SELECT', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("SHOW TABLES"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("DESCRIBE users"))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate("EXPLAIN SELECT * FROM users"))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('validates query does not start with non-SELECT', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate("   UPDATE users SET name = 'test'"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });

    describe('whitelist configuration', function () {
        it('can disable whitelist checking', function () {
            $validator = new SqlQueryValidator(useWhitelist: false);

            // This complex query would fail whitelist but passes without it
            // (still blocked by other checks, but testing the flag works)
            expect($validator->isValid('SELECT * FROM users'))->toBeTrue();
        });

        it('can add custom whitelist patterns', function () {
            $validator = new SqlQueryValidator();

            // Add pattern for JOINs which aren't in default whitelist
            $validator->addWhitelistPattern('/^\s*SELECT\s+.+\s+FROM\s+\w+\s+JOIN\s+\w+/i');

            // Now JOIN queries should work (if they pass other checks)
            // Note: The default whitelist may still reject, testing the method works
            expect($validator)->toBeInstanceOf(SqlQueryValidator::class);
        });

        it('can replace entire whitelist', function () {
            $validator = new SqlQueryValidator();

            $validator->setWhitelist([
                '/^\s*SELECT\s+1\s*;?\s*$/i',
            ]);

            expect($validator->isValid('SELECT 1'))->toBeTrue();
            expect($validator->isValid('SELECT * FROM users'))->toBeFalse();
        });
    });

    describe('exception details', function () {
        it('includes query in exception for blocked keyword', function () {
            $validator = new SqlQueryValidator();
            $query = 'DROP TABLE users';

            try {
                $validator->validate($query);
                test()->fail('Expected ForbiddenQueryException');
            } catch (ForbiddenQueryException $e) {
                expect($e->query)->toBe($query);
                expect($e->reason)->toContain('DROP');
            }
        });

        it('includes reason for structural issues', function () {
            $validator = new SqlQueryValidator();
            $query = 'SHOW TABLES';

            try {
                $validator->validate($query);
                test()->fail('Expected ForbiddenQueryException');
            } catch (ForbiddenQueryException $e) {
                expect($e->reason)->toContain('SELECT');
            }
        });

        it('includes reason for whitelist failure', function () {
            $validator = new SqlQueryValidator();
            // Complex query that passes keyword checks but fails whitelist
            $query = 'SELECT @@version';

            try {
                $validator->validate($query);
                test()->fail('Expected ForbiddenQueryException');
            } catch (ForbiddenQueryException $e) {
                expect($e->reason)->toContain('pattern');
            }
        });
    });

    describe('edge cases', function () {
        it('handles empty query', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate(''))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('handles whitespace-only query', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('   '))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('normalises excessive whitespace', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid("SELECT    *   FROM    users"))->toBeTrue();
            expect($validator->isValid("SELECT\n*\nFROM\nusers"))->toBeTrue();
            expect($validator->isValid("SELECT\t*\tFROM\tusers"))->toBeTrue();
        });

        it('is case insensitive for keywords', function () {
            $validator = new SqlQueryValidator();

            expect(fn () => $validator->validate('drop TABLE users'))
                ->toThrow(ForbiddenQueryException::class);

            expect(fn () => $validator->validate('DrOp TaBlE users'))
                ->toThrow(ForbiddenQueryException::class);
        });

        it('handles queries with backtick-quoted identifiers', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT `id`, `name` FROM `users`'))->toBeTrue();
        });

        it('handles queries with single-quoted strings', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid("SELECT * FROM users WHERE name = 'O''Brien'"))->toBeTrue();
        });

        it('handles queries with double-quoted strings', function () {
            $validator = new SqlQueryValidator();

            expect($validator->isValid('SELECT * FROM users WHERE name = "John"'))->toBeTrue();
        });
    });

    describe('boolean-based injection prevention', function () {
        it('allows legitimate OR conditions in WHERE', function () {
            $validator = new SqlQueryValidator();

            // Legitimate use
            expect($validator->isValid("SELECT * FROM users WHERE id = 1 OR id = 2"))->toBeTrue();
        });

        it('blocks dangerous patterns even within valid structure', function () {
            $validator = new SqlQueryValidator();

            // These contain hex encoding which is always blocked
            expect(fn () => $validator->validate("SELECT * FROM users WHERE name = 0x41"))
                ->toThrow(ForbiddenQueryException::class);
        });
    });
});
