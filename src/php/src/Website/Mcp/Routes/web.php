<?php

use Illuminate\Support\Facades\Route;
use Core\Mcp\Middleware\McpAuthenticate;
use Core\Website\Mcp\Controllers\McpRegistryController;

/*
|--------------------------------------------------------------------------
| MCP Portal Routes (HTML)
|--------------------------------------------------------------------------
|
| Human-readable documentation portal for the MCP domain.
| Wrapped in 'web' middleware by Website\Mcp\Boot for sessions/CSRF (Livewire).
|
| Functional API routes (tools/call, servers.json, etc.) are registered
| via the McpRoutesRegistering lifecycle event in Core\Mcp\Boot.
|
*/

Route::domain(config('mcp.domain'))->name('mcp.')->group(function () {
    // Agent discovery endpoint (always JSON, no auth)
    Route::get('.well-known/mcp-servers.json', [McpRegistryController::class, 'registry'])
        ->name('registry');

    // ── Human-readable portal (optional auth) ────────────────────
    Route::get('/', [McpRegistryController::class, 'landing'])
        ->middleware(McpAuthenticate::class.':optional')
        ->name('landing');

    Route::get('servers', [McpRegistryController::class, 'index'])
        ->middleware(McpAuthenticate::class.':optional')
        ->name('servers.index');

    Route::get('servers/{id}', [McpRegistryController::class, 'show'])
        ->middleware(McpAuthenticate::class.':optional')
        ->name('servers.show')
        ->where('id', '[a-z0-9-]+');

    Route::get('connect', [McpRegistryController::class, 'connect'])
        ->middleware(McpAuthenticate::class.':optional')
        ->name('connect');

    // OpenAPI spec
    Route::get('openapi.json', [McpRegistryController::class, 'openapi'])->name('openapi.json');
    Route::get('openapi.yaml', [McpRegistryController::class, 'openapi'])->name('openapi.yaml');
});
