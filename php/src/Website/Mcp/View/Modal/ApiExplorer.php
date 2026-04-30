<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Livewire\Component;
use Core\Mod\Api\Services\ApiSnippetService;

/**
 * Interactive API Explorer
 *
 * Provides a browser-based interface for exploring API endpoints
 * with live code snippets in multiple languages and real-time testing.
 */
class ApiExplorer extends Component
{
    public string $apiKey = '';

    public string $selectedEndpoint = '';

    public string $selectedLanguage = 'curl';

    public string $baseUrl = '';

    // Request configuration
    public string $method = 'GET';

    public string $path = '/api/v1/workspaces';

    public string $bodyJson = '{}';

    // Response state
    public ?array $response = null;

    public ?string $error = null;

    public bool $isLoading = false;

    public int $responseTime = 0;

    // Available endpoints for quick selection
    public array $endpoints = [
        [
            'name' => 'List Workspaces',
            'method' => 'GET',
            'path' => '/api/v1/workspaces',
            'description' => 'Get all workspaces for the authenticated user',
            'body' => null,
        ],
        [
            'name' => 'Create Workspace',
            'method' => 'POST',
            'path' => '/api/v1/workspaces',
            'description' => 'Create a new workspace',
            'body' => ['name' => 'My Workspace', 'description' => 'A new workspace'],
        ],
        [
            'name' => 'Get Workspace',
            'method' => 'GET',
            'path' => '/api/v1/workspaces/{id}',
            'description' => 'Get a specific workspace by ID',
            'body' => null,
        ],
        [
            'name' => 'Update Workspace',
            'method' => 'PATCH',
            'path' => '/api/v1/workspaces/{id}',
            'description' => 'Update workspace details',
            'body' => ['name' => 'Updated Workspace', 'settings' => ['timezone' => 'UTC']],
        ],
        [
            'name' => 'List Namespaces',
            'method' => 'GET',
            'path' => '/api/v1/namespaces',
            'description' => 'Get all namespaces accessible to the user',
            'body' => null,
        ],
        [
            'name' => 'Check Entitlement',
            'method' => 'POST',
            'path' => '/api/v1/namespaces/{id}/entitlements/check',
            'description' => 'Check if a namespace has access to a feature',
            'body' => ['feature' => 'storage', 'quantity' => 1073741824],
        ],
        [
            'name' => 'List API Keys',
            'method' => 'GET',
            'path' => '/api/v1/api-keys',
            'description' => 'Get all API keys for the workspace',
            'body' => null,
        ],
        [
            'name' => 'Create API Key',
            'method' => 'POST',
            'path' => '/api/v1/api-keys',
            'description' => 'Create a new API key',
            'body' => ['name' => 'Production Key', 'scopes' => ['read:all'], 'rate_limit_tier' => 'pro'],
        ],
    ];

    protected ApiSnippetService $snippetService;

    public function boot(ApiSnippetService $snippetService): void
    {
        $this->snippetService = $snippetService;
    }

    public function mount(): void
    {
        // Set base URL from current request (mcp domain)
        $this->baseUrl = request()->getSchemeAndHttpHost();

        // Pre-select first endpoint
        if (! empty($this->endpoints)) {
            $this->selectEndpoint(0);
        }
    }

    public function selectEndpoint(int $index): void
    {
        if (! isset($this->endpoints[$index])) {
            return;
        }

        $endpoint = $this->endpoints[$index];
        $this->selectedEndpoint = (string) $index;
        $this->method = $endpoint['method'];
        $this->path = $endpoint['path'];
        $this->bodyJson = $endpoint['body']
            ? json_encode($endpoint['body'], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES)
            : '{}';
        $this->response = null;
        $this->error = null;
    }

    public function getCodeSnippet(): string
    {
        $headers = [
            'Authorization' => 'Bearer '.($this->apiKey ?: 'YOUR_API_KEY'),
            'Content-Type' => 'application/json',
            'Accept' => 'application/json',
        ];

        $body = null;
        if (in_array($this->method, ['POST', 'PUT', 'PATCH']) && $this->bodyJson !== '{}') {
            $body = json_decode($this->bodyJson, true);
        }

        return $this->snippetService->generate(
            $this->selectedLanguage,
            $this->method,
            $this->path,
            $headers,
            $body,
            $this->baseUrl
        );
    }

    public function getAllSnippets(): array
    {
        $headers = [
            'Authorization' => 'Bearer '.($this->apiKey ?: 'YOUR_API_KEY'),
            'Content-Type' => 'application/json',
            'Accept' => 'application/json',
        ];

        $body = null;
        if (in_array($this->method, ['POST', 'PUT', 'PATCH']) && $this->bodyJson !== '{}') {
            $body = json_decode($this->bodyJson, true);
        }

        return $this->snippetService->generateAll(
            $this->method,
            $this->path,
            $headers,
            $body,
            $this->baseUrl
        );
    }

    public function copyToClipboard(): void
    {
        $this->dispatch('copy-to-clipboard', code: $this->getCodeSnippet());
    }

    public function sendRequest(): void
    {
        if (empty($this->apiKey)) {
            $this->error = 'Please enter your API key to send requests';

            return;
        }

        $this->isLoading = true;
        $this->response = null;
        $this->error = null;

        try {
            $startTime = microtime(true);

            $url = rtrim($this->baseUrl, '/').'/'.ltrim($this->path, '/');

            $options = [
                'http' => [
                    'method' => $this->method,
                    'header' => [
                        "Authorization: Bearer {$this->apiKey}",
                        'Content-Type: application/json',
                        'Accept: application/json',
                    ],
                    'timeout' => 30,
                    'ignore_errors' => true,
                ],
            ];

            if (in_array($this->method, ['POST', 'PUT', 'PATCH']) && $this->bodyJson !== '{}') {
                $options['http']['content'] = $this->bodyJson;
            }

            $context = stream_context_create($options);
            $result = @file_get_contents($url, false, $context);

            $this->responseTime = (int) round((microtime(true) - $startTime) * 1000);

            if ($result === false) {
                $this->error = 'Request failed - check your API key and endpoint';

                return;
            }

            // Parse response headers
            $statusCode = 200;
            if (isset($http_response_header[0])) {
                preg_match('/HTTP\/\d+\.?\d* (\d+)/', $http_response_header[0], $matches);
                $statusCode = (int) ($matches[1] ?? 200);
            }

            $this->response = [
                'status' => $statusCode,
                'body' => json_decode($result, true) ?? $result,
                'headers' => $http_response_header ?? [],
            ];

        } catch (\Exception $e) {
            $this->error = $e->getMessage();
        } finally {
            $this->isLoading = false;
        }
    }

    public function formatBody(): void
    {
        try {
            $decoded = json_decode($this->bodyJson, true);
            if (json_last_error() === JSON_ERROR_NONE) {
                $this->bodyJson = json_encode($decoded, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
            }
        } catch (\Exception $e) {
            // Ignore
        }
    }

    public function render()
    {
        return view('mcp::web.api-explorer', [
            'languages' => ApiSnippetService::getLanguages(),
            'snippet' => $this->getCodeSnippet(),
        ]);
    }
}
