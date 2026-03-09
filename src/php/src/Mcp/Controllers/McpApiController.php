<?php

declare(strict_types=1);

namespace Core\Mcp\Controllers;

use Core\Front\Controller;
use Core\Mcp\Services\McpQuotaService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Core\Api\Models\ApiKey;
use Core\Mcp\Models\McpApiRequest;
use Core\Mcp\Models\McpToolCall;
use Core\Mcp\Services\McpWebhookDispatcher;
use Symfony\Component\Yaml\Yaml;

/**
 * MCP HTTP API Controller.
 *
 * Provides HTTP bridge to MCP servers for external integrations.
 */
class McpApiController extends Controller
{
    /**
     * List all available MCP servers.
     *
     * GET /api/v1/mcp/servers
     */
    public function servers(Request $request): JsonResponse
    {
        $registry = $this->loadRegistry();

        $servers = collect($registry['servers'] ?? [])
            ->map(fn ($ref) => $this->loadServerSummary($ref['id']))
            ->filter()
            ->values();

        return response()->json([
            'servers' => $servers,
            'count' => $servers->count(),
        ]);
    }

    /**
     * Get server details with tools and resources.
     *
     * GET /api/v1/mcp/servers/{id}
     */
    public function server(Request $request, string $id): JsonResponse
    {
        $server = $this->loadServerFull($id);

        if (! $server) {
            return response()->json(['error' => 'Server not found'], 404);
        }

        return response()->json($server);
    }

    /**
     * List tools for a specific server.
     *
     * GET /api/v1/mcp/servers/{id}/tools
     */
    public function tools(Request $request, string $id): JsonResponse
    {
        $server = $this->loadServerFull($id);

        if (! $server) {
            return response()->json(['error' => 'Server not found'], 404);
        }

        return response()->json([
            'server' => $id,
            'tools' => $server['tools'] ?? [],
            'count' => count($server['tools'] ?? []),
        ]);
    }

