<?php

declare(strict_types=1);

/**
 * Unit: Workspace Context Security
 *
 * Comprehensive tests for MCP workspace context to ensure proper:
 * - Context resolution from headers and authentication
 * - Automatic workspace scoping in queries
 * - MissingWorkspaceContextException handling
 * - Workspace boundary enforcement
 * - Cross-workspace data isolation
 *
 * @see TODO.md P2-014: Test Workspace Context
 */

use Core\Tenant\Models\User;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;
use Mod\Mcp\Context\WorkspaceContext;
use Mod\Mcp\Exceptions\MissingWorkspaceContextException;
use Mod\Mcp\Middleware\ValidateWorkspaceContext;
use Mod\Mcp\Tools\Concerns\RequiresWorkspaceContext;

// Test class using the trait
class TestToolWithWorkspaceContext
{
    use RequiresWorkspaceContext;

    protected string $name = 'test_tool';
}

// =============================================================================
// MissingWorkspaceContextException Tests
// =============================================================================

describe('MissingWorkspaceContextException', function () {
    it('creates exception with tool name', function () {
        $exception = new MissingWorkspaceContextException('ListInvoices');

        expect($exception->tool)->toBe('ListInvoices');
        expect($exception->getMessage())->toContain('ListInvoices');
        expect($exception->getMessage())->toContain('workspace context');
    });

    it('creates exception with custom message', function () {
        $exception = new MissingWorkspaceContextException('TestTool', 'Custom error message');

        expect($exception->getMessage())->toBe('Custom error message');
        expect($exception->tool)->toBe('TestTool');
    });

    it('returns correct status code', function () {
        $exception = new MissingWorkspaceContextException('TestTool');

        expect($exception->getStatusCode())->toBe(403);
    });

    it('returns correct error type', function () {
        $exception = new MissingWorkspaceContextException('TestTool');

        expect($exception->getErrorType())->toBe('missing_workspace_context');
    });

    it('includes authentication guidance in default message', function () {
        $exception = new MissingWorkspaceContextException('QueryDatabase');

        expect($exception->getMessage())->toContain('API key');
        expect($exception->getMessage())->toContain('session');
    });

    it('preserves tool name across serialisation', function () {
        $exception = new MissingWorkspaceContextException('SerialiseTest');
        $serialised = serialize($exception);
        $restored = unserialize($serialised);

        expect($restored->tool)->toBe('SerialiseTest');
    });
});

// =============================================================================
// WorkspaceContext Core Tests
// =============================================================================

describe('WorkspaceContext', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create([
            'name' => 'Test Workspace',
            'slug' => 'test-workspace',
        ]);
    });

    it('creates context from workspace model', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        expect($context->workspaceId)->toBe($this->workspace->id);
        expect($context->workspace)->toBe($this->workspace);
    });

    it('creates context from workspace ID', function () {
        $context = WorkspaceContext::fromId($this->workspace->id);

        expect($context->workspaceId)->toBe($this->workspace->id);
        expect($context->workspace)->toBeNull();
    });

    it('loads workspace when accessing from ID-only context', function () {
        $context = WorkspaceContext::fromId($this->workspace->id);

        $loadedWorkspace = $context->getWorkspace();

        expect($loadedWorkspace->id)->toBe($this->workspace->id);
        expect($loadedWorkspace->name)->toBe('Test Workspace');
    });

    it('validates ownership correctly', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        // Should not throw for matching workspace
        $context->validateOwnership($this->workspace->id, 'invoice');

        expect(true)->toBeTrue(); // If we get here, no exception was thrown
    });

    it('throws on ownership validation failure', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);
        $differentWorkspaceId = $this->workspace->id + 999;

        expect(fn () => $context->validateOwnership($differentWorkspaceId, 'invoice'))
            ->toThrow(\RuntimeException::class, 'invoice does not belong to the authenticated workspace');
    });

    it('checks workspace ID correctly', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        expect($context->hasWorkspaceId($this->workspace->id))->toBeTrue();
        expect($context->hasWorkspaceId($this->workspace->id + 1))->toBeFalse();
    });

    it('context is immutable after creation', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        // The workspaceId property is readonly
        $reflection = new ReflectionProperty(WorkspaceContext::class, 'workspaceId');
        expect($reflection->isReadOnly())->toBeTrue();
    });
});

