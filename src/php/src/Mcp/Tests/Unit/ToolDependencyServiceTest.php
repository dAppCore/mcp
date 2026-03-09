<?php

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Dependencies\DependencyType;
use Core\Mcp\Dependencies\ToolDependency;
use Core\Mcp\Exceptions\MissingDependencyException;
use Core\Mcp\Services\ToolDependencyService;
use Illuminate\Support\Facades\Cache;
use Tests\TestCase;

class ToolDependencyServiceTest extends TestCase
{
    protected ToolDependencyService $service;

    protected function setUp(): void
    {
        parent::setUp();
        $this->service = new ToolDependencyService;
        Cache::flush();
    }

    public function test_can_register_dependencies(): void
    {
        $deps = [
            ToolDependency::toolCalled('plan_create'),
            ToolDependency::contextExists('workspace_id'),
        ];

        $this->service->register('custom_tool', $deps);

        $registered = $this->service->getDependencies('custom_tool');

        $this->assertCount(2, $registered);
        $this->assertSame('plan_create', $registered[0]->key);
        $this->assertSame(DependencyType::TOOL_CALLED, $registered[0]->type);
    }

    public function test_returns_empty_for_unregistered_tool(): void
    {
        $deps = $this->service->getDependencies('nonexistent_tool');

        $this->assertEmpty($deps);
    }

    public function test_check_dependencies_passes_when_no_deps(): void
    {
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'tool_without_deps',
            context: [],
            args: [],
        );

