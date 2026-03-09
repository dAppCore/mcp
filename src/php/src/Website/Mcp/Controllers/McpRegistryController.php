<?php

declare(strict_types=1);

namespace Core\Website\Mcp\Controllers;

use Core\Front\Controller;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Core\Mcp\Models\McpToolCall;
use Core\Mcp\Services\OpenApiGenerator;
use Symfony\Component\Yaml\Yaml;

/**
 * MCP Registry Controller.
 *
 * Serves the MCP server registry with content negotiation:
 * - JSON for agents (Accept: application/json or .json extension)
 * - HTML for humans (browser requests)
 */
class McpRegistryController extends Controller
{
    /**
     * Cache duration for YAML parsing (10 minutes in production, 0 in dev).
     */
    protected function getCacheTtl(): int
    {
        return app()->environment('production') ? 600 : 0;
    }

    /**
     * Discovery endpoint: /.well-known/mcp-servers.json
     *
     * Returns the registry of all available MCP servers.
     * This is the entry point for agent discovery.
     */
    public function registry(Request $request)
    {
        $registry = $this->loadRegistry();

        // Build server summaries for discovery
        $servers = collect($registry['servers'] ?? [])
            ->map(fn ($ref) => $this->loadServerSummary($ref['id']))
            ->filter()
            ->values()
            ->all();

        $data = [
            'servers' => $servers,
            'registry_version' => $registry['registry_version'] ?? '1.0',
            'organization' => $registry['organization'] ?? 'Host UK',
        ];

        // Always return JSON for .well-known
        return response()->json($data);
    }

    /**
     * Server list page: /servers
     *
     * Shows all available servers (HTML) or returns JSON array.
     */
    public function index(Request $request)
    {
        $registry = $this->loadRegistry();

        $servers = collect($registry['servers'] ?? [])
            ->map(fn ($ref) => $this->loadServerFull($ref['id']))
            ->filter()
            ->values();

        // Include planned servers for display
        $plannedServers = collect($registry['planned_servers'] ?? []);

        if ($this->wantsJson($request)) {
            return response()->json([
                'servers' => $servers,
                'planned' => $plannedServers,
            ]);
        }

        return view('mcp::web.index', [
            'servers' => $servers,
            'plannedServers' => $plannedServers,
        ]);
    }

    /**
     * Server detail: /servers/{id} or /servers/{id}.json
     *
     * Returns full server definition with all tools, resources, workflows.
     */
    public function show(Request $request, string $id)
    {
        // Remove .json extension if present
        $id = preg_replace('/\.json$/', '', $id);

        $server = $this->loadServerFull($id);

        if (! $server) {
            if ($this->wantsJson($request)) {
                return response()->json(['error' => 'Server not found'], 404);
            }
            abort(404, 'Server not found');
        }

        if ($this->wantsJson($request)) {
            return response()->json($server);
        }

        return view('mcp::web.show', ['server' => $server]);
    }

    /**
     * Landing page: /
     *
     * MCP portal landing page for humans.
     */
    public function landing(Request $request)
    {
        $registry = $this->loadRegistry();

        $servers = collect($registry['servers'] ?? [])
            ->map(fn ($ref) => $this->loadServerSummary($ref['id']))
            ->filter()
            ->values();

        $plannedServers = collect($registry['planned_servers'] ?? []);

        return view('mcp::web.landing', [
            'servers' => $servers,
            'plannedServers' => $plannedServers,
            'organization' => $registry['organization'] ?? 'Host UK',
        ]);
    }

    /**
     * Connection config generator: /connect
     *
     * Shows how to add MCP servers to Claude Code etc.
     */
    public function connect(Request $request)
    {
        $registry = $this->loadRegistry();

        $servers = collect($registry['servers'] ?? [])
            ->map(fn ($ref) => $this->loadServerFull($ref['id']))
            ->filter()
            ->values();

        return view('mcp::web.connect', [
            'servers' => $servers,
            'templates' => $registry['connection_templates'] ?? [],
            'workspace' => $request->attributes->get('mcp_workspace'),
        ]);
    }

