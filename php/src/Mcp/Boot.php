<?php

declare(strict_types=1);

namespace Core\Mcp;

use Core\Events\AdminPanelBooting;
use Core\Events\ConsoleBooting;
use Core\Events\McpRoutesRegistering;
use Core\Events\McpToolsRegistering;
use Core\Mcp\Events\ToolExecuted;
use Core\Mcp\Listeners\RecordToolExecution;
use Core\Mcp\Services\AuditLogService;
use Core\Mcp\Services\McpQuotaService;
use Core\Mcp\Services\QueryAuditService;
use Core\Mcp\Services\QueryExecutionService;
use Core\Mcp\Services\ToolAnalyticsService;
use Core\Mcp\Services\ToolDependencyService;
use Core\Mcp\Services\ToolRegistry;
use Core\Mcp\Services\ToolVersionService;
use Illuminate\Support\Facades\Event;
use Illuminate\Support\Facades\Route;
use Illuminate\Support\ServiceProvider;

class Boot extends ServiceProvider
{
    /**
     * The module name.
     */
    protected string $moduleName = 'mcp';

    /**
     * Events this module listens to for lazy loading.
     *
     * @var array<class-string, string>
     */
    public static array $listens = [
        AdminPanelBooting::class => 'onAdminPanel',
        ConsoleBooting::class => 'onConsole',
        McpRoutesRegistering::class => 'onMcpRoutes',
        McpToolsRegistering::class => 'onMcpTools',
    ];

    /**
     * Register any application services.
     */
    public function register(): void
    {
        $this->app->singleton(ToolRegistry::class);
        $this->app->singleton(ToolAnalyticsService::class);
        $this->app->singleton(McpQuotaService::class);
        $this->app->singleton(ToolDependencyService::class);
        $this->app->singleton(AuditLogService::class);
        $this->app->singleton(ToolVersionService::class);
        $this->app->singleton(QueryAuditService::class);
        $this->app->singleton(QueryExecutionService::class);
    }

    /**
     * Bootstrap any application services.
     */
    public function boot(): void
    {
        $this->loadMigrationsFrom(__DIR__.'/Migrations');

        // Register event listener for tool execution analytics
        Event::listen(ToolExecuted::class, RecordToolExecution::class);
    }

    // -------------------------------------------------------------------------
    // Event-driven handlers
    // -------------------------------------------------------------------------

    public function onAdminPanel(AdminPanelBooting $event): void
    {
        $event->views($this->moduleName, __DIR__.'/View/Blade');

        if (file_exists(__DIR__.'/Routes/admin.php')) {
            $event->routes(fn () => require __DIR__.'/Routes/admin.php');
        }

        $event->livewire('mcp.admin.api-key-manager', View\Modal\Admin\ApiKeyManager::class);
        $event->livewire('mcp.admin.playground', View\Modal\Admin\Playground::class);
        $event->livewire('mcp.admin.mcp-playground', View\Modal\Admin\McpPlayground::class);
        $event->livewire('mcp.admin.request-log', View\Modal\Admin\RequestLog::class);
        $event->livewire('mcp.admin.tool-analytics-dashboard', View\Modal\Admin\ToolAnalyticsDashboard::class);
        $event->livewire('mcp.admin.tool-analytics-detail', View\Modal\Admin\ToolAnalyticsDetail::class);
        $event->livewire('mcp.admin.quota-usage', View\Modal\Admin\QuotaUsage::class);
        $event->livewire('mcp.admin.audit-log-viewer', View\Modal\Admin\AuditLogViewer::class);
        $event->livewire('mcp.admin.tool-version-manager', View\Modal\Admin\ToolVersionManager::class);
    }

    public function onMcpRoutes(McpRoutesRegistering $event): void
    {
        // Register middleware aliases
        $event->middleware('mcp.auth', Middleware\McpApiKeyAuth::class);
        $event->middleware('mcp.workspace', Middleware\ValidateWorkspaceContext::class);
        $event->middleware('mcp.authenticate', Middleware\McpAuthenticate::class);
        $event->middleware('mcp.quota', Middleware\CheckMcpQuota::class);
        $event->middleware('mcp.dependencies', Middleware\ValidateToolDependencies::class);

        $domain = config('mcp.domain');

        $event->routes(fn () => Route::domain($domain)
            ->middleware('mcp.auth')
            ->name('mcp.')
            ->group(function () {
                Route::post('tools/call', [Controllers\McpApiController::class, 'callTool'])->name('tools.call');
                Route::get('resources/{uri}', [Controllers\McpApiController::class, 'resource'])->name('resources.read')
                    ->where('uri', '.+');
                Route::get('servers.json', [Controllers\McpApiController::class, 'servers'])->name('servers.json');
                Route::get('servers/{id}.json', [Controllers\McpApiController::class, 'server'])->name('servers.json.show')
                    ->where('id', '[a-z0-9-]+');
                Route::get('servers/{id}/tools', [Controllers\McpApiController::class, 'tools'])->name('servers.tools')
                    ->where('id', '[a-z0-9-]+');
                Route::get('servers/{id}/resources', [Controllers\McpApiController::class, 'resources'])->name('servers.resources')
                    ->where('id', '[a-z0-9-]+');
            })
        );
    }

    public function onConsole(ConsoleBooting $event): void
    {
        // Middleware aliases for CLI context (artisan route:list etc.)
        $event->middleware('mcp.auth', Middleware\McpApiKeyAuth::class);
        $event->middleware('mcp.workspace', Middleware\ValidateWorkspaceContext::class);
        $event->middleware('mcp.authenticate', Middleware\McpAuthenticate::class);
        $event->middleware('mcp.quota', Middleware\CheckMcpQuota::class);
        $event->middleware('mcp.dependencies', Middleware\ValidateToolDependencies::class);

        $event->command(Console\Commands\McpAgentServerCommand::class);
        $event->command(Console\Commands\PruneMetricsCommand::class);
        $event->command(Console\Commands\VerifyAuditLogCommand::class);
    }

    public function onMcpTools(McpToolsRegistering $event): void
    {
        // MCP tool handlers will be registered here once extracted
        // from the monolithic McpAgentServerCommand
    }
}
