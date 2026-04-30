<?php

declare(strict_types=1);

/**
 * Unit: ValidateWorkspaceContext Middleware
 *
 * Tests for the MCP workspace context validation middleware.
 */

use Core\Tenant\Models\User;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;
use Mod\Mcp\Context\WorkspaceContext;
use Mod\Mcp\Middleware\ValidateWorkspaceContext;

describe('ValidateWorkspaceContext Middleware', function () {
    beforeEach(function () {
        $this->middleware = new ValidateWorkspaceContext;
        $this->user = User::factory()->create();
        $this->workspace = Workspace::factory()->create();
        $this->workspace->users()->attach($this->user->id, [
            'role' => 'owner',
            'is_default' => true,
        ]);
    });

    it('sets workspace context when mcp_workspace attribute exists', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace', $this->workspace);

        $contextSet = null;
        $response = $this->middleware->handle($request, function ($request) use (&$contextSet) {
            $contextSet = $request->attributes->get('mcp_workspace_context');

            return response()->json(['success' => true]);
        });

        expect($contextSet)->toBeInstanceOf(WorkspaceContext::class);
        expect($contextSet->workspaceId)->toBe($this->workspace->id);
        expect($response->getStatusCode())->toBe(200);
    });

    it('rejects requests without workspace when mode is required', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->headers->set('Accept', 'application/json');

        $response = $this->middleware->handle($request, function () {
            return response()->json(['success' => true]);
        }, 'required');

        expect($response->getStatusCode())->toBe(403);

        $data = json_decode($response->getContent(), true);
        expect($data['error'])->toBe('missing_workspace_context');
    });

    it('allows requests without workspace when mode is optional', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');

        $response = $this->middleware->handle($request, function ($request) {
            $context = $request->attributes->get('mcp_workspace_context');

            return response()->json(['has_context' => $context !== null]);
        }, 'optional');

        expect($response->getStatusCode())->toBe(200);

        $data = json_decode($response->getContent(), true);
        expect($data['has_context'])->toBeFalse();
    });

    it('extracts workspace from authenticated user', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->setUserResolver(fn () => $this->user);

        $contextSet = null;
        $response = $this->middleware->handle($request, function ($request) use (&$contextSet) {
            $contextSet = $request->attributes->get('mcp_workspace_context');

            return response()->json(['success' => true]);
        });

        expect($contextSet)->toBeInstanceOf(WorkspaceContext::class);
        expect($contextSet->workspaceId)->toBe($this->workspace->id);
    });

    it('defaults to required mode', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->headers->set('Accept', 'application/json');

        $response = $this->middleware->handle($request, function () {
            return response()->json(['success' => true]);
        });

        expect($response->getStatusCode())->toBe(403);
    });

    it('returns HTML response for non-API requests', function () {
        $request = Request::create('/mcp/tools', 'GET');
        // Not setting Accept: application/json

        $response = $this->middleware->handle($request, function () {
            return response()->json(['success' => true]);
        }, 'required');

        expect($response->getStatusCode())->toBe(403);
        expect($response->headers->get('Content-Type'))->not->toContain('application/json');
    });
});
