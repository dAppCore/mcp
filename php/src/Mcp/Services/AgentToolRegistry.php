<?php

declare(strict_types=1);

namespace Mod\Mcp\Services;

use Core\Mcp\Dependencies\HasDependencies;
use Core\Mcp\Services\ToolDependencyService;
use Illuminate\Support\Collection;
use Mod\Api\Models\ApiKey;
use Mod\Mcp\Tools\Agent\Contracts\AgentToolInterface;

/**
 * Registry for MCP Agent Server tools.
 *
 * Provides discovery, permission checking, and execution
 * of registered agent tools.
 */
class AgentToolRegistry
{
    /**
     * Registered tools indexed by name.
     *
     * @var array<string, AgentToolInterface>
     */
    protected array $tools = [];

    /**
     * Register a tool.
     *
     * If the tool implements HasDependencies, its dependencies
     * are automatically registered with the ToolDependencyService.
     */
    public function register(AgentToolInterface $tool): self
    {
        $this->tools[$tool->name()] = $tool;

        // Auto-register dependencies if tool declares them
        if ($tool instanceof HasDependencies && method_exists($tool, 'dependencies')) {
            $dependencies = $tool->dependencies();
            if (! empty($dependencies)) {
                app(ToolDependencyService::class)->register($tool->name(), $dependencies);
            }
        }

        return $this;
    }

    /**
     * Register multiple tools at once.
     *
     * @param  array<AgentToolInterface>  $tools
     */
    public function registerMany(array $tools): self
    {
        foreach ($tools as $tool) {
            $this->register($tool);
        }

        return $this;
    }

    /**
     * Check if a tool is registered.
     */
    public function has(string $name): bool
    {
        return isset($this->tools[$name]);
    }

    /**
     * Get a tool by name.
     */
    public function get(string $name): ?AgentToolInterface
    {
        return $this->tools[$name] ?? null;
    }

    /**
     * Get all registered tools.
     *
     * @return Collection<string, AgentToolInterface>
     */
    public function all(): Collection
    {
        return collect($this->tools);
    }

    /**
     * Get tools filtered by category.
     *
     * @return Collection<string, AgentToolInterface>
     */
    public function byCategory(string $category): Collection
    {
        return $this->all()->filter(
            fn (AgentToolInterface $tool) => $tool->category() === $category
        );
    }

    /**
     * Get tools accessible by an API key.
     *
     * @return Collection<string, AgentToolInterface>
     */
    public function forApiKey(ApiKey $apiKey): Collection
    {
        return $this->all()->filter(function (AgentToolInterface $tool) use ($apiKey) {
            // Check if API key has required scopes
            foreach ($tool->requiredScopes() as $scope) {
                if (! $apiKey->hasScope($scope)) {
                    return false;
                }
            }

            // Check if API key has tool-level permission
            return $this->apiKeyCanAccessTool($apiKey, $tool->name());
        });
    }

    /**
     * Check if an API key can access a specific tool.
     */
    public function apiKeyCanAccessTool(ApiKey $apiKey, string $toolName): bool
    {
        $allowedTools = $apiKey->tool_scopes ?? null;

        // Null means all tools allowed
        if ($allowedTools === null) {
            return true;
        }

        return in_array($toolName, $allowedTools, true);
    }

    /**
     * Execute a tool with permission and dependency checking.
     *
     * @param  string  $name  Tool name
     * @param  array  $args  Tool arguments
     * @param  array  $context  Execution context
     * @param  ApiKey|null  $apiKey  Optional API key for permission checking
     * @param  bool  $validateDependencies  Whether to validate dependencies
     * @return array Tool result
     *
     * @throws \InvalidArgumentException If tool not found
     * @throws \RuntimeException If permission denied
     * @throws \Core\Mcp\Exceptions\MissingDependencyException If dependencies not met
     */
    public function execute(
        string $name,
        array $args,
        array $context = [],
        ?ApiKey $apiKey = null,
        bool $validateDependencies = true
    ): array {
        $tool = $this->get($name);

        if (! $tool) {
            throw new \InvalidArgumentException("Unknown tool: {$name}");
        }

        // Permission check if API key provided
        if ($apiKey !== null) {
            // Check scopes
            foreach ($tool->requiredScopes() as $scope) {
                if (! $apiKey->hasScope($scope)) {
                    throw new \RuntimeException(
                        "Permission denied: API key missing scope '{$scope}' for tool '{$name}'"
                    );
                }
            }

            // Check tool-level permission
            if (! $this->apiKeyCanAccessTool($apiKey, $name)) {
                throw new \RuntimeException(
                    "Permission denied: API key does not have access to tool '{$name}'"
                );
            }
        }

        // Dependency check
        if ($validateDependencies) {
            $sessionId = $context['session_id'] ?? 'anonymous';
            $dependencyService = app(ToolDependencyService::class);

            $dependencyService->validateDependencies($sessionId, $name, $context, $args);
        }

        $result = $tool->handle($args, $context);

        // Record successful tool call for dependency tracking
        if ($validateDependencies && ($result['success'] ?? true) !== false) {
            $sessionId = $context['session_id'] ?? 'anonymous';
            app(ToolDependencyService::class)->recordToolCall($sessionId, $name, $args);
        }

        return $result;
    }

    /**
     * Get all tools as MCP tool definitions.
     *
     * @param  ApiKey|null  $apiKey  Filter by API key permissions
     */
    public function toMcpDefinitions(?ApiKey $apiKey = null): array
    {
        $tools = $apiKey !== null
            ? $this->forApiKey($apiKey)
            : $this->all();

        return $tools->map(fn (AgentToolInterface $tool) => $tool->toMcpDefinition())
            ->values()
            ->all();
    }

    /**
     * Get tool categories with counts.
     */
    public function categories(): Collection
    {
        return $this->all()
            ->groupBy(fn (AgentToolInterface $tool) => $tool->category())
            ->map(fn ($tools) => $tools->count());
    }

    /**
     * Get all tool names.
     *
     * @return array<string>
     */
    public function names(): array
    {
        return array_keys($this->tools);
    }

    /**
     * Get tool count.
     */
    public function count(): int
    {
        return count($this->tools);
    }
}