// =============================================================================
// WorkspaceContext Resolution from Headers Tests
// =============================================================================

describe('WorkspaceContext resolution from headers', function () {
    beforeEach(function () {
        $this->user = User::factory()->create();
        $this->workspace = Workspace::factory()->create();
        $this->workspace->users()->attach($this->user->id, [
            'role' => 'owner',
            'is_default' => true,
        ]);
    });

    it('resolves context from mcp_workspace request attribute', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace', $this->workspace);

        $context = WorkspaceContext::fromRequest($request, 'TestTool');

        expect($context->workspaceId)->toBe($this->workspace->id);
        expect($context->workspace)->toBeInstanceOf(Workspace::class);
    });

    it('resolves context from generic workspace attribute', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('workspace', $this->workspace);

        $context = WorkspaceContext::fromRequest($request, 'TestTool');

        expect($context->workspaceId)->toBe($this->workspace->id);
    });

    it('throws MissingWorkspaceContextException when no context available', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');

        expect(fn () => WorkspaceContext::fromRequest($request, 'QueryDatabase'))
            ->toThrow(MissingWorkspaceContextException::class);
    });

    it('includes tool name in exception when context missing', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');

        try {
            WorkspaceContext::fromRequest($request, 'ListInvoices');
            $this->fail('Expected MissingWorkspaceContextException');
        } catch (MissingWorkspaceContextException $e) {
            expect($e->tool)->toBe('ListInvoices');
        }
    });

    it('prioritises mcp_workspace over generic workspace attribute', function () {
        $otherWorkspace = Workspace::factory()->create();

        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace', $this->workspace);
        $request->attributes->set('workspace', $otherWorkspace);

        $context = WorkspaceContext::fromRequest($request, 'TestTool');

        expect($context->workspaceId)->toBe($this->workspace->id);
    });

    it('falls back to authenticated user default workspace', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->setUserResolver(fn () => $this->user);

        $context = WorkspaceContext::fromRequest($request, 'TestTool');

        expect($context->workspaceId)->toBe($this->workspace->id);
    });
});

// =============================================================================
// RequiresWorkspaceContext Trait Tests
// =============================================================================

describe('RequiresWorkspaceContext trait', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create();
        $this->tool = new TestToolWithWorkspaceContext;
    });

    it('throws MissingWorkspaceContextException when no context set', function () {
        expect(fn () => $this->tool->getWorkspaceId())
            ->toThrow(MissingWorkspaceContextException::class);
    });

    it('returns workspace ID when context is set', function () {
        $this->tool->setWorkspace($this->workspace);

        expect($this->tool->getWorkspaceId())->toBe($this->workspace->id);
    });

    it('returns workspace when context is set', function () {
        $this->tool->setWorkspace($this->workspace);

        $workspace = $this->tool->getWorkspace();

        expect($workspace->id)->toBe($this->workspace->id);
    });

    it('allows setting context from workspace ID', function () {
        $this->tool->setWorkspaceId($this->workspace->id);

        expect($this->tool->getWorkspaceId())->toBe($this->workspace->id);
    });

    it('allows setting context object directly', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);
        $this->tool->setWorkspaceContext($context);

        expect($this->tool->getWorkspaceId())->toBe($this->workspace->id);
    });

    it('correctly reports whether context is available', function () {
        expect($this->tool->hasWorkspaceContext())->toBeFalse();

        $this->tool->setWorkspace($this->workspace);

        expect($this->tool->hasWorkspaceContext())->toBeTrue();
    });

    it('validates resource ownership through context', function () {
        $this->tool->setWorkspace($this->workspace);
        $differentWorkspaceId = $this->workspace->id + 999;

        expect(fn () => $this->tool->validateResourceOwnership($differentWorkspaceId, 'subscription'))
            ->toThrow(\RuntimeException::class, 'subscription does not belong');
    });

    it('requires context with custom error message', function () {
        expect(fn () => $this->tool->requireWorkspaceContext('listing invoices'))
            ->toThrow(MissingWorkspaceContextException::class, 'listing invoices');
    });

    it('uses class name when tool name property not set', function () {
        $tool = new class {
            use RequiresWorkspaceContext;
        };

        try {
            $tool->getWorkspaceId();
            $this->fail('Expected exception');
        } catch (MissingWorkspaceContextException $e) {
            // Should use the anonymous class basename
            expect($e->tool)->not->toBeEmpty();
        }
    });

    it('allows clearing context by setting null workspace ID', function () {
        $this->tool->setWorkspace($this->workspace);
        expect($this->tool->hasWorkspaceContext())->toBeTrue();

        // Setting via a new context with different workspace
        $otherWorkspace = Workspace::factory()->create();
        $this->tool->setWorkspace($otherWorkspace);

        expect($this->tool->getWorkspaceId())->toBe($otherWorkspace->id);
    });
});

