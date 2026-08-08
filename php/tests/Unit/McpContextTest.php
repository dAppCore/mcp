<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Front\Mcp\Contracts\McpToolHandler;
use Core\Front\Mcp\McpContext;
use ReflectionClass;
use ReflectionMethod;
use Tests\TestCase;

/**
 * Session, plan and transport-callback behaviour on McpContext, plus the
 * McpToolHandler contract shape.
 *
 * These came across from dappcore/agent with the class itself. Agent declared
 * the same two FQCNs in its own tree and held the only tests for them, while
 * this package owns the copy a consumer actually loads — so the behaviour was
 * tested on one side and shipped from the other. Deleting agent's copy without
 * bringing these would have left the surviving code untested.
 */
class McpContextTest extends TestCase
{
    public function test_getters_good_track_the_current_session_and_plan(): void
    {
        $plan = (object) ['slug' => 'plan-1'];
        $context = new McpContext('sess-1', $plan);

        $this->assertSame('sess-1', $context->getSessionId());
        $this->assertSame($plan, $context->getCurrentPlan());
        $this->assertTrue($context->hasSession());
        $this->assertTrue($context->hasPlan());
    }

    public function test_callbacks_bad_are_optional_and_can_be_left_unset(): void
    {
        $context = new McpContext;

        // Must not throw with no transport hooks attached.
        $context->sendNotification('mcp.progress', ['value' => 50]);
        $context->logToSession('noop');

        $this->assertFalse($context->hasSession());
        $this->assertFalse($context->hasPlan());
    }

    public function test_callbacks_ugly_forward_notifications_and_session_logs(): void
    {
        $captured = [];

        $context = new McpContext(
            notificationCallback: function (string $method, array $params) use (&$captured): void {
                $captured['notification'] = [$method, $params];
            },
            logCallback: function (string $message, string $type, array $data) use (&$captured): void {
                $captured['log'] = [$message, $type, $data];
            },
        );

        $context->sendNotification('mcp.progress', ['value' => 100]);
        $context->logToSession('finished', 'info', ['ok' => true]);

        $this->assertSame(['mcp.progress', ['value' => 100]], $captured['notification']);
        $this->assertSame(['finished', 'info', ['ok' => true]], $captured['log']);
    }

    public function test_tool_handler_good_can_be_implemented_with_the_expected_shape(): void
    {
        $handler = new class implements McpToolHandler
        {
            public static function schema(): array
            {
                return [
                    'name' => 'list_posts',
                    'description' => 'List CMS posts',
                    'inputSchema' => ['type' => 'object'],
                ];
            }

            public function handle(array $args, McpContext $context): array
            {
                return ['ok' => true];
            }
        };

        $this->assertSame([
            'name' => 'list_posts',
            'description' => 'List CMS posts',
            'inputSchema' => ['type' => 'object'],
        ], $handler::schema());
    }

    public function test_tool_handler_bad_receives_the_transport_agnostic_context(): void
    {
        $handler = new class implements McpToolHandler
        {
            public static function schema(): array
            {
                return ['name' => 'ping', 'description' => 'Ping', 'inputSchema' => ['type' => 'object']];
            }

            public function handle(array $args, McpContext $context): array
            {
                return ['session_id' => $context->getSessionId()];
            }
        };

        $this->assertSame('sess-1', $handler->handle([], new McpContext('sess-1'))['session_id']);
    }

    public function test_tool_handler_ugly_exposes_exactly_the_two_contract_methods(): void
    {
        $methods = array_map(
            static fn (ReflectionMethod $method): string => $method->getName(),
            (new ReflectionClass(McpToolHandler::class))->getMethods(),
        );

        $this->assertSame(['schema', 'handle'], $methods);
    }
}
