<x-layouts::mcp>
    <x-slot:title>MCP Portal</x-slot:title>
    <x-slot:description>Connect AI agents to platform infrastructure via Model Context Protocol. Machine-readable, agent-optimised, human-friendly.</x-slot:description>

    <!-- Hero -->
    <div class="text-center mb-16">
        <h1 class="text-4xl font-bold text-zinc-900 dark:text-white mb-4">
            MCP Ecosystem
        </h1>
        <p class="text-xl text-zinc-600 dark:text-zinc-400 max-w-2xl mx-auto mb-8">
            Connect AI agents to platform infrastructure via MCP.<br>
            <span class="text-cyan-600 dark:text-cyan-400">Machine-readable</span> &bull;
            <span class="text-cyan-600 dark:text-cyan-400">Agent-optimised</span> &bull;
            <span class="text-cyan-600 dark:text-cyan-400">Human-friendly</span>
        </p>
        <div class="flex flex-wrap justify-center gap-4">
            <flux:button href="{{ route('mcp.servers.index') }}" icon="server-stack" variant="primary">
                Browse Servers
            </flux:button>
            <flux:button href="{{ route('mcp.connect') }}" icon="document-text" variant="ghost">
                Setup Guide
            </flux:button>
        </div>
    </div>

    <!-- Developer Tools -->
    <section class="mb-16">
        <h2 class="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">Developer Tools</h2>
        <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-4">
            <a href="{{ route('mcp.servers.index') }}"
               class="p-4 bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 dark:hover:border-cyan-500 transition-colors">
                <div class="flex items-center space-x-3">
                    <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
                        <flux:icon.server-stack class="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                    </div>
                    <div>
                        <h3 class="font-semibold text-zinc-900 dark:text-white">Server Registry</h3>
                        <p class="text-xs text-zinc-500 dark:text-zinc-400">Browse available servers</p>
                    </div>
                </div>
            </a>

            <a href="{{ route('mcp.connect') }}"
               class="p-4 bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 dark:hover:border-cyan-500 transition-colors">
                <div class="flex items-center space-x-3">
                    <div class="p-2 bg-violet-100 dark:bg-violet-900/30 rounded-lg">
                        <flux:icon.document-text class="w-5 h-5 text-violet-600 dark:text-violet-400" />
                    </div>
                    <div>
                        <h3 class="font-semibold text-zinc-900 dark:text-white">Setup Guide</h3>
                        <p class="text-xs text-zinc-500 dark:text-zinc-400">Connect to MCP servers</p>
                    </div>
                </div>
            </a>

            <a href="{{ route('mcp.openapi.json') }}" target="_blank"
               class="p-4 bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 dark:hover:border-cyan-500 transition-colors">
                <div class="flex items-center space-x-3">
                    <div class="p-2 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg">
                        <flux:icon.code-bracket class="w-5 h-5 text-cyan-600 dark:text-cyan-400" />
                    </div>
                    <div>
                        <h3 class="font-semibold text-zinc-900 dark:text-white">OpenAPI Spec</h3>
                        <p class="text-xs text-zinc-500 dark:text-zinc-400">API documentation</p>
                    </div>
                </div>
            </a>
        </div>
    </section>

    <!-- Available Servers -->
    <section class="mb-16">
        <h2 class="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">Available Servers</h2>
        <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            @forelse($servers as $server)
                <a href="{{ route('mcp.servers.show', $server['id']) }}"
                   class="block p-6 bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 hover:border-cyan-500 dark:hover:border-cyan-500 transition-colors">
                    <div class="flex items-start justify-between mb-4">
                        <div class="p-2 bg-cyan-100 dark:bg-cyan-900/30 rounded-lg">
                            @switch($server['id'])
                                @case('hosthub-agent')
                                    <flux:icon.clipboard-document-check class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('commerce')
                                    <flux:icon.credit-card class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('socialhost')
                                    <flux:icon.share class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('biohost')
                                    <flux:icon.link class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('supporthost')
                                    <flux:icon.chat-bubble-left-right class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('openbrain')
                                    <flux:icon.light-bulb class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('analyticshost')
                                    <flux:icon.chart-bar class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @case('eaas')
                                    <flux:icon.shield-check class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                                    @break
                                @default
                                    <flux:icon.server class="w-6 h-6 text-cyan-600 dark:text-cyan-400" />
                            @endswitch
                        </div>
                        @php
                            $status = $server['status'] ?? 'available';
                        @endphp
                        @if($status === 'available')
                            <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
                                Available
                            </span>
                        @elseif($status === 'beta')
                            <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                                Beta
                            </span>
                        @endif
                    </div>
                    <h3 class="text-lg font-semibold text-zinc-900 dark:text-white mb-2">
                        {{ $server['name'] }}
                    </h3>
                    <p class="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
                        {{ $server['tagline'] ?? '' }}
                    </p>
                    <div class="flex items-center space-x-4 text-xs text-zinc-500 dark:text-zinc-400">
                        @if(in_array('tools', $server['capabilities'] ?? []))
                            <span class="flex items-center">
                                <flux:icon.wrench-screwdriver class="w-4 h-4 mr-1" />
                                Tools
                            </span>
                        @endif
                        @if(in_array('resources', $server['capabilities'] ?? []))
                            <span class="flex items-center">
                                <flux:icon.document class="w-4 h-4 mr-1" />
                                Resources
                            </span>
                        @endif
                    </div>
                </a>
            @empty
                <div class="col-span-3 text-center py-12">
                    <flux:icon.server class="w-12 h-12 text-zinc-400 mx-auto mb-4" />
                    <p class="text-zinc-500 dark:text-zinc-400">No MCP servers available yet.</p>
                </div>
            @endforelse
        </div>
    </section>

    <!-- Planned Servers -->
    @if($plannedServers->isNotEmpty())
        <section class="mb-16">
            <h2 class="text-2xl font-semibold text-zinc-900 dark:text-white mb-6">Coming Soon</h2>
            <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
                @foreach($plannedServers as $server)
                    <div class="p-6 bg-zinc-100 dark:bg-zinc-800/50 rounded-xl border border-zinc-200 dark:border-zinc-700 opacity-75">
                        <div class="flex items-start justify-between mb-4">
                            <div class="p-2 bg-zinc-200 dark:bg-zinc-700 rounded-lg">
                                @switch($server['id'])
                                    @case('analyticshost')
                                        <flux:icon.chart-bar class="w-6 h-6 text-zinc-400" />
                                        @break
                                    @case('upstream')
                                        <flux:icon.arrows-up-down class="w-6 h-6 text-zinc-400" />
                                        @break
                                    @default
                                        <flux:icon.server class="w-6 h-6 text-zinc-400" />
                                @endswitch
                            </div>
                            <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-zinc-200 text-zinc-600 dark:bg-zinc-700 dark:text-zinc-400">
                                Planned
                            </span>
                        </div>
                        <h3 class="text-lg font-semibold text-zinc-600 dark:text-zinc-300 mb-2">
                            {{ $server['name'] }}
                        </h3>
                        <p class="text-sm text-zinc-500 dark:text-zinc-400 mb-4">
                            {{ $server['tagline'] ?? '' }}
                        </p>
                    </div>
                @endforeach
            </div>
        </section>
    @endif

    <!-- Quick Start -->
    @php
        $mcpUrl = request()->getSchemeAndHttpHost();
    @endphp
    <section class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-8">
        <h2 class="text-2xl font-semibold text-zinc-900 dark:text-white mb-4">Quick Start</h2>
        <p class="text-zinc-600 dark:text-zinc-400 mb-6">
            Call MCP tools via HTTP with your API key:
        </p>
        <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm mb-6"><code class="text-emerald-400">curl -X POST {{ $mcpUrl }}/tools/call \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "server": "openbrain",
    "tool": "brain_recall",
    "arguments": { "query": "recent decisions" }
  }'</code></pre>
        <div class="flex flex-wrap items-center gap-4">
            <flux:button href="{{ route('mcp.connect') }}" icon="document-text" variant="primary">
                Full Setup Guide
            </flux:button>
            <flux:button href="{{ route('mcp.openapi.json') }}" icon="code-bracket" variant="ghost" target="_blank">
                OpenAPI Spec
            </flux:button>
        </div>
    </section>
</x-layouts::mcp>
