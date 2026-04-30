<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mod\Api\Models\ApiKey;
use Core\Mcp\Services\ToolRegistry;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\RateLimiter;
use Illuminate\Support\Facades\Session;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Component;

/**
 * MCP Playground - Interactive tool testing interface.
 *
 * Provides a comprehensive UI for testing MCP tools with:
 * - Tool browser with search and category filtering
 * - Dynamic input form generation from JSON schemas
 * - Response viewer with syntax highlighting
 * - Session-based conversation history (last 50 executions)
 * - Example inputs per tool
 */
#[Layout('hub::admin.layouts.app')]
class McpPlayground extends Component
{
    /**
     * Currently selected MCP server ID.
     */
    public string $selectedServer = '';

    /**
     * Currently selected tool name.
     */
    public ?string $selectedTool = null;

    /**
     * Tool input parameters (key-value pairs).
     */
    public array $toolInput = [];

    /**
     * Last response from tool execution.
     */
    public ?array $lastResponse = null;

    /**
     * Conversation/execution history from session.
     */
    public array $conversationHistory = [];

    /**
     * Search query for filtering tools.
     */
    public string $searchQuery = '';

    /**
     * Selected category for filtering tools.
     */
    public string $selectedCategory = '';

    /**
     * API key for authentication.
     */
    public string $apiKey = '';

    /**
     * API key validation status.
     */
    public ?string $keyStatus = null;

    /**
     * Validated API key info.
     */
    public ?array $keyInfo = null;

    /**
     * Error message for display.
     */
    public ?string $error = null;

    /**
     * Whether a request is currently executing.
     */
    public bool $isExecuting = false;

    /**
     * Last execution duration in milliseconds.
     */
    public int $executionTime = 0;

    /**
     * Session key for conversation history.
     */
    protected const HISTORY_SESSION_KEY = 'mcp_playground_history';

    /**
     * Maximum history entries to keep.
     */
    protected const MAX_HISTORY_ENTRIES = 50;

    public function mount(): void
    {
        $this->loadConversationHistory();

        // Auto-select first server if available
        $servers = $this->getServers();
        if ($servers->isNotEmpty()) {
            $this->selectedServer = $servers->first()['id'];
        }
    }

    /**
     * Handle server selection change.
     */
    public function updatedSelectedServer(): void
    {
        $this->selectedTool = null;
        $this->toolInput = [];
        $this->lastResponse = null;
        $this->error = null;
        $this->searchQuery = '';
        $this->selectedCategory = '';
    }

    /**
     * Handle tool selection change.
     */
    public function updatedSelectedTool(): void
    {
        $this->toolInput = [];
        $this->lastResponse = null;
        $this->error = null;

        if ($this->selectedTool) {
            $this->loadExampleInputs();
        }
    }

    /**
     * Handle API key change.
     */
    public function updatedApiKey(): void
    {
        $this->keyStatus = null;
        $this->keyInfo = null;
    }

    /**
     * Validate the API key.
     */
    public function validateKey(): void
    {
        $this->keyStatus = null;
        $this->keyInfo = null;

        if (empty($this->apiKey)) {
            $this->keyStatus = 'empty';

            return;
        }

        $key = ApiKey::findByPlainKey($this->apiKey);

        if (! $key) {
            $this->keyStatus = 'invalid';

            return;
        }

        if ($key->isExpired()) {
            $this->keyStatus = 'expired';

            return;
        }

        $this->keyStatus = 'valid';
        $this->keyInfo = [
            'name' => $key->name,
            'scopes' => $key->scopes ?? [],
            'workspace' => $key->workspace?->name ?? 'Unknown',
            'last_used' => $key->last_used_at?->diffForHumans() ?? 'Never',
        ];
    }

    /**
     * Select a tool by name.
     */
    public function selectTool(string $toolName): void
    {
        $this->selectedTool = $toolName;
        $this->updatedSelectedTool();
    }