        $this->assertTrue($result);
    }

    public function test_check_dependencies_fails_when_tool_not_called(): void
    {
        $this->service->register('dependent_tool', [
            ToolDependency::toolCalled('required_tool'),
        ]);

        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'dependent_tool',
            context: [],
            args: [],
        );

        $this->assertFalse($result);
    }

    public function test_check_dependencies_passes_after_tool_called(): void
    {
        $this->service->register('dependent_tool', [
            ToolDependency::toolCalled('required_tool'),
        ]);

        // Record the required tool call
        $this->service->recordToolCall('test-session', 'required_tool');

        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'dependent_tool',
            context: [],
            args: [],
        );

        $this->assertTrue($result);
    }

    public function test_check_context_exists_dependency(): void
    {
        $this->service->register('workspace_tool', [
            ToolDependency::contextExists('workspace_id'),
        ]);

        // Without workspace_id
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'workspace_tool',
            context: [],
            args: [],
        );
        $this->assertFalse($result);

        // With workspace_id
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'workspace_tool',
            context: ['workspace_id' => 123],
            args: [],
        );
        $this->assertTrue($result);
    }

    public function test_check_session_state_dependency(): void
    {
        $this->service->register('session_tool', [
            ToolDependency::sessionState('session_id'),
        ]);

        // Without session_id
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'session_tool',
            context: [],
            args: [],
        );
        $this->assertFalse($result);

        // With null session_id (should still fail)
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'session_tool',
            context: ['session_id' => null],
            args: [],
        );
        $this->assertFalse($result);

        // With valid session_id
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'session_tool',
            context: ['session_id' => 'ses_123'],
            args: [],
        );
        $this->assertTrue($result);
    }

    public function test_get_missing_dependencies(): void
    {
        $this->service->register('multi_dep_tool', [
            ToolDependency::toolCalled('tool_a'),
            ToolDependency::toolCalled('tool_b'),
            ToolDependency::contextExists('workspace_id'),
        ]);

        // Record one tool call
        $this->service->recordToolCall('test-session', 'tool_a');

        $missing = $this->service->getMissingDependencies(
            sessionId: 'test-session',
            toolName: 'multi_dep_tool',
            context: [],
            args: [],
        );

        $this->assertCount(2, $missing);
        $this->assertSame('tool_b', $missing[0]->key);
        $this->assertSame('workspace_id', $missing[1]->key);
    }

    public function test_validate_dependencies_throws_exception(): void
    {
        $this->service->register('validated_tool', [
            ToolDependency::toolCalled('required_tool', 'You must call required_tool first'),
        ]);

        $this->expectException(MissingDependencyException::class);
        $this->expectExceptionMessage('Cannot execute \'validated_tool\'');

        $this->service->validateDependencies(
            sessionId: 'test-session',
            toolName: 'validated_tool',
            context: [],
            args: [],
        );
    }

    public function test_validate_dependencies_passes_when_met(): void
    {
        $this->service->register('validated_tool', [
            ToolDependency::toolCalled('required_tool'),
        ]);

        $this->service->recordToolCall('test-session', 'required_tool');

        // Should not throw
        $this->service->validateDependencies(
            sessionId: 'test-session',
            toolName: 'validated_tool',
            context: [],
            args: [],
        );

        $this->assertTrue(true); // No exception means pass
    }

    public function test_optional_dependencies_are_skipped(): void
    {
        $this->service->register('soft_dep_tool', [
            ToolDependency::toolCalled('hard_req'),
            ToolDependency::toolCalled('soft_req')->asOptional(),
        ]);

        $this->service->recordToolCall('test-session', 'hard_req');

        // Should pass even though soft_req not called
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'soft_dep_tool',
            context: [],
            args: [],
        );

        $this->assertTrue($result);
    }

    public function test_record_and_get_tool_call_history(): void
    {
        $this->service->recordToolCall('test-session', 'tool_a', ['arg1' => 'value1']);
        $this->service->recordToolCall('test-session', 'tool_b');
        $this->service->recordToolCall('test-session', 'tool_a', ['arg1' => 'value2']);

        $calledTools = $this->service->getCalledTools('test-session');

        $this->assertCount(2, $calledTools);
        $this->assertContains('tool_a', $calledTools);
        $this->assertContains('tool_b', $calledTools);

        $history = $this->service->getToolHistory('test-session');

        $this->assertCount(3, $history);
        $this->assertSame('tool_a', $history[0]['tool']);
        $this->assertSame(['arg1' => 'value1'], $history[0]['args']);
    }

    public function test_clear_session(): void
    {
        $this->service->recordToolCall('test-session', 'tool_a');

        $this->assertNotEmpty($this->service->getCalledTools('test-session'));

        $this->service->clearSession('test-session');

        $this->assertEmpty($this->service->getCalledTools('test-session'));
    }

    public function test_get_dependency_graph(): void
    {
        $this->service->register('tool_a', []);
        $this->service->register('tool_b', [
            ToolDependency::toolCalled('tool_a'),
        ]);
        $this->service->register('tool_c', [
            ToolDependency::toolCalled('tool_b'),
        ]);

        $graph = $this->service->getDependencyGraph();

        $this->assertArrayHasKey('tool_a', $graph);
        $this->assertArrayHasKey('tool_b', $graph);
        $this->assertArrayHasKey('tool_c', $graph);

        // tool_b depends on tool_a
        $this->assertContains('tool_b', $graph['tool_a']['dependents']);

        // tool_c depends on tool_b
        $this->assertContains('tool_c', $graph['tool_b']['dependents']);
    }

    public function test_get_dependent_tools(): void
    {
        $this->service->register('base_tool', []);
        $this->service->register('dep_tool_1', [
            ToolDependency::toolCalled('base_tool'),
        ]);
        $this->service->register('dep_tool_2', [
            ToolDependency::toolCalled('base_tool'),
        ]);

        $dependents = $this->service->getDependentTools('base_tool');

        $this->assertCount(2, $dependents);
        $this->assertContains('dep_tool_1', $dependents);
        $this->assertContains('dep_tool_2', $dependents);
    }

    public function test_get_topological_order(): void
    {
        $this->service->register('tool_a', []);
        $this->service->register('tool_b', [
            ToolDependency::toolCalled('tool_a'),
        ]);
        $this->service->register('tool_c', [
            ToolDependency::toolCalled('tool_b'),
        ]);

        $order = $this->service->getTopologicalOrder();

        $indexA = array_search('tool_a', $order);
        $indexB = array_search('tool_b', $order);
        $indexC = array_search('tool_c', $order);

        $this->assertLessThan($indexB, $indexA);
        $this->assertLessThan($indexC, $indexB);
    }

    public function test_custom_validator(): void
    {
        $this->service->register('custom_validated_tool', [
            ToolDependency::custom('has_permission', 'User must have admin permission'),
        ]);

        // Register custom validator that checks for admin role
        $this->service->registerCustomValidator('has_permission', function ($context, $args) {
            return ($context['role'] ?? null) === 'admin';
        });

        // Without admin role
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'custom_validated_tool',
            context: ['role' => 'user'],
            args: [],
        );
        $this->assertFalse($result);

        // With admin role
        $result = $this->service->checkDependencies(
            sessionId: 'test-session',
            toolName: 'custom_validated_tool',
            context: ['role' => 'admin'],
            args: [],
        );
        $this->assertTrue($result);
    }

    public function test_suggested_tool_order(): void
    {
        $this->service->register('tool_a', []);
        $this->service->register('tool_b', [
            ToolDependency::toolCalled('tool_a'),
        ]);
        $this->service->register('tool_c', [
            ToolDependency::toolCalled('tool_b'),
        ]);

        try {
            $this->service->validateDependencies(
                sessionId: 'test-session',
                toolName: 'tool_c',
                context: [],
                args: [],
            );
            $this->fail('Should have thrown MissingDependencyException');
        } catch (MissingDependencyException $e) {
            $this->assertContains('tool_a', $e->suggestedOrder);
            $this->assertContains('tool_b', $e->suggestedOrder);
            $this->assertContains('tool_c', $e->suggestedOrder);

            // Verify order
            $indexA = array_search('tool_a', $e->suggestedOrder);
            $indexB = array_search('tool_b', $e->suggestedOrder);
            $this->assertLessThan($indexB, $indexA);
        }
    }

    public function test_session_isolation(): void
    {
        $this->service->register('isolated_tool', [
            ToolDependency::toolCalled('prereq'),
        ]);

        // Record in session 1
        $this->service->recordToolCall('session-1', 'prereq');

        // Session 1 should pass
        $result1 = $this->service->checkDependencies(
            sessionId: 'session-1',
            toolName: 'isolated_tool',
            context: [],
            args: [],
        );
        $this->assertTrue($result1);

        // Session 2 should fail (different session)
        $result2 = $this->service->checkDependencies(
            sessionId: 'session-2',
            toolName: 'isolated_tool',
            context: [],
            args: [],
        );
        $this->assertFalse($result2);
    }

    public function test_missing_dependency_exception_api_response(): void
    {
        $missing = [
            ToolDependency::toolCalled('tool_a', 'Tool A must be called first'),
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];

        $exception = new MissingDependencyException(
            toolName: 'target_tool',
            missingDependencies: $missing,
            suggestedOrder: ['tool_a', 'target_tool'],
        );

        $response = $exception->toApiResponse();

        $this->assertSame('dependency_not_met', $response['error']);
        $this->assertSame('target_tool', $response['tool']);
        $this->assertCount(2, $response['missing_dependencies']);
        $this->assertSame(['tool_a', 'target_tool'], $response['suggested_order']);
        $this->assertArrayHasKey('help', $response);
    }

    public function test_default_dependencies_registered(): void
    {
        // The service should have default dependencies registered
        $sessionLogDeps = $this->service->getDependencies('session_log');

        $this->assertNotEmpty($sessionLogDeps);
        $this->assertSame(DependencyType::SESSION_STATE, $sessionLogDeps[0]->type);
        $this->assertSame('session_id', $sessionLogDeps[0]->key);
    }

    public function test_tool_dependency_factory_methods(): void
    {
        $toolCalled = ToolDependency::toolCalled('some_tool');
        $this->assertSame(DependencyType::TOOL_CALLED, $toolCalled->type);
        $this->assertSame('some_tool', $toolCalled->key);

        $sessionState = ToolDependency::sessionState('session_key');
        $this->assertSame(DependencyType::SESSION_STATE, $sessionState->type);

        $contextExists = ToolDependency::contextExists('context_key');
        $this->assertSame(DependencyType::CONTEXT_EXISTS, $contextExists->type);

        $entityExists = ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']);
        $this->assertSame(DependencyType::ENTITY_EXISTS, $entityExists->type);
        $this->assertSame('plan_slug', $entityExists->metadata['arg_key']);

        $custom = ToolDependency::custom('custom_check', 'Custom validation');
        $this->assertSame(DependencyType::CUSTOM, $custom->type);
    }

    public function test_tool_dependency_to_and_from_array(): void
    {
        $original = ToolDependency::toolCalled('some_tool', 'Must call first')
            ->asOptional();

        $array = $original->toArray();

        $this->assertSame('tool_called', $array['type']);
        $this->assertSame('some_tool', $array['key']);
        $this->assertTrue($array['optional']);

        $restored = ToolDependency::fromArray($array);

        $this->assertSame($original->type, $restored->type);
        $this->assertSame($original->key, $restored->key);
        $this->assertSame($original->optional, $restored->optional);
    }
}
