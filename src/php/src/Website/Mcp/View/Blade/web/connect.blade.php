<x-layouts::mcp>
    <x-slot:title>Setup Guide</x-slot:title>

    @php
        $mcpUrl = request()->getSchemeAndHttpHost();
    @endphp

    <div class="max-w-4xl mx-auto">
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">Setup Guide</h1>
            <p class="mt-2 text-lg text-zinc-600 dark:text-zinc-400">
                Connect AI agents to MCP servers via HTTP.
            </p>
        </div>

        <!-- Quick Links -->
        <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mb-8">
            <a href="{{ route('mcp.servers.index') }}"
               class="p-3 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 transition-colors text-center">
                <flux:icon.server-stack class="w-5 h-5 mx-auto mb-1 text-emerald-600 dark:text-emerald-400" />
                <span class="text-sm font-medium text-zinc-900 dark:text-white">Servers</span>
            </a>
            <a href="{{ route('mcp.openapi.json') }}" target="_blank"
               class="p-3 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 transition-colors text-center">
                <flux:icon.code-bracket class="w-5 h-5 mx-auto mb-1 text-cyan-600 dark:text-cyan-400" />
                <span class="text-sm font-medium text-zinc-900 dark:text-white">OpenAPI</span>
            </a>
            <a href="{{ route('mcp.registry') }}" target="_blank"
               class="p-3 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 transition-colors text-center">
                <flux:icon.document-text class="w-5 h-5 mx-auto mb-1 text-violet-600 dark:text-violet-400" />
                <span class="text-sm font-medium text-zinc-900 dark:text-white">Registry JSON</span>
            </a>
        </div>

        <!-- HTTP API -->
        <div class="bg-white dark:bg-zinc-800 rounded-xl border-2 border-cyan-500 p-6 mb-8">
            <div class="flex items-center space-x-3 mb-4">
                <div class="p-2 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg">
                    <flux:icon.globe-alt class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                </div>
                <div>
                    <h2 class="text-xl font-semibold text-zinc-900 dark:text-white">HTTP API</h2>
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-cyan-100 text-cyan-800 dark:bg-cyan-900/30 dark:text-cyan-300">
                        All platforms
                    </span>
                </div>
            </div>

            <p class="text-zinc-600 dark:text-zinc-400 mb-6">
                Call MCP tools from any language, platform, or AI agent using standard HTTP requests.
                Works with Claude Code, Cursor, custom agents, webhooks, and any HTTP client.
            </p>

            <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">1. Get your API key</h3>
            <p class="text-zinc-600 dark:text-zinc-400 mb-4 text-sm">
                Create an API key from your admin dashboard. Keys use the <code class="px-1 py-0.5 bg-zinc-100 dark:bg-zinc-700 rounded text-xs">hk_</code> prefix.
            </p>

            <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">2. Discover available servers</h3>
            <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm mb-4"><code class="text-emerald-400">curl {{ $mcpUrl }}/servers.json \
  -H "Authorization: Bearer YOUR_API_KEY"</code></pre>

            <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">3. Call a tool</h3>
            <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm mb-4"><code class="text-emerald-400">curl -X POST {{ $mcpUrl }}/tools/call \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "server": "openbrain",
    "tool": "brain_recall",
    "arguments": { "query": "authentication decisions" }
  }'</code></pre>

            <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">4. Read a resource</h3>
            <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm"><code class="text-emerald-400">curl {{ $mcpUrl }}/resources/plans://all \
  -H "Authorization: Bearer YOUR_API_KEY"</code></pre>

            <div class="mt-6 p-4 bg-zinc-50 dark:bg-zinc-900/50 rounded-lg">
                <div class="flex items-center justify-between mb-2">
                    <h4 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300">Endpoints</h4>
                    <a href="{{ route('mcp.openapi.json') }}" target="_blank" class="text-xs text-cyan-600 hover:text-cyan-700 dark:text-cyan-400">
                        View OpenAPI Spec →
                    </a>
                </div>
                <div class="space-y-2 text-sm">
                    <div class="flex items-center justify-between">
                        <code class="text-zinc-600 dark:text-zinc-400">GET /.well-known/mcp-servers.json</code>
                        <span class="text-zinc-500">Agent discovery</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <code class="text-zinc-600 dark:text-zinc-400">GET /servers</code>
                        <span class="text-zinc-500">List all servers</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <code class="text-zinc-600 dark:text-zinc-400">GET /servers/{id}</code>
                        <span class="text-zinc-500">Server details + tools</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <code class="text-zinc-600 dark:text-zinc-400">POST /tools/call</code>
                        <span class="text-zinc-500">Execute a tool</span>
                    </div>
                    <div class="flex items-center justify-between">
                        <code class="text-zinc-600 dark:text-zinc-400">GET /resources/{uri}</code>
                        <span class="text-zinc-500">Read a resource</span>
                    </div>
                </div>
            </div>
        </div>

        <!-- Code Examples -->
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <div class="flex items-center space-x-3 mb-4">
                <div class="p-2 bg-violet-100 dark:bg-violet-900/30 rounded-lg">
                    <flux:icon.code-bracket class="w-6 h-6 text-violet-600 dark:text-violet-400" />
                </div>
                <div>
                    <h2 class="text-xl font-semibold text-zinc-900 dark:text-white">Code Examples</h2>
                </div>
            </div>

            <div class="space-y-6">
                <!-- Python -->
                <div>
                    <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">Python</h3>
                    <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm"><code class="text-emerald-400">import requests

