<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mcp\Dependencies\DependencyType;
use Core\Mcp\Dependencies\ToolDependency;
use Core\Mcp\Exceptions\MissingDependencyException;
use Illuminate\Support\Facades\Cache;

/**
 * Service for validating tool dependency graphs.
 *
 * Ensures tools have their prerequisites met before execution.
 * Tracks which tools have been called in a session and validates
 * against defined dependency rules.
 */
class ToolDependencyService
{
    /**
     * Cache key prefix for session tool history.
     */
    protected const SESSION_CACHE_PREFIX = 'mcp:session_tools:';

    /**
     * Cache TTL for session data (24 hours).
     */
    protected const SESSION_CACHE_TTL = 86400;

    /**
     * Registered tool dependencies.
     *
     * @var array<string, array<ToolDependency>>
     */
    protected array $dependencies = [];

    /**
     * Custom dependency validators.
     *
     * @var array<string, callable>
     */
    protected array $customValidators = [];

    public function __construct()
    {
        $this->registerDefaultDependencies();
    }

    /**
     * Register dependencies for a tool.
     *
     * @param  string  $toolName  The tool name
     * @param  array<ToolDependency>  $dependencies  List of dependencies
     */
    public function register(string $toolName, array $dependencies): self
    {
        $this->dependencies[$toolName] = $dependencies;

        return $this;
    }

    /**
     * Register a custom validator for CUSTOM dependency types.
     *
     * @param  string  $name  The custom dependency name
     * @param  callable  $validator  Function(array $context, array $args): bool
     */
    public function registerCustomValidator(string $name, callable $validator): self
    {
        $this->customValidators[$name] = $validator;

        return $this;
    }

    /**
     * Get dependencies for a tool.
     *
     * @return array<ToolDependency>
     */
    public function getDependencies(string $toolName): array
    {
        return $this->dependencies[$toolName] ?? [];
    }

    /**
     * Check if all dependencies are met for a tool.
     *
     * @param  string  $sessionId  The session identifier
     * @param  string  $toolName  The tool to check
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     * @return bool True if all dependencies are met
     */
    public function checkDependencies(string $sessionId, string $toolName, array $context = [], array $args = []): bool
    {
        $missing = $this->getMissingDependencies($sessionId, $toolName, $context, $args);

        return empty($missing);
    }

    /**
     * Get list of missing dependencies for a tool.
     *
     * @param  string  $sessionId  The session identifier
     * @param  string  $toolName  The tool to check
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     * @return array<ToolDependency> List of unmet dependencies
     */
    public function getMissingDependencies(string $sessionId, string $toolName, array $context = [], array $args = []): array
    {
        $dependencies = $this->getDependencies($toolName);

        if (empty($dependencies)) {
            return [];
        }

        $calledTools = $this->getCalledTools($sessionId);
        $missing = [];

        foreach ($dependencies as $dependency) {
            if ($dependency->optional) {
                continue; // Skip optional dependencies
            }

            $isMet = $this->isDependencyMet($dependency, $calledTools, $context, $args);

            if (! $isMet) {
                $missing[] = $dependency;
            }
        }

        return $missing;
    }

    /**
     * Validate dependencies and throw exception if not met.
     *
     * @param  string  $sessionId  The session identifier
     * @param  string  $toolName  The tool to validate
     * @param  array  $context  The execution context
     * @param  array  $args  The tool arguments
     *
     * @throws MissingDependencyException If dependencies are not met
     */
    public function validateDependencies(string $sessionId, string $toolName, array $context = [], array $args = []): void
    {
        $missing = $this->getMissingDependencies($sessionId, $toolName, $context, $args);

        if (! empty($missing)) {
            $suggestedOrder = $this->getSuggestedToolOrder($toolName, $missing);

            throw new MissingDependencyException($toolName, $missing, $suggestedOrder);
        }
    }

    /**
     * Record that a tool was called in a session.
     *
     * @param  string  $sessionId  The session identifier
     * @param  string  $toolName  The tool that was called
     * @param  array  $args  The arguments used (for entity tracking)
     */
    public function recordToolCall(string $sessionId, string $toolName, array $args = []): void
    {
        $key = self::SESSION_CACHE_PREFIX.$sessionId;
        $history = Cache::get($key, []);

        $history[] = [
            'tool' => $toolName,
            'args' => $args,
            'timestamp' => now()->toIso8601String(),
        ];

        Cache::put($key, $history, self::SESSION_CACHE_TTL);
    }