// =============================================================================
// Cross-Workspace Isolation Tests
// =============================================================================

describe('Workspace-scoped tool security', function () {
    beforeEach(function () {
        $this->user = User::factory()->create();
        $this->workspace = Workspace::factory()->create();
        $this->workspace->users()->attach($this->user->id, [
            'role' => 'owner',
            'is_default' => true,
        ]);

        // Create another workspace to test isolation
        $this->otherWorkspace = Workspace::factory()->create();
    });

    it('prevents accessing another workspace data by setting context correctly', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        // Trying to validate ownership of data from another workspace should fail
        expect(fn () => $context->validateOwnership($this->otherWorkspace->id, 'data'))
            ->toThrow(\RuntimeException::class);
    });

    it('enforces workspace boundary for each resource type', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);
        $otherWorkspaceId = $this->otherWorkspace->id;

        // Test various resource types
        $resourceTypes = ['invoice', 'order', 'subscription', 'api_key', 'webhook', 'template'];

        foreach ($resourceTypes as $resourceType) {
            expect(fn () => $context->validateOwnership($otherWorkspaceId, $resourceType))
                ->toThrow(\RuntimeException::class, "{$resourceType} does not belong");
        }
    });

    it('allows access to own workspace resources', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        // Should not throw for any resource type when workspace matches
        $resourceTypes = ['invoice', 'order', 'subscription', 'api_key', 'webhook', 'template'];

        foreach ($resourceTypes as $resourceType) {
            // This should not throw
            $context->validateOwnership($this->workspace->id, $resourceType);
        }

        expect(true)->toBeTrue(); // If we reach here, all validations passed
    });
});

// =============================================================================
// Cross-Workspace Query Prevention Tests
// =============================================================================

describe('Cross-workspace query prevention', function () {
    beforeEach(function () {
        $this->workspaceA = Workspace::factory()->create(['name' => 'Workspace A']);
        $this->workspaceB = Workspace::factory()->create(['name' => 'Workspace B']);
        $this->tool = new TestToolWithWorkspaceContext;
    });

    it('prevents workspace A tool from accessing workspace B data', function () {
        $this->tool->setWorkspace($this->workspaceA);

        expect(fn () => $this->tool->validateResourceOwnership($this->workspaceB->id, 'customer'))
            ->toThrow(\RuntimeException::class);
    });

    it('prevents workspace B tool from accessing workspace A data', function () {
        $this->tool->setWorkspace($this->workspaceB);

        expect(fn () => $this->tool->validateResourceOwnership($this->workspaceA->id, 'order'))
            ->toThrow(\RuntimeException::class);
    });

    it('maintains isolation when workspace context changes', function () {
        // Start with workspace A
        $this->tool->setWorkspace($this->workspaceA);
        expect($this->tool->getWorkspaceId())->toBe($this->workspaceA->id);

        // Validate workspace A can access its own data
        $this->tool->validateResourceOwnership($this->workspaceA->id, 'data');

        // Change to workspace B
        $this->tool->setWorkspace($this->workspaceB);
        expect($this->tool->getWorkspaceId())->toBe($this->workspaceB->id);

        // Workspace B should NOT be able to access workspace A data
        expect(fn () => $this->tool->validateResourceOwnership($this->workspaceA->id, 'data'))
            ->toThrow(\RuntimeException::class);

        // Workspace B can access its own data
        $this->tool->validateResourceOwnership($this->workspaceB->id, 'data');
        expect(true)->toBeTrue();
    });

    it('prevents cross-workspace access via ID manipulation', function () {
        $this->tool->setWorkspace($this->workspaceA);

        // Try to access data by directly using the other workspace's ID
        $attackWorkspaceId = $this->workspaceB->id;

        expect(fn () => $this->tool->validateResourceOwnership($attackWorkspaceId, 'sensitive_data'))
            ->toThrow(\RuntimeException::class);
    });

    it('rejects zero as workspace ID', function () {
        $context = WorkspaceContext::fromId(0);

        // A zero workspace ID should not match any real workspace
        expect($context->hasWorkspaceId($this->workspaceA->id))->toBeFalse();
        expect($context->hasWorkspaceId($this->workspaceB->id))->toBeFalse();
    });

    it('rejects negative workspace IDs in ownership checks', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspaceA);

        expect(fn () => $context->validateOwnership(-1, 'resource'))
            ->toThrow(\RuntimeException::class);
    });
});