    /**
     * Load example inputs for the selected tool.
     */
    public function loadExampleInputs(): void
    {
        if (! $this->selectedTool) {
            return;
        }

        $tool = $this->getRegistry()->getTool($this->selectedServer, $this->selectedTool);

        if (! $tool) {
            return;
        }

        // Load example inputs
        $examples = $tool['examples'] ?? [];

        // Also populate from schema defaults if no examples
        if (empty($examples) && isset($tool['inputSchema']['properties'])) {
            foreach ($tool['inputSchema']['properties'] as $name => $schema) {
                if (isset($schema['default'])) {
                    $examples[$name] = $schema['default'];
                }
            }
        }

        $this->toolInput = $examples;
    }

    /**
     * Execute the selected tool.
     */
    public function execute(): void
    {
        if (! $this->selectedServer || ! $this->selectedTool) {
            $this->error = 'Please select a server and tool.';

            return;
        }

        // Rate limiting: 10 executions per minute
        $rateLimitKey = 'mcp-playground:'.$this->getRateLimitKey();
        if (RateLimiter::tooManyAttempts($rateLimitKey, 10)) {
            $this->error = 'Too many requests. Please wait before trying again.';

            return;
        }
        RateLimiter::hit($rateLimitKey, 60);

        $this->isExecuting = true;
        $this->lastResponse = null;
        $this->error = null;

        try {
            $startTime = microtime(true);

            // Filter empty values from input
            $args = array_filter($this->toolInput, fn ($v) => $v !== '' && $v !== null);

            // Type conversion for arguments
            $args = $this->convertArgumentTypes($args);

            // Execute the tool
            if ($this->keyStatus === 'valid') {
                $result = $this->executeViaApi($args);
            } else {
                $result = $this->generateRequestPreview($args);
            }

            $this->executionTime = (int) round((microtime(true) - $startTime) * 1000);
            $this->lastResponse = $result;

            // Add to conversation history
            $this->addToHistory([
                'server' => $this->selectedServer,
                'tool' => $this->selectedTool,
                'input' => $args,
                'output' => $result,
                'success' => ! isset($result['error']),
                'duration_ms' => $this->executionTime,
                'timestamp' => now()->toIso8601String(),
            ]);

        } catch (\Throwable $e) {
            $this->error = $e->getMessage();
            $this->lastResponse = ['error' => $e->getMessage()];
        } finally {
            $this->isExecuting = false;
        }
    }

    /**
     * Re-run a historical execution.
     */
    public function rerunFromHistory(int $index): void
    {
        if (! isset($this->conversationHistory[$index])) {
            return;
        }

        $entry = $this->conversationHistory[$index];

        $this->selectedServer = $entry['server'];
        $this->selectedTool = $entry['tool'];
        $this->toolInput = $entry['input'] ?? [];

        $this->execute();
    }

    /**
     * View a historical execution result.
     */
    public function viewFromHistory(int $index): void
    {
        if (! isset($this->conversationHistory[$index])) {
            return;
        }

        $entry = $this->conversationHistory[$index];

        $this->selectedServer = $entry['server'];
        $this->selectedTool = $entry['tool'];
        $this->toolInput = $entry['input'] ?? [];
        $this->lastResponse = $entry['output'] ?? null;
        $this->executionTime = $entry['duration_ms'] ?? 0;
    }

    /**
     * Clear conversation history.
     */
    public function clearHistory(): void
    {
        $this->conversationHistory = [];
        Session::forget(self::HISTORY_SESSION_KEY);
    }

    /**
     * Get available servers.
     */
    #[Computed]
    public function getServers(): \Illuminate\Support\Collection
    {
        return $this->getRegistry()->getServers();
    }

    /**
     * Get tools for the selected server.
     */
    #[Computed]
    public function getTools(): \Illuminate\Support\Collection
    {
        if (empty($this->selectedServer)) {
            return collect();
        }

        $tools = $this->getRegistry()->getToolsForServer($this->selectedServer);

        // Apply search filter
        if (! empty($this->searchQuery)) {
            $query = strtolower($this->searchQuery);
            $tools = $tools->filter(function ($tool) use ($query) {
                return str_contains(strtolower($tool['name']), $query)
                    || str_contains(strtolower($tool['description']), $query);
            });
        }

        // Apply category filter
        if (! empty($this->selectedCategory)) {
            $tools = $tools->filter(fn ($tool) => $tool['category'] === $this->selectedCategory);
        }

        return $tools->values();
    }