    /**
     * Get list of tools called in a session.
     *
     * @return array<string> Tool names that have been called
     */
    public function getCalledTools(string $sessionId): array
    {
        $key = self::SESSION_CACHE_PREFIX.$sessionId;
        $history = Cache::get($key, []);

        return array_unique(array_column($history, 'tool'));
    }

    /**
     * Get full tool call history for a session.
     *
     * @return array<array{tool: string, args: array, timestamp: string}>
     */
    public function getToolHistory(string $sessionId): array
    {
        $key = self::SESSION_CACHE_PREFIX.$sessionId;

        return Cache::get($key, []);
    }

    /**
     * Clear session tool history.
     */
    public function clearSession(string $sessionId): void
    {
        Cache::forget(self::SESSION_CACHE_PREFIX.$sessionId);
    }

    /**
     * Get the full dependency graph for visualization.
     *
     * @return array<string, array{dependencies: array, dependents: array}>
     */
    public function getDependencyGraph(): array
    {
        $graph = [];

        // Build forward dependencies
        foreach ($this->dependencies as $tool => $deps) {
            $graph[$tool] = [
                'dependencies' => array_map(fn (ToolDependency $d) => $d->toArray(), $deps),
                'dependents' => [],
            ];
        }

        // Build reverse dependencies (who depends on whom)
        foreach ($this->dependencies as $tool => $deps) {
            foreach ($deps as $dep) {
                if ($dep->type === DependencyType::TOOL_CALLED) {
                    if (! isset($graph[$dep->key])) {
                        $graph[$dep->key] = [
                            'dependencies' => [],
                            'dependents' => [],
                        ];
                    }
                    $graph[$dep->key]['dependents'][] = $tool;
                }
            }
        }

        return $graph;
    }

    /**
     * Get all tools that depend on a specific tool.
     *
     * @return array<string> Tool names that depend on the given tool
     */
    public function getDependentTools(string $toolName): array
    {
        $dependents = [];

        foreach ($this->dependencies as $tool => $deps) {
            foreach ($deps as $dep) {
                if ($dep->type === DependencyType::TOOL_CALLED && $dep->key === $toolName) {
                    $dependents[] = $tool;
                }
            }
        }

        return $dependents;
    }

    /**
     * Get all tools in dependency order (topological sort).
     *
     * @return array<string> Tools sorted by dependency order
     */
    public function getTopologicalOrder(): array
    {
        $visited = [];
        $order = [];
        $tools = array_keys($this->dependencies);

        foreach ($tools as $tool) {
            $this->topologicalVisit($tool, $visited, $order);
        }

        return $order;
    }

    /**
     * Check if a specific dependency is met.
     */
    protected function isDependencyMet(
        ToolDependency $dependency,
        array $calledTools,
        array $context,
        array $args
    ): bool {
        return match ($dependency->type) {
            DependencyType::TOOL_CALLED => in_array($dependency->key, $calledTools, true),
            DependencyType::SESSION_STATE => isset($context[$dependency->key]) && $context[$dependency->key] !== null,
            DependencyType::CONTEXT_EXISTS => array_key_exists($dependency->key, $context),
            DependencyType::ENTITY_EXISTS => $this->checkEntityExists($dependency, $args, $context),
            DependencyType::CUSTOM => $this->checkCustomDependency($dependency, $context, $args),
        };
    }

    /**
     * Check if an entity exists based on the dependency configuration.
     */
    protected function checkEntityExists(ToolDependency $dependency, array $args, array $context): bool
    {
        $entityType = $dependency->key;
        $argKey = $dependency->metadata['arg_key'] ?? null;

        if (! $argKey || ! isset($args[$argKey])) {
            return false;
        }

        // Check based on entity type
        return match ($entityType) {
            'plan' => $this->planExists($args[$argKey]),
            'session' => $this->sessionExists($args[$argKey] ?? $context['session_id'] ?? null),
            'phase' => $this->phaseExists($args['plan_slug'] ?? null, $args[$argKey] ?? null),
            default => true, // Unknown entity types pass by default
        };
    }