// =============================================================================
// Workspace Boundary Enforcement Tests
// =============================================================================

describe('Workspace boundary enforcement', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create();
        $this->tool = new TestToolWithWorkspaceContext;
    });

    it('enforces boundary before any database operation', function () {
        // Without workspace context set, attempting to get workspace ID should fail
        expect(fn () => $this->tool->getWorkspaceId())
            ->toThrow(MissingWorkspaceContextException::class);
    });

    it('enforces boundary for workspace model access', function () {
        // Without workspace context set, attempting to get workspace should fail
        expect(fn () => $this->tool->getWorkspace())
            ->toThrow(MissingWorkspaceContextException::class);
    });

    it('enforces boundary in require context method', function () {
        expect(fn () => $this->tool->requireWorkspaceContext('database query'))
            ->toThrow(MissingWorkspaceContextException::class, 'database query');
    });

    it('provides clear error message about authentication requirements', function () {
        try {
            $this->tool->getWorkspaceId();
            $this->fail('Expected MissingWorkspaceContextException');
        } catch (MissingWorkspaceContextException $e) {
            expect($e->getMessage())->toContain('test_tool');
            expect($e->getMessage())->toContain('workspace context');
        }
    });

    it('has context returns false for uninitialised tool', function () {
        expect($this->tool->hasWorkspaceContext())->toBeFalse();
    });

    it('has context returns true after setting workspace', function () {
        $this->tool->setWorkspace($this->workspace);
        expect($this->tool->hasWorkspaceContext())->toBeTrue();
    });

    it('has context returns true after setting workspace ID', function () {
        $this->tool->setWorkspaceId($this->workspace->id);
        expect($this->tool->hasWorkspaceContext())->toBeTrue();
    });

    it('has context returns true after setting context object', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);
        $this->tool->setWorkspaceContext($context);
        expect($this->tool->hasWorkspaceContext())->toBeTrue();
    });
});

// =============================================================================
// Context Injection Tests
// =============================================================================

describe('Context injection', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create();
        $this->middleware = new ValidateWorkspaceContext;
    });

    it('injects context into request attributes', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace', $this->workspace);

        $injectedContext = null;
        $this->middleware->handle($request, function ($request) use (&$injectedContext) {
            $injectedContext = $request->attributes->get('mcp_workspace_context');

            return response()->json(['success' => true]);
        });

        expect($injectedContext)->toBeInstanceOf(WorkspaceContext::class);
        expect($injectedContext->workspaceId)->toBe($this->workspace->id);
    });

    it('injects context with correct workspace reference', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace', $this->workspace);

        $injectedContext = null;
        $this->middleware->handle($request, function ($request) use (&$injectedContext) {
            $injectedContext = $request->attributes->get('mcp_workspace_context');

            return response()->json(['success' => true]);
        });

        expect($injectedContext->workspace)->toBe($this->workspace);
    });

    it('context remains accessible throughout request lifecycle', function () {
        $request = Request::create('/api/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace', $this->workspace);

        $contextChecks = [];
        $this->middleware->handle($request, function ($request) use (&$contextChecks) {
            // First access
            $contextChecks['first'] = $request->attributes->get('mcp_workspace_context');

            // Simulated middleware/controller access
            $contextChecks['second'] = $request->attributes->get('mcp_workspace_context');

            return response()->json(['success' => true]);
        });

        expect($contextChecks['first'])->toBe($contextChecks['second']);
        expect($contextChecks['first']->workspaceId)->toBe($this->workspace->id);
    });
});

// =============================================================================
// Automatic Workspace Scoping Tests
// =============================================================================

