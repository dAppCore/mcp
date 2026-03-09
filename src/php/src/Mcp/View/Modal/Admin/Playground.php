<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mod\Api\Models\ApiKey;
use Illuminate\Support\Facades\Http;
use Livewire\Attributes\Layout;
use Livewire\Component;
use Symfony\Component\Yaml\Yaml;

/**
 * MCP Playground - interactive tool testing in the browser.
 */
#[Layout('hub::admin.layouts.app')]
class Playground extends Component
{
    public string $selectedServer = '';

    public string $selectedTool = '';

    public array $arguments = [];

    public string $response = '';

    public bool $loading = false;

    public string $apiKey = '';

    public ?string $error = null;

    public ?string $keyStatus = null;

    public ?array $keyInfo = null;

    public array $servers = [];

    public array $tools = [];

    public ?array $toolSchema = null;

    public function mount(): void
    {
        $this->loadServers();
    }

    public function loadServers(): void
    {
        try {
            $registry = $this->loadRegistry();
            $this->servers = collect($registry['servers'] ?? [])
                ->map(fn ($ref) => $this->loadServerSummary($ref['id']))
                ->filter()
                ->values()
                ->toArray();
        } catch (\Throwable $e) {
            $this->error = 'Failed to load servers';
            $this->servers = [];
        }
    }

    public function updatedSelectedServer(): void
    {
        $this->error = null;
        $this->selectedTool = '';
        $this->toolSchema = null;
        $this->arguments = [];
        $this->response = '';

        if (! $this->selectedServer) {
            $this->tools = [];

            return;
        }

        try {
            $server = $this->loadServerFull($this->selectedServer);
            $this->tools = $server['tools'] ?? [];
        } catch (\Throwable $e) {
            $this->error = 'Failed to load server tools';
            $this->tools = [];
        }
    }

    public function updatedSelectedTool(): void
    {
        $this->error = null;
        $this->arguments = [];
        $this->response = '';

        if (! $this->selectedTool) {
            $this->toolSchema = null;

            return;
        }

        try {
            $this->toolSchema = collect($this->tools)->firstWhere('name', $this->selectedTool);

            // Pre-fill arguments with defaults
            $params = $this->toolSchema['inputSchema']['properties'] ?? [];
            foreach ($params as $name => $schema) {
                $this->arguments[$name] = $schema['default'] ?? '';
            }
        } catch (\Throwable $e) {
            $this->error = 'Failed to load tool schema';
            $this->toolSchema = null;
        }
    }

    public function updatedApiKey(): void
    {
        // Clear key status when key changes
        $this->keyStatus = null;
        $this->keyInfo = null;
    }

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
            'scopes' => $key->scopes,
            'server_scopes' => $key->getAllowedServers(),
            'workspace' => $key->workspace?->name ?? 'Unknown',
            'last_used' => $key->last_used_at?->diffForHumans() ?? 'Never',
        ];
    }

    public function isAuthenticated(): bool
    {
        return auth()->check();
    }

    public function execute(): void
    {
        if (! $this->selectedServer || ! $this->selectedTool) {
            return;
        }

        $this->loading = true;
        $this->response = '';
        $this->error = null;

        try {
            // Filter out empty arguments
            $args = array_filter($this->arguments, fn ($v) => $v !== '' && $v !== null);

            // Convert numeric strings to numbers where appropriate
            foreach ($args as $key => $value) {
                if (is_numeric($value)) {
                    $args[$key] = str_contains($value, '.') ? (float) $value : (int) $value;
                }
                if ($value === 'true') {
                    $args[$key] = true;
                }
                if ($value === 'false') {
                    $args[$key] = false;
                }
            }

            $payload = [
                'server' => $this->selectedServer,
                'tool' => $this->selectedTool,
                'arguments' => $args,
            ];

            // If we have an API key, make a real request
            if (! empty($this->apiKey) && $this->keyStatus === 'valid') {
                $response = Http::withToken($this->apiKey)
                    ->timeout(30)
                    ->post(config('app.url').'/api/v1/mcp/tools/call', $payload);

                $this->response = json_encode([
                    'status' => $response->status(),
                    'response' => $response->json(),
                ], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);

                return;
            }

            // Otherwise, just show request format
            $this->response = json_encode([
                'request' => $payload,
                'note' => 'Add an API key above to execute this request live.',
                'curl' => sprintf(
                    "curl -X POST %s/api/v1/mcp/tools/call \\\n  -H \"Authorization: Bearer YOUR_API_KEY\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '%s'",
                    config('app.url'),
                    json_encode($payload, JSON_UNESCAPED_SLASHES)
                ),
            ], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
        } catch (\Throwable $e) {
            $this->response = json_encode([
                'error' => $e->getMessage(),
            ], JSON_PRETTY_PRINT);
        } finally {
            $this->loading = false;
        }
    }

    public function render()
    {
        $isAuthenticated = $this->isAuthenticated();
        $workspace = $isAuthenticated ? auth()->user()?->defaultHostWorkspace() : null;

        return view('mcp::admin.playground', [
            'isAuthenticated' => $isAuthenticated,
            'workspace' => $workspace,
        ]);
    }

    protected function loadRegistry(): array
    {
        $path = resource_path('mcp/registry.yaml');

        return file_exists($path) ? Yaml::parseFile($path) : ['servers' => []];
    }

    protected function loadServerFull(string $id): ?array
    {
        $path = resource_path("mcp/servers/{$id}.yaml");

        return file_exists($path) ? Yaml::parseFile($path) : null;
    }

    protected function loadServerSummary(string $id): ?array
    {
        $server = $this->loadServerFull($id);
        if (! $server) {
            return null;
        }

        return [
            'id' => $server['id'],
            'name' => $server['name'],
            'tagline' => $server['tagline'] ?? '',
        ];
    }
}