    /**
     * Check if a plan exists.
     */
    protected function planExists(?string $slug): bool
    {
        if (! $slug) {
            return false;
        }

        // Use a simple database check - the model namespace may vary
        return \DB::table('agent_plans')->where('slug', $slug)->exists();
    }

    /**
     * Check if a session exists.
     */
    protected function sessionExists(?string $sessionId): bool
    {
        if (! $sessionId) {
            return false;
        }

        return \DB::table('agent_sessions')->where('session_id', $sessionId)->exists();
    }

    /**
     * Check if a phase exists.
     */
    protected function phaseExists(?string $planSlug, ?string $phaseIdentifier): bool
    {
        if (! $planSlug || ! $phaseIdentifier) {
            return false;
        }

        $plan = \DB::table('agent_plans')->where('slug', $planSlug)->first();
        if (! $plan) {
            return false;
        }

        $query = \DB::table('agent_phases')->where('agent_plan_id', $plan->id);

        if (is_numeric($phaseIdentifier)) {
            return $query->where('order', (int) $phaseIdentifier)->exists();
        }

        return $query->where('name', $phaseIdentifier)->exists();
    }

    /**
     * Check a custom dependency using registered validator.
     */
    protected function checkCustomDependency(ToolDependency $dependency, array $context, array $args): bool
    {
        $validator = $this->customValidators[$dependency->key] ?? null;

        if (! $validator) {
            // No validator registered - pass by default with warning
            return true;
        }

        return call_user_func($validator, $context, $args);
    }

    /**
     * Get suggested tool order to satisfy dependencies.
     *
     * @param  array<ToolDependency>  $missing
     * @return array<string>
     */
    protected function getSuggestedToolOrder(string $targetTool, array $missing): array
    {
        $order = [];

        foreach ($missing as $dep) {
            if ($dep->type === DependencyType::TOOL_CALLED) {
                // Recursively get dependencies of the required tool
                $preDeps = $this->getDependencies($dep->key);
                foreach ($preDeps as $preDep) {
                    if ($preDep->type === DependencyType::TOOL_CALLED && ! in_array($preDep->key, $order, true)) {
                        $order[] = $preDep->key;
                    }
                }

                if (! in_array($dep->key, $order, true)) {
                    $order[] = $dep->key;
                }
            }
        }

        $order[] = $targetTool;

        return $order;
    }

    /**
     * Helper for topological sort.
     */
    protected function topologicalVisit(string $tool, array &$visited, array &$order): void
    {
        if (isset($visited[$tool])) {
            return;
        }

        $visited[$tool] = true;

        foreach ($this->getDependencies($tool) as $dep) {
            if ($dep->type === DependencyType::TOOL_CALLED) {
                $this->topologicalVisit($dep->key, $visited, $order);
            }
        }

        $order[] = $tool;
    }

    /**
     * Register default dependencies for known tools.
     */
    protected function registerDefaultDependencies(): void
    {
        // Session tools - session_log/artifact/handoff require active session
        $this->register('session_log', [
            ToolDependency::sessionState('session_id', 'Active session required. Call session_start first.'),
        ]);

        $this->register('session_artifact', [
            ToolDependency::sessionState('session_id', 'Active session required. Call session_start first.'),
        ]);

        $this->register('session_handoff', [
            ToolDependency::sessionState('session_id', 'Active session required. Call session_start first.'),
        ]);

        $this->register('session_end', [
            ToolDependency::sessionState('session_id', 'Active session required. Call session_start first.'),
        ]);

        // Plan tools - require workspace context
        $this->register('plan_create', [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ]);

        // Task tools - require plan to exist
        $this->register('task_update', [
            ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']),
        ]);

        $this->register('task_toggle', [
            ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']),
        ]);

        // Phase tools - require plan to exist
        $this->register('phase_get', [
            ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']),
        ]);

        $this->register('phase_update_status', [
            ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']),
        ]);

        $this->register('phase_add_checkpoint', [
            ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']),
        ]);

        // Content tools - require brief to exist for generation
        $this->register('content_generate', [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ]);

        $this->register('content_batch_generate', [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ]);
    }
}