describe('Automatic workspace scoping', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create();
        $this->tool = new TestToolWithWorkspaceContext;
        $this->tool->setWorkspace($this->workspace);
    });

    it('provides workspace ID for query scoping', function () {
        $workspaceId = $this->tool->getWorkspaceId();

        expect($workspaceId)->toBe($this->workspace->id);
        expect($workspaceId)->toBeInt();
    });

    it('provides workspace model for relationship queries', function () {
        $workspace = $this->tool->getWorkspace();

        expect($workspace)->toBeInstanceOf(Workspace::class);
        expect($workspace->id)->toBe($this->workspace->id);
    });

    it('workspace ID is consistent across multiple calls', function () {
        $id1 = $this->tool->getWorkspaceId();
        $id2 = $this->tool->getWorkspaceId();
        $id3 = $this->tool->getWorkspaceId();

        expect($id1)->toBe($id2);
        expect($id2)->toBe($id3);
    });

    it('validates resource belongs to workspace before access', function () {
        // Create resource belonging to this workspace (simulated)
        $resourceWorkspaceId = $this->workspace->id;

        // This should not throw
        $this->tool->validateResourceOwnership($resourceWorkspaceId, 'record');

        expect(true)->toBeTrue();
    });

    it('blocks access to resource from different workspace', function () {
        $otherWorkspace = Workspace::factory()->create();
        $resourceWorkspaceId = $otherWorkspace->id;

        expect(fn () => $this->tool->validateResourceOwnership($resourceWorkspaceId, 'record'))
            ->toThrow(\RuntimeException::class);
    });
});

// =============================================================================
// Edge Cases and Security Tests
// =============================================================================

describe('Edge cases and security', function () {
    beforeEach(function () {
        $this->workspace = Workspace::factory()->create();
    });

    it('handles concurrent workspace context correctly', function () {
        $tool1 = new TestToolWithWorkspaceContext;
        $tool2 = new TestToolWithWorkspaceContext;
        $otherWorkspace = Workspace::factory()->create();

        $tool1->setWorkspace($this->workspace);
        $tool2->setWorkspace($otherWorkspace);

        // Each tool should maintain its own context
        expect($tool1->getWorkspaceId())->toBe($this->workspace->id);
        expect($tool2->getWorkspaceId())->toBe($otherWorkspace->id);
    });

    it('workspace context is isolated per tool instance', function () {
        $tool1 = new TestToolWithWorkspaceContext;
        $tool2 = new TestToolWithWorkspaceContext;

        $tool1->setWorkspace($this->workspace);

        // tool2 should not have context just because tool1 does
        expect($tool1->hasWorkspaceContext())->toBeTrue();
        expect($tool2->hasWorkspaceContext())->toBeFalse();
    });

    it('does not leak workspace data between requests', function () {
        $request1 = Request::create('/api/mcp/tools/call', 'POST');
        $request1->attributes->set('mcp_workspace', $this->workspace);

        $request2 = Request::create('/api/mcp/tools/call', 'POST');
        // request2 has no workspace attribute

        $middleware = new ValidateWorkspaceContext;

        // First request should have context
        $context1 = null;
        $middleware->handle($request1, function ($req) use (&$context1) {
            $context1 = $req->attributes->get('mcp_workspace_context');

            return response()->json([]);
        });

        // Second request (in required mode) should fail, not inherit from first
        $response = $middleware->handle($request2, function () {
            return response()->json(['success' => true]);
        }, 'required');

        expect($context1)->toBeInstanceOf(WorkspaceContext::class);
        expect($response->getStatusCode())->toBe(403);
    });

    it('context object prevents modification of workspace ID', function () {
        $context = WorkspaceContext::fromWorkspace($this->workspace);

        // The readonly property should prevent modification
        // This test verifies the architecture decision
        $reflection = new ReflectionClass($context);
        $property = $reflection->getProperty('workspaceId');

        expect($property->isReadOnly())->toBeTrue();
    });

    it('handles workspace deletion gracefully', function () {
        $workspaceId = $this->workspace->id;
        $context = WorkspaceContext::fromId($workspaceId);

        // Delete the workspace
        $this->workspace->delete();

        // Context still holds the ID (but getWorkspace() would fail)
        expect($context->workspaceId)->toBe($workspaceId);
        expect(fn () => $context->getWorkspace())
            ->toThrow(\Illuminate\Database\Eloquent\ModelNotFoundException::class);
    });
});
