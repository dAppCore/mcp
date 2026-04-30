<?php

use Core\Mcp\View\Modal\Admin\ApiKeyManager;
use Core\Mcp\View\Modal\Admin\AuditLogViewer;
use Core\Mcp\View\Modal\Admin\McpPlayground;
use Core\Mcp\View\Modal\Admin\Playground;
use Core\Mcp\View\Modal\Admin\QuotaUsage;
use Core\Mcp\View\Modal\Admin\RequestLog;
use Core\Mcp\View\Modal\Admin\ToolAnalyticsDashboard;
use Core\Mcp\View\Modal\Admin\ToolAnalyticsDetail;
use Core\Mcp\View\Modal\Admin\ToolVersionManager;
use Core\Website\Mcp\Controllers\McpRegistryController;
use Core\Website\Mcp\View\Modal\Dashboard;
use Illuminate\Support\Facades\Route;

/*
|--------------------------------------------------------------------------
| MCP Admin Routes
|--------------------------------------------------------------------------
|
| Protected routes for MCP portal management.
| Requires authentication via admin middleware.
|
*/

Route::prefix('mcp')->name('mcp.')->group(function () {
    // Dashboard (workspace MCP usage overview)
    Route::get('dashboard', Dashboard::class)
        ->name('dashboard');

    // API key management
    Route::get('keys', ApiKeyManager::class)
        ->name('keys');

    // Enhanced MCP Playground with tool browser, history, and examples
    Route::get('playground', McpPlayground::class)
        ->name('playground');

    // Legacy simple playground (API-key focused)
    Route::get('playground/simple', Playground::class)
        ->name('playground.simple');

    // Request log for debugging
    Route::get('logs', RequestLog::class)
        ->name('logs');

    // Analytics endpoints
    Route::get('servers/{id}/analytics', [McpRegistryController::class, 'analytics'])
        ->name('servers.analytics');

    // Tool Usage Analytics Dashboard
    Route::get('analytics', ToolAnalyticsDashboard::class)
        ->name('analytics');

    // Single tool analytics detail
    Route::get('analytics/tool/{name}', ToolAnalyticsDetail::class)
        ->name('analytics.tool');

    // Audit log viewer (compliance and security)
    Route::get('audit-log', AuditLogViewer::class)
        ->name('audit-log');

    // Tool version management (Hades only)
    Route::get('versions', ToolVersionManager::class)
        ->name('versions');

    // Quota usage overview
    Route::get('quotas', QuotaUsage::class)
        ->name('quotas');
});
