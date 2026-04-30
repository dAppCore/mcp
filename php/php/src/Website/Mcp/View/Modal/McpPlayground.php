<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\RateLimiter;
use Livewire\Attributes\Layout;
use Livewire\Component;
use Core\Mcp\Models\McpToolCall;
use Symfony\Component\Process\Process;
use Symfony\Component\Yaml\Yaml;

/**
 * MCP Tool Testing Playground
 *
 * A browser-based UI for testing MCP tool calls.
 * Allows users to select a server, pick a tool, and execute it with custom parameters.
 */
#[Layout('layouts::mcp')]
class McpPlayground extends Component
{
    public string $selectedServer = '';

    public string $selectedTool = '';

    public string $inputJson = '{}';

    public ?array $lastResult = null;

    public ?string $lastError = null;

    public bool $isExecuting = false;

    public int $executionTime = 0;

    public array $servers = [];

    public array $tools = [];

    protected $rules = [
        'selectedServer' => 'required|string',
        'selectedTool' => 'required|string',
        'inputJson' => 'required|json',
    ];

    public function mount(): void
    {
        $this->loadServers();

        if (! empty($this->servers)) {
            $this->selectedServer = $this->servers[0]['id'];
            $this->loadTools();
        }
    }

    public function updatedSelectedServer(): void
    {
        $this->loadTools();
        $this->selectedTool = '';
        $this->inputJson = '{}';
        $this->lastResult = null;
        $this->lastError = null;
    }

    public function updatedSelectedTool(): void
    {
        // Pre-fill example parameters based on tool definition
        $this->prefillParameters();
        $this->lastResult = null;
        $this->lastError = null;
    }