    /**
     * Get tools grouped by category.
     */
    #[Computed]
    public function getToolsByCategory(): \Illuminate\Support\Collection
    {
        return $this->getTools()->groupBy('category')->sortKeys();
    }

    /**
     * Get available categories.
     */
    #[Computed]
    public function getCategories(): \Illuminate\Support\Collection
    {
        if (empty($this->selectedServer)) {
            return collect();
        }

        return $this->getRegistry()
            ->getToolsForServer($this->selectedServer)
            ->pluck('category')
            ->unique()
            ->sort()
            ->values();
    }

    /**
     * Get the current tool schema.
     */
    #[Computed]
    public function getCurrentTool(): ?array
    {
        if (! $this->selectedTool) {
            return null;
        }

        return $this->getRegistry()->getTool($this->selectedServer, $this->selectedTool);
    }

    /**
     * Check if user is authenticated.
     */
    public function isAuthenticated(): bool
    {
        return auth()->check();
    }

    public function render()
    {
        return view('mcp::admin.mcp-playground', [
            'servers' => $this->getServers(),
            'tools' => $this->getTools(),
            'toolsByCategory' => $this->getToolsByCategory(),
            'categories' => $this->getCategories(),
            'currentTool' => $this->getCurrentTool(),
            'isAuthenticated' => $this->isAuthenticated(),
        ]);
    }

    /**
     * Get the tool registry service.
     */
    protected function getRegistry(): ToolRegistry
    {
        return app(ToolRegistry::class);
    }

    /**
     * Get rate limit key based on user or IP.
     */
    protected function getRateLimitKey(): string
    {
        if (auth()->check()) {
            return 'user:'.auth()->id();
        }

        return 'ip:'.request()->ip();
    }

    /**
     * Convert argument types based on their values.
     */
    protected function convertArgumentTypes(array $args): array
    {
        foreach ($args as $key => $value) {
            if (is_numeric($value)) {
                $args[$key] = str_contains((string) $value, '.') ? (float) $value : (int) $value;
            }
            if ($value === 'true') {
                $args[$key] = true;
            }
            if ($value === 'false') {
                $args[$key] = false;
            }
        }

        return $args;
    }

    /**
     * Execute tool via HTTP API.
     */
    protected function executeViaApi(array $args): array
    {
        $payload = [
            'server' => $this->selectedServer,
            'tool' => $this->selectedTool,
            'arguments' => $args,
        ];

        $response = Http::withToken($this->apiKey)
            ->timeout(30)
            ->post(config('app.url').'/api/v1/mcp/tools/call', $payload);

        return [
            'status' => $response->status(),
            'response' => $response->json(),
            'executed' => true,
        ];
    }

    /**
     * Generate a request preview without executing.
     */
    protected function generateRequestPreview(array $args): array
    {
        $payload = [
            'server' => $this->selectedServer,
            'tool' => $this->selectedTool,
            'arguments' => $args,
        ];

        return [
            'request' => $payload,
            'note' => 'Add a valid API key to execute this request live.',
            'curl' => sprintf(
                "curl -X POST %s/api/v1/mcp/tools/call \\\n  -H \"Authorization: Bearer YOUR_API_KEY\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '%s'",
                config('app.url'),
                json_encode($payload, JSON_UNESCAPED_SLASHES)
            ),
            'executed' => false,
        ];
    }

    /**
     * Load conversation history from session.
     */
    protected function loadConversationHistory(): void
    {
        $this->conversationHistory = Session::get(self::HISTORY_SESSION_KEY, []);
    }

    /**
     * Add an entry to conversation history.
     */
    protected function addToHistory(array $entry): void
    {
        // Prepend new entry
        array_unshift($this->conversationHistory, $entry);

        // Keep only last N entries
        $this->conversationHistory = array_slice($this->conversationHistory, 0, self::MAX_HISTORY_ENTRIES);

        // Save to session
        Session::put(self::HISTORY_SESSION_KEY, $this->conversationHistory);
    }
}
