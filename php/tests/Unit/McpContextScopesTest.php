<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Front\Mcp\McpContext;
use Illuminate\Http\Request;
use Tests\TestCase;

/**
 * Scope resolution on McpContext.
 *
 * These came across with the methods from dappcore/agent, which declared this
 * same FQCN in its own tree. Coverage follows ownership: agent testing a class
 * it no longer owns proves nothing about the copy a consumer actually loads.
 */
class McpContextScopesTest extends TestCase
{
    /**
     * A session-like object exposing an API key with permissions, which is one
     * of the shapes scopeCandidates() walks.
     */
    private function scopeSession(array $scopes = []): object
    {
        return new class($scopes)
        {
            public function __construct(private readonly array $scopes) {}

            public function getApiKey(): object
            {
                return (object) ['permissions' => $this->scopes];
            }
        };
    }

    public function test_get_scopes_good_returns_session_scopes(): void
    {
        $context = new McpContext(scopeSource: $this->scopeSession([
            'brain.remember',
            'brain.recall',
        ]));

        $this->assertSame(['brain.remember', 'brain.recall'], $context->getScopes());
    }

    public function test_get_scopes_ugly_defaults_to_an_empty_array_for_an_empty_session(): void
    {
        $this->assertSame([], (new McpContext(scopeSource: $this->scopeSession([])))->getScopes());
    }

    public function test_get_scopes_ugly_is_empty_when_no_source_is_given_at_all(): void
    {
        // No scope source, no plan, no request attribute — the resolver must
        // report nothing rather than reaching for something that is not there.
        $this->assertSame([], (new McpContext)->getScopes());
    }

    public function test_has_scope_good_returns_true_for_a_present_scope(): void
    {
        $context = new McpContext(scopeSource: $this->scopeSession([
            'brain.remember',
            'brain.recall',
        ]));

        $this->assertTrue($context->hasScope('brain.recall'));
    }

    public function test_has_scope_bad_returns_false_for_a_missing_scope(): void
    {
        $context = new McpContext(scopeSource: $this->scopeSession(['brain.remember']));

        $this->assertFalse($context->hasScope('brain.forget'));
    }

    public function test_get_scopes_good_reads_scopes_off_the_authenticated_request(): void
    {
        // The HTTP transport puts the resolved workspace context on the request
        // rather than passing a scope source, so the resolver checks there too.
        $request = Request::create('/api/v1/mcp/tools/call', 'POST');
        $request->attributes->set('mcp_workspace_context', [
            'workspace_id' => 1,
            'api_key' => (object) ['permissions' => ['brain.remember', 'brain.recall']],
        ]);

        $original = app()->bound('request') ? app('request') : null;
        app()->instance('request', $request);

        try {
            $this->assertSame(['brain.remember', 'brain.recall'], (new McpContext)->getScopes());
        } finally {
            if ($original !== null) {
                app()->instance('request', $original);
            }
        }
    }
}