resp = requests.post(
    "{{ $mcpUrl }}/tools/call",
    headers={"Authorization": "Bearer hk_your_key"},
    json={
        "server": "openbrain",
        "tool": "brain_recall",
        "arguments": {"query": "recent decisions"}
    }
)
print(resp.json())</code></pre>
                </div>

                <!-- JavaScript -->
                <div>
                    <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">JavaScript</h3>
                    <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm"><code class="text-emerald-400">const resp = await fetch("{{ $mcpUrl }}/tools/call", {
  method: "POST",
  headers: {
    "Authorization": "Bearer hk_your_key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    server: "openbrain",
    tool: "brain_recall",
    arguments: { query: "recent decisions" },
  }),
});
const data = await resp.json();</code></pre>
                </div>
            </div>
        </div>

        <!-- Authentication Methods -->
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-xl font-semibold text-zinc-900 dark:text-white mb-4">Authentication</h2>

            <div class="space-y-4">
                <div class="p-4 bg-zinc-50 dark:bg-zinc-900/50 rounded-lg">
                    <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">Authorization Header (Recommended)</h3>
                    <pre class="bg-zinc-100 dark:bg-zinc-900 rounded p-2 text-sm"><code class="text-zinc-800 dark:text-zinc-200">Authorization: Bearer hk_abc123_your_key_here</code></pre>
                </div>

                <div class="p-4 bg-zinc-50 dark:bg-zinc-900/50 rounded-lg">
                    <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-2">X-API-Key Header</h3>
                    <pre class="bg-zinc-100 dark:bg-zinc-900 rounded p-2 text-sm"><code class="text-zinc-800 dark:text-zinc-200">X-API-Key: hk_abc123_your_key_here</code></pre>
                </div>
            </div>

            <div class="mt-4 p-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
                <h4 class="text-sm font-semibold text-amber-800 dark:text-amber-300 mb-1">Server-scoped keys</h4>
                <p class="text-sm text-amber-700 dark:text-amber-400">
                    API keys can be restricted to specific MCP servers. If you get a 403 error,
                    check your key's server scopes in your admin dashboard.
                </p>
            </div>

            <div class="mt-4 p-4 bg-cyan-50 dark:bg-cyan-900/20 rounded-lg border border-cyan-200 dark:border-cyan-800">
                <h4 class="text-sm font-semibold text-cyan-800 dark:text-cyan-300 mb-1">Rate limiting</h4>
                <p class="text-sm text-cyan-700 dark:text-cyan-400">
                    Requests are rate limited to 120 per minute. Rate limit headers
                    (<code class="text-xs">X-RateLimit-Limit</code>, <code class="text-xs">X-RateLimit-Remaining</code>)
                    are included in all responses.
                </p>
            </div>
        </div>

        <!-- Discovery -->
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-xl font-semibold text-zinc-900 dark:text-white mb-4">Discovery</h2>
            <p class="text-zinc-600 dark:text-zinc-400 mb-4">
                Agents discover available servers automatically via the well-known endpoint:
            </p>
            <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm mb-4"><code class="text-emerald-400">curl {{ $mcpUrl }}/.well-known/mcp-servers.json</code></pre>
            <p class="text-sm text-zinc-500 dark:text-zinc-400">
                Returns the server registry with capabilities and connection details.
                No authentication required for discovery.
            </p>
        </div>

        <!-- Help -->
        <div class="text-center py-8">
            <p class="text-zinc-500 dark:text-zinc-400 mb-4">Need help setting up?</p>
            <div class="flex justify-center gap-4">
                <flux:button href="{{ route('mcp.servers.index') }}" icon="server-stack">
                    Browse Servers
                </flux:button>
                <flux:button href="{{ route('mcp.openapi.json') }}" icon="code-bracket" variant="ghost" target="_blank">
                    OpenAPI Spec
                </flux:button>
            </div>
        </div>
    </div>
</x-layouts::mcp>