    /**
     * Dashboard: /dashboard
     *
     * Shows MCP usage for the authenticated workspace.
     */
    public function dashboard(Request $request)
    {
        $workspace = $request->attributes->get('mcp_workspace');
        $entitlement = $request->attributes->get('mcp_entitlement');

        // Get tool call stats for this workspace
        $stats = $this->getWorkspaceStats($workspace);

        return view('mcp::web.dashboard', [
            'workspace' => $workspace,
            'entitlement' => $entitlement,
            'stats' => $stats,
        ]);
    }

    /**
     * API Keys management: /keys
     *
     * Manage API keys for MCP access.
     */
    public function keys(Request $request)
    {
        $workspace = $request->attributes->get('mcp_workspace');

        return view('mcp::web.keys', [
            'workspace' => $workspace,
            'keys' => $workspace->apiKeys ?? collect(),
        ]);
    }

    /**
     * Get MCP usage stats for a workspace.
     */
    protected function getWorkspaceStats($workspace): array
    {
        $since = now()->subDays(30);

        // Use aggregate queries instead of loading all records into memory
        $baseQuery = McpToolCall::where('created_at', '>=', $since);

        if ($workspace) {
            $baseQuery->where('workspace_id', $workspace->id);
        }

        $totalCalls = (clone $baseQuery)->count();
        $successfulCalls = (clone $baseQuery)->where('success', true)->count();

        $byServer = (clone $baseQuery)
            ->selectRaw('server_id, COUNT(*) as count')
            ->groupBy('server_id')
            ->orderByDesc('count')
            ->limit(5)
            ->pluck('count', 'server_id')
            ->all();

        $byDay = (clone $baseQuery)
            ->selectRaw('DATE(created_at) as date, COUNT(*) as count')
            ->groupBy('date')
            ->orderBy('date')
            ->pluck('count', 'date')
            ->all();

        return [
            'total_calls' => $totalCalls,
            'successful_calls' => $successfulCalls,
            'by_server' => $byServer,
            'by_day' => $byDay,
        ];
    }

    /**
     * Usage analytics endpoint: /servers/{id}/analytics
     *
     * Shows tool usage stats for a specific server.
     */
    public function analytics(Request $request, string $id)
    {
        $server = $this->loadServerFull($id);

        if (! $server) {
            if ($this->wantsJson($request)) {
                return response()->json(['error' => 'Server not found'], 404);
            }
            abort(404, 'Server not found');
        }

        // Validate days parameter - bound to reasonable range
        $days = min(max($request->integer('days', 7), 1), 90);

        // Get tool call stats for this server
        $stats = $this->getServerAnalytics($id, $days);

        if ($this->wantsJson($request)) {
            return response()->json([
                'server_id' => $id,
                'period_days' => $days,
                'stats' => $stats,
            ]);
        }

        return view('mcp::web.analytics', [
            'server' => $server,
            'stats' => $stats,
            'days' => $days,
        ]);
    }

    /**
     * OpenAPI specification.
     *
     * GET /openapi.json or /openapi.yaml
     */
    public function openapi(Request $request)
    {
        $generator = new OpenApiGenerator;
        $format = $request->query('format', 'json');

        if ($format === 'yaml' || str_ends_with($request->path(), '.yaml')) {
            return response($generator->toYaml())
                ->header('Content-Type', 'application/x-yaml');
        }

        return response()->json($generator->generate());
    }

