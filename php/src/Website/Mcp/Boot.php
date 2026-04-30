<?php

declare(strict_types=1);

namespace Core\Website\Mcp;

use Illuminate\Support\Facades\Blade;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\ServiceProvider;
use Livewire\Livewire;

/**
 * MCP Host website.
 *
 * Public-facing MCP server registry and documentation portal.
 * Serves mcp.host.uk.com domain.
 */
class Boot extends ServiceProvider
{
    public function register(): void
    {
        //
    }

    public function boot(): void
    {
        $this->loadViewsFrom(__DIR__.'/View/Blade', 'mcp');

        // Register mcp layout into the layouts:: namespace
        $layoutsPath = dirname(__DIR__, 2).'/Front/View/Blade/layouts';
        $this->loadViewsFrom($layoutsPath, 'layouts');
        Blade::anonymousComponentPath($layoutsPath, 'layouts');

        $this->registerLivewireComponents();
        $this->registerRoutes();
    }

    protected function registerLivewireComponents(): void
    {
        Livewire::component('mcp.dashboard', View\Modal\Dashboard::class);
        Livewire::component('mcp.api-key-manager', View\Modal\ApiKeyManager::class);
        Livewire::component('mcp.api-explorer', View\Modal\ApiExplorer::class);
        Livewire::component('mcp.mcp-metrics', View\Modal\McpMetrics::class);
        Livewire::component('mcp.mcp-playground', View\Modal\McpPlayground::class);
        Livewire::component('mcp.playground', View\Modal\Playground::class);
        Livewire::component('mcp.request-log', View\Modal\RequestLog::class);
        Livewire::component('mcp.unified-search', View\Modal\UnifiedSearch::class);
    }

    protected function registerRoutes(): void
    {
        // HTML portal routes need web middleware (sessions, CSRF for Livewire)
        Route::middleware('web')->group(__DIR__.'/Routes/web.php');
    }
}
