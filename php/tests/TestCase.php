<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Tests;

use Core\Front\Mcp\Boot as McpBoot;
use Orchestra\Testbench\TestCase as BaseTestCase;

/**
 * Base test case for the core-mcp package.
 *
 * Testbench, not Illuminate\Foundation\Testing\TestCase: the latter expects a
 * host application to have booted already and gives a package suite no app at
 * all, so every test died before reaching its assertions. Testbench builds the
 * minimal application this package needs.
 */
abstract class TestCase extends BaseTestCase
{
    /**
     * Register the package's own service provider.
     *
     * Testbench does not run Laravel's package auto-discovery, so the provider
     * named in composer.json's extra.laravel.providers has to be given here or
     * none of the package's bindings, routes or config exist under test.
     */
    protected function getPackageProviders($app): array
    {
        return [
            McpBoot::class,
        ];
    }

    /**
     * Testbench boots a bare app with no APP_KEY, and the encrypter is reached
     * through the session middleware every rendered view runs through, so
     * without a key the failure surfaces as a ViewException rather than as the
     * behaviour under test. A fixed key, not a random one: it keeps runs
     * reproducible and nothing here encrypts data that outlives the test.
     * Exactly 32 bytes — AES-256-CBC rejects any other length.
     */
    protected function defineEnvironment($app): void
    {
        $app['config']->set('app.key', 'base64:'.base64_encode('core-mcp-testing-key-32-bytes!!!'));
    }

    /**
     * Run both packages' migrations against the in-memory database.
     *
     * Boot.php publishes the package's migrations for a host application to
     * run rather than loading them itself, so under Testbench there is no
     * schema at all unless the directories are named here. Without this the
     * suite failed 180 of its 187 tests on "no such table" — workspaces and
     * users from php-tenant, mcp_tool_metrics and mcp_tool_versions from this
     * package — none of which is the behaviour any of those tests assert.
     *
     * php-tenant first: the mcp tables carry workspace_id foreign keys.
     */
    protected function defineDatabaseMigrations(): void
    {
        $this->loadMigrationsFrom(dirname(__DIR__, 2).'/vendor/dappcore/php-tenant/Migrations');
        $this->loadMigrationsFrom(dirname(__DIR__).'/src/Mcp/Migrations');
    }
}
