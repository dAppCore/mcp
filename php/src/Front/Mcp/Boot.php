<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Front\Mcp;

use Core\LifecycleEventProvider;
use Illuminate\Foundation\Configuration\Middleware;
use Illuminate\Support\ServiceProvider;

/**
 * MCP frontage - MCP API stage.
 *
 * Provides mcp middleware group for MCP protocol routes.
 * Authentication middleware should be added by the core-mcp package.
 */
class Boot extends ServiceProvider
{
    /**
     * Configure mcp middleware group.
     */
    public static function middleware(Middleware $middleware): void
    {
        $middleware->group('mcp', [
            \Illuminate\Routing\Middleware\ThrottleRequests::class.':api',
            \Illuminate\Routing\Middleware\SubstituteBindings::class,
        ]);
    }

    public function register(): void
    {
        //
    }

    public function boot(): void
    {
        // Fire McpRoutesRegistering event for lazy-loaded modules
        LifecycleEventProvider::fireMcpRoutes();

        // Fire McpToolsRegistering so modules can register tool handlers
        LifecycleEventProvider::fireMcpTools();
    }
}