    /**
     * Get analytics for a specific server.
     */
    protected function getServerAnalytics(string $serverId, int $days = 7): array
    {
        $since = now()->subDays($days);

        $baseQuery = McpToolCall::forServer($serverId)
            ->where('created_at', '>=', $since);

        // Get aggregate stats without loading all records into memory
        $totalCalls = (clone $baseQuery)->count();
        $successfulCalls = (clone $baseQuery)->where('success', true)->count();
        $failedCalls = $totalCalls - $successfulCalls;
        $avgDuration = (clone $baseQuery)->avg('duration_ms') ?? 0;

        // Tool breakdown with aggregates
        $byTool = (clone $baseQuery)
            ->selectRaw('tool_name, COUNT(*) as calls, SUM(CASE WHEN success = 1 THEN 1 ELSE 0 END) as success_count, AVG(duration_ms) as avg_duration')
            ->groupBy('tool_name')
            ->orderByDesc('calls')
            ->limit(10)
            ->get()
            ->mapWithKeys(fn ($row) => [
                $row->tool_name => [
                    'calls' => (int) $row->calls,
                    'success_rate' => $row->calls > 0
                        ? round($row->success_count / $row->calls * 100, 1)
                        : 0,
                    'avg_duration_ms' => round($row->avg_duration ?? 0),
                ],
            ])
            ->all();

        // Daily breakdown
        $byDay = (clone $baseQuery)
            ->selectRaw('DATE(created_at) as date, COUNT(*) as count')
            ->groupBy('date')
            ->orderBy('date')
            ->pluck('count', 'date')
            ->all();

        // Error breakdown
        $errors = (clone $baseQuery)
            ->where('success', false)
            ->whereNotNull('error_code')
            ->selectRaw('error_code, COUNT(*) as count')
            ->groupBy('error_code')
            ->orderByDesc('count')
            ->limit(5)
            ->pluck('count', 'error_code')
            ->all();

        return [
            'total_calls' => $totalCalls,
            'successful_calls' => $successfulCalls,
            'failed_calls' => $failedCalls,
            'success_rate' => $totalCalls > 0 ? round($successfulCalls / $totalCalls * 100, 1) : 0,
            'avg_duration_ms' => round($avgDuration),
            'by_tool' => $byTool,
            'by_day' => $byDay,
            'errors' => $errors,
        ];
    }

    /**
     * Load the main registry file.
     */
    protected function loadRegistry(): array
    {
        return Cache::remember('mcp:registry', $this->getCacheTtl(), function () {
            $path = resource_path('mcp/registry.yaml');

            if (! file_exists($path)) {
                return ['servers' => [], 'planned_servers' => []];
            }

            return Yaml::parseFile($path);
        });
    }

    /**
     * Load a server's YAML file.
     */
    protected function loadServerYaml(string $id): ?array
    {
        // Sanitise server ID to prevent path traversal attacks
        $id = basename($id, '.yaml');

        // Validate ID format (alphanumeric with hyphens only)
        if (! preg_match('/^[a-z0-9-]+$/', $id)) {
            return null;
        }

        return Cache::remember("mcp:server:{$id}", $this->getCacheTtl(), function () use ($id) {
            $path = resource_path("mcp/servers/{$id}.yaml");

            if (! file_exists($path)) {
                return null;
            }

            return Yaml::parseFile($path);
        });
    }

    /**
     * Load server summary for registry discovery.
     *
     * Returns minimal info: id, name, description, use_when, connection type.
     */
    protected function loadServerSummary(string $id): ?array
    {
        $server = $this->loadServerYaml($id);

        if (! $server) {
            return null;
        }

        return [
            'id' => $server['id'],
            'name' => $server['name'],
            'description' => $server['description'] ?? $server['tagline'] ?? '',
            'tagline' => $server['tagline'] ?? '',
            'icon' => $server['icon'] ?? 'server',
            'status' => $server['status'] ?? 'available',
            'use_when' => $server['use_when'] ?? [],
            'connection' => [
                'type' => $server['connection']['type'] ?? 'stdio',
            ],
            'capabilities' => $this->extractCapabilities($server),
            'related_servers' => $server['related_servers'] ?? [],
        ];
    }

    /**
     * Load full server definition for detail view.
     */
    protected function loadServerFull(string $id): ?array
    {
        $server = $this->loadServerYaml($id);

        if (! $server) {
            return null;
        }

        // Add computed fields
        $server['tool_count'] = count($server['tools'] ?? []);
        $server['resource_count'] = count($server['resources'] ?? []);
        $server['workflow_count'] = count($server['workflows'] ?? []);
        $server['capabilities'] = $this->extractCapabilities($server);

        return $server;
    }

    /**
     * Extract capability summary from server definition.
     */
    protected function extractCapabilities(array $server): array
    {
        $caps = [];

        if (! empty($server['tools'])) {
            $caps[] = 'tools';
        }

        if (! empty($server['resources'])) {
            $caps[] = 'resources';
        }

        return $caps;
    }

    /**
     * Check if request wants JSON response.
     */
    protected function wantsJson(Request $request): bool
    {
        // Explicit .json extension
        if (str_ends_with($request->path(), '.json')) {
            return true;
        }

        // Accept header
        if ($request->wantsJson()) {
            return true;
        }

        // Query param override
        if ($request->query('format') === 'json') {
            return true;
        }

        return false;
    }
}