    public function execute(): void
    {
        $this->validate();

        // Rate limit: 10 executions per minute per user/IP
        $rateLimitKey = 'mcp-playground:'.$this->getRateLimitKey();
        if (RateLimiter::tooManyAttempts($rateLimitKey, 10)) {
            $this->lastError = 'Too many requests. Please wait before trying again.';

            return;
        }
        RateLimiter::hit($rateLimitKey, 60);

        $this->isExecuting = true;
        $this->lastResult = null;
        $this->lastError = null;

        try {
            $params = json_decode($this->inputJson, true);
            if (json_last_error() !== JSON_ERROR_NONE) {
                $this->lastError = 'Invalid JSON: '.json_last_error_msg();

                return;
            }

            $startTime = microtime(true);
            $result = $this->callTool($this->selectedServer, $this->selectedTool, $params);
            $this->executionTime = (int) round((microtime(true) - $startTime) * 1000);

            if (isset($result['error'])) {
                $this->lastError = $result['error'];
                $this->lastResult = $result;
            } else {
                $this->lastResult = $result;
            }

        } catch (\Exception $e) {
            $this->lastError = $e->getMessage();
        } finally {
            $this->isExecuting = false;
        }
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

    public function formatJson(): void
    {
        try {
            $decoded = json_decode($this->inputJson, true);
            if (json_last_error() === JSON_ERROR_NONE) {
                $this->inputJson = json_encode($decoded, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
            }
        } catch (\Exception $e) {
            // Ignore formatting errors
        }
    }

    protected function loadServers(): void
    {
        $registry = $this->loadRegistry();

        $this->servers = collect($registry['servers'] ?? [])
            ->map(fn ($ref) => $this->loadServerYaml($ref['id']))
            ->filter()
            ->map(fn ($server) => [
                'id' => $server['id'],
                'name' => $server['name'],
                'tagline' => $server['tagline'] ?? '',
                'tool_count' => count($server['tools'] ?? []),
            ])
            ->values()
            ->all();
    }

    protected function loadTools(): void
    {
        if (empty($this->selectedServer)) {
            $this->tools = [];

            return;
        }

        $server = $this->loadServerYaml($this->selectedServer);

        $this->tools = collect($server['tools'] ?? [])
            ->map(fn ($tool) => [
                'name' => $tool['name'],
                'purpose' => $tool['purpose'] ?? '',
                'parameters' => $tool['parameters'] ?? [],
            ])
            ->values()
            ->all();
    }

    protected function prefillParameters(): void
    {
        if (empty($this->selectedTool)) {
            $this->inputJson = '{}';

            return;
        }

        $tool = collect($this->tools)->firstWhere('name', $this->selectedTool);

        if (! $tool || empty($tool['parameters'])) {
            $this->inputJson = '{}';

            return;
        }

        // Build example params from parameter definitions
        $params = [];
        foreach ($tool['parameters'] as $paramName => $paramDef) {
            if (is_array($paramDef)) {
                $type = $paramDef['type'] ?? 'string';
                $default = $paramDef['default'] ?? null;
                $required = $paramDef['required'] ?? false;

                if ($default !== null) {
                    $params[$paramName] = $default;
                } elseif ($required) {
                    // Add placeholder
                    $params[$paramName] = match ($type) {
                        'boolean' => false,
                        'integer', 'number' => 0,
                        'array' => [],
                        default => '',
                    };
                }
            }
        }

        $this->inputJson = json_encode($params, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    }

    protected function callTool(string $serverId, string $toolName, array $params): array
    {
        $server = $this->loadServerYaml($serverId);

        if (! $server) {
            return ['error' => 'Server not found'];
        }

        $connection = $server['connection'] ?? [];
        $type = $connection['type'] ?? 'stdio';

        if ($type !== 'stdio') {
            return ['error' => "Connection type '{$type}' not supported in playground"];
        }

        $command = $connection['command'] ?? null;
        $args = $connection['args'] ?? [];
        $cwd = $this->resolveEnvVars($connection['cwd'] ?? getcwd());

        if (! $command) {
            return ['error' => 'No command configured for this server'];
        }

        // Build MCP tool call request
        $request = json_encode([
            'jsonrpc' => '2.0',
            'method' => 'tools/call',
            'params' => [
                'name' => $toolName,
                'arguments' => $params,
            ],
            'id' => 1,
        ]);

        try {
            $startTime = microtime(true);

            $fullCommand = array_merge([$command], $args);
            $process = new Process($fullCommand, $cwd);
            $process->setInput($request);
            $process->setTimeout(30);

            $process->run();

            $duration = (int) round((microtime(true) - $startTime) * 1000);
            $output = $process->getOutput();

            // Log the tool call
            McpToolCall::log(
                serverId: $serverId,
                toolName: $toolName,
                params: $params,
                success: $process->isSuccessful(),
                durationMs: $duration,
                errorMessage: $process->isSuccessful() ? null : $process->getErrorOutput(),
            );

            if (! $process->isSuccessful()) {
                return [
                    'error' => 'Process failed',
                    'exit_code' => $process->getExitCode(),
                    'stderr' => $process->getErrorOutput(),
                ];
            }

            // Parse JSON-RPC response
            $lines = explode("\n", trim($output));
            foreach ($lines as $line) {
                $response = json_decode($line, true);
                if ($response) {
                    if (isset($response['error'])) {
                        return [
                            'error' => $response['error']['message'] ?? 'Unknown error',
                            'code' => $response['error']['code'] ?? null,
                            'data' => $response['error']['data'] ?? null,
                        ];
                    }
                    if (isset($response['result'])) {
                        return $response['result'];
                    }
                }
            }

            return [
                'error' => 'No valid response received',
                'raw_output' => $output,
            ];

        } catch (\Exception $e) {
            return ['error' => $e->getMessage()];
        }
    }

    protected function loadRegistry(): array
    {
        return Cache::remember('mcp:registry', 0, function () {
            $path = resource_path('mcp/registry.yaml');
            if (! file_exists($path)) {
                return ['servers' => []];
            }

            return Yaml::parseFile($path);
        });
    }

    protected function loadServerYaml(string $id): ?array
    {
        // Sanitise server ID to prevent path traversal attacks
        $id = basename($id, '.yaml');

        // Validate ID format (alphanumeric with hyphens only)
        if (! preg_match('/^[a-z0-9-]+$/', $id)) {
            return null;
        }

        $path = resource_path("mcp/servers/{$id}.yaml");
        if (! file_exists($path)) {
            return null;
        }

        return Yaml::parseFile($path);
    }

    protected function resolveEnvVars(string $value): string
    {
        return preg_replace_callback('/\$\{([^}]+)\}/', function ($matches) {
            $parts = explode(':-', $matches[1], 2);
            $var = $parts[0];
            $default = $parts[1] ?? '';

            return env($var, $default);
        }, $value);
    }

    public function render()
    {
        return view('mcp::web.mcp-playground');
    }
}
