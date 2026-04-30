<?php

/**
 * UseCase: MCP API Key Manager (Basic Flow)
 *
 * Acceptance test for the MCP admin panel.
 * Tests the primary admin flow through the API key manager.
 */

use Core\Tenant\Models\User;
use Core\Tenant\Models\Workspace;

describe('MCP API Key Manager', function () {
    beforeEach(function () {
        // Create user with workspace
        $this->user = User::factory()->create([
            'email' => 'test@example.com',
            'password' => bcrypt('password'),
        ]);

        $this->workspace = Workspace::factory()->create();
        $this->workspace->users()->attach($this->user->id, [
            'role' => 'owner',
            'is_default' => true,
        ]);
    });

    it('can view the API key manager page with all sections', function () {
        // Login and navigate to MCP keys page
        $this->actingAs($this->user);

        $response = $this->get(route('mcp.keys'));

        $response->assertOk();

        // Verify page title and description
        $response->assertSee(__('mcp::mcp.keys.title'));
        $response->assertSee(__('mcp::mcp.keys.description'));

        // Verify empty state when no keys exist
        $response->assertSee(__('mcp::mcp.keys.empty.title'));
        $response->assertSee(__('mcp::mcp.keys.empty.description'));

        // Verify action buttons
        $response->assertSee(__('mcp::mcp.keys.actions.create'));
    });

    it('can view the playground page', function () {
        $this->actingAs($this->user);

        $response = $this->get(route('mcp.playground'));

        $response->assertOk();

        // Verify page title and description
        $response->assertSee(__('mcp::mcp.playground.title'));
        $response->assertSee(__('mcp::mcp.playground.description'));
    });

    it('can view the request log page', function () {
        $this->actingAs($this->user);

        $response = $this->get(route('mcp.logs'));

        $response->assertOk();

        // Verify page title and description
        $response->assertSee(__('mcp::mcp.logs.title'));
        $response->assertSee(__('mcp::mcp.logs.description'));
    });
});