    /**
     * Execute a tool on an MCP server.
     *
     * POST /api/v1/mcp/tools/call
     */
    public function callTool(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'server' => 'required|string|max:64',
            'tool' => 'required|string|max:128',
            'arguments' => 'nullable|array',
        ]);

        $server = $this->loadServerFull($validated['server']);
        if (! $server) {
            return response()->json(['error' => 'Server not found'], 404);
        }

        // Verify tool exists
        $toolDef = collect($server['tools'] ?? [])->firstWhere('name', $validated['tool']);
        if (! $toolDef) {
            return response()->json(['error' => 'Tool not found'], 404);
        }

        // Validate arguments against tool's input schema
        $validationErrors = $this->validateToolArguments($toolDef, $validated['arguments'] ?? []);
        if (! empty($validationErrors)) {
            return response()->json([
                'error' => 'validation_failed',
                'message' => 'Tool arguments do not match input schema',
                'validation_errors' => $validationErrors,
            ], 422);
        }

        // Get API key for logging
        $apiKey = $request->attributes->get('api_key');
        $workspace = $apiKey?->workspace;

        $startTime = microtime(true);

        try {
            // Execute the tool via in-process registry or artisan fallback
            $result = $this->executeTool(
                $validated['tool'],
                $validated['arguments'] ?? [],
                $apiKey
            );

            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            // Log the call
            $this->logToolCall($apiKey, $validated, $result, $durationMs, true);

            // Record quota usage
            $this->recordQuotaUsage($workspace);

            // Dispatch webhooks
            $this->dispatchWebhook($apiKey, $validated, true, $durationMs);

            $response = [
                'success' => true,
                'server' => $validated['server'],
                'tool' => $validated['tool'],
                'result' => $result,
                'duration_ms' => $durationMs,
            ];

            // Log full request for debugging/replay
            $this->logApiRequest($request, $validated, 200, $response, $durationMs, $apiKey);

            return response()->json($response);
        } catch (\Throwable $e) {
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            $this->logToolCall($apiKey, $validated, null, $durationMs, false, $e->getMessage());

            // Dispatch webhooks (even on failure)
            $this->dispatchWebhook($apiKey, $validated, false, $durationMs, $e->getMessage());

            $response = [
                'success' => false,
                'error' => $e->getMessage(),
                'server' => $validated['server'],
                'tool' => $validated['tool'],
            ];

            // Log full request for debugging/replay
            $this->logApiRequest($request, $validated, 500, $response, $durationMs, $apiKey, $e->getMessage());

            return response()->json($response, 500);
        }
    }

    /**
     * Read a resource from an MCP server.
     *
     * GET /api/v1/mcp/resources/{uri}
     *
     * NOTE: Resource reading is not yet implemented. Returns 501 Not Implemented.
     */
    public function resource(Request $request, string $uri): JsonResponse
    {
        // Parse URI format: server://resource/path
        if (! preg_match('/^([a-z0-9-]+):\/\/(.+)$/', $uri, $matches)) {
            return response()->json(['error' => 'Invalid resource URI format'], 400);
        }

        $serverId = $matches[1];

        $server = $this->loadServerFull($serverId);
        if (! $server) {
            return response()->json(['error' => 'Server not found'], 404);
        }

        // Resource reading not yet implemented
        return response()->json([
            'error' => 'not_implemented',
            'message' => 'MCP resource reading is not yet implemented. Use tool calls instead.',
            'uri' => $uri,
        ], 501);
    }

    /**
     * Execute a tool via the in-process AgentToolRegistry.
     *
     * Tools are registered at boot via the McpToolsRegistering lifecycle event.
     * This avoids the overhead of spawning artisan sub-processes for each call.
     *
     * @throws \RuntimeException If tool not found in registry
     */
    protected function executeTool(string $tool, array $arguments, ?ApiKey $apiKey): mixed
    {
        $registryClass = \Core\Mod\Agentic\Services\AgentToolRegistry::class;

        if (! app()->bound($registryClass)) {
            throw new \RuntimeException("AgentToolRegistry not available — is the agentic module installed?");
        }

        $registry = app($registryClass);

        if (! $registry->has($tool)) {
            throw new \RuntimeException("Tool not found: {$tool}");
        }

        $context = [];

        if ($apiKey?->workspace_id) {
            $context['workspace_id'] = $apiKey->workspace_id;
        }

        return $registry->execute(
            name: $tool,
            args: $arguments,
            context: $context,
            apiKey: $apiKey,
            validateDependencies: false
        );
    }

    /**
     * Log full API request for debugging and replay.
     */
    protected function logApiRequest(
        Request $request,
        array $validated,
        int $status,
        array $response,
        int $durationMs,
        ?ApiKey $apiKey,
        ?string $error = null
    ): void {
        try {
            McpApiRequest::log(
                method: $request->method(),
                path: '/tools/call',
                requestBody: $validated,
                responseStatus: $status,
                responseBody: $response,
                durationMs: $durationMs,
                workspaceId: $apiKey?->workspace_id,
                apiKeyId: $apiKey?->id,
                serverId: $validated['server'],
                toolName: $validated['tool'],
                errorMessage: $error,
                ipAddress: $request->ip(),
                headers: $request->headers->all()
            );
        } catch (\Throwable $e) {
            // Don't let logging failures affect API response
            report($e);
        }
    }

    /**
     * Dispatch webhook for tool execution.
     */
    protected function dispatchWebhook(
        ?ApiKey $apiKey,
        array $request,
        bool $success,
        int $durationMs,
        ?string $error = null
    ): void {
        if (! $apiKey?->workspace_id) {
            return;
        }

        try {
            $dispatcher = new McpWebhookDispatcher;
            $dispatcher->dispatchToolExecuted(
                workspaceId: $apiKey->workspace_id,
                serverId: $request['server'],
                toolName: $request['tool'],
                arguments: $request['arguments'] ?? [],
                success: $success,
                durationMs: $durationMs,
                errorMessage: $error
            );
        } catch (\Throwable $e) {
            // Don't let webhook failures affect API response
            report($e);
        }
    }

    /**
     * Log tool call for analytics.
     */
    protected function logToolCall(
        ?ApiKey $apiKey,
        array $request,
        mixed $result,
        int $durationMs,
        bool $success,
        ?string $error = null
    ): void {
        try {
            McpToolCall::log(
                serverId: $request['server'],
                toolName: $request['tool'],
                params: $request['arguments'] ?? [],
                success: $success,
                durationMs: $durationMs,
                errorMessage: $error,
                workspaceId: $apiKey?->workspace_id
            );
        } catch (\Throwable $e) {
            // Don't let logging failures affect API response
            report($e);
        }
    }

    /**
     * Validate tool arguments against the tool's input schema.
     *
     * @return array<string> Validation errors (empty if valid)
     */
    protected function validateToolArguments(array $toolDef, array $arguments): array
    {
        $inputSchema = $toolDef['inputSchema'] ?? null;

        // No schema = no validation
        if (! $inputSchema || ! is_array($inputSchema)) {
            return [];
        }

        $errors = [];
        $properties = $inputSchema['properties'] ?? [];
        $required = $inputSchema['required'] ?? [];

        // Check required properties
        foreach ($required as $requiredProp) {
            if (! array_key_exists($requiredProp, $arguments)) {
                $errors[] = "Missing required argument: {$requiredProp}";
            }
        }

        // Type validation for provided arguments
        foreach ($arguments as $key => $value) {
            // Check if argument is defined in schema
            if (! isset($properties[$key])) {
                // Allow extra properties unless additionalProperties is false
                if (isset($inputSchema['additionalProperties']) && $inputSchema['additionalProperties'] === false) {
                    $errors[] = "Unknown argument: {$key}";
                }

                continue;
            }

            $propSchema = $properties[$key];
            $expectedType = $propSchema['type'] ?? null;

            if ($expectedType && ! $this->validateType($value, $expectedType)) {
                $errors[] = "Argument '{$key}' must be of type {$expectedType}";
            }

            // Validate enum values
            if (isset($propSchema['enum']) && ! in_array($value, $propSchema['enum'], true)) {
                $allowedValues = implode(', ', $propSchema['enum']);
                $errors[] = "Argument '{$key}' must be one of: {$allowedValues}";
            }

            // Validate string constraints
            if ($expectedType === 'string' && is_string($value)) {
                if (isset($propSchema['minLength']) && strlen($value) < $propSchema['minLength']) {
                    $errors[] = "Argument '{$key}' must be at least {$propSchema['minLength']} characters";
                }
                if (isset($propSchema['maxLength']) && strlen($value) > $propSchema['maxLength']) {
                    $errors[] = "Argument '{$key}' must be at most {$propSchema['maxLength']} characters";
                }
            }

            // Validate numeric constraints
            if (in_array($expectedType, ['integer', 'number']) && is_numeric($value)) {
                if (isset($propSchema['minimum']) && $value < $propSchema['minimum']) {
                    $errors[] = "Argument '{$key}' must be at least {$propSchema['minimum']}";
                }
                if (isset($propSchema['maximum']) && $value > $propSchema['maximum']) {
                    $errors[] = "Argument '{$key}' must be at most {$propSchema['maximum']}";
                }
            }
        }

        return $errors;
    }

    /**
     * Validate a value against a JSON Schema type.
     */
    protected function validateType(mixed $value, string $type): bool
    {
        return match ($type) {
            'string' => is_string($value),
            'integer' => is_int($value) || (is_numeric($value) && floor((float) $value) == $value),
            'number' => is_numeric($value),
            'boolean' => is_bool($value),
            'array' => is_array($value) && array_is_list($value),
            'object' => is_array($value) && ! array_is_list($value),
            'null' => is_null($value),
            default => true, // Unknown types pass validation
        };
    }

    // Registry loading methods (shared with McpRegistryController)

    protected function loadRegistry(): array
    {
        return Cache::remember('mcp:registry', 600, function () {
            $path = resource_path('mcp/registry.yaml');

            return file_exists($path) ? Yaml::parseFile($path) : ['servers' => []];
        });
    }

    protected function loadServerFull(string $id): ?array
    {
        return Cache::remember("mcp:server:{$id}", 600, function () use ($id) {
            $path = resource_path("mcp/servers/{$id}.yaml");

            return file_exists($path) ? Yaml::parseFile($path) : null;
        });
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
            'status' => $server['status'] ?? 'available',
            'tool_count' => count($server['tools'] ?? []),
            'resource_count' => count($server['resources'] ?? []),
        ];
    }

    /**
     * Record quota usage for successful tool calls.
     */
    protected function recordQuotaUsage($workspace): void
    {
        if (! $workspace) {
            return;
        }

        try {
            $quotaService = app(McpQuotaService::class);
            $quotaService->recordUsage($workspace, toolCalls: 1);
        } catch (\Throwable $e) {
            // Don't let quota recording failures affect API response
            report($e);
        }
    }
}
