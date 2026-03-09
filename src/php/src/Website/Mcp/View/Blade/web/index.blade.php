<x-layouts::mcp>
    <x-slot:title>MCP Servers</x-slot:title>

    <div class="mb-8">
        <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">MCP Servers</h1>
        <p class="mt-2 text-zinc-600 dark:text-zinc-400">
            All available MCP servers for AI agent integration.
        </p>
    </div>

    <!-- Available Servers -->
    <div class="grid md:grid-cols-2 gap-6 mb-12">
        @forelse($servers as $server)
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden">
                <div class="p-6">
                    <div class="flex items-start justify-between mb-4">
                        <div class="flex items-center space-x-4">
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
                            <div>
                                <h2 class="text-xl font-semibold text-zinc-900 dark:text-white">
                                    {{ $server['name'] }}
                                </h2>
                                <p class="text-sm text-zinc-500 dark:text-zinc-400">{{ $server['id'] }}</p>
                            </div>
                        </div>
                        <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
                            {{ ucfirst($server['status'] ?? 'available') }}
                        </span>
                    </div>

                    <p class="text-zinc-600 dark:text-zinc-400 mb-4">
                        {{ $server['tagline'] ?? $server['description'] ?? '' }}
                    </p>

                    <!-- Stats -->
                    <div class="flex items-center space-x-6 text-sm text-zinc-500 dark:text-zinc-400 mb-4">
                        <span class="flex items-center">
                            <flux:icon.wrench-screwdriver class="w-4 h-4 mr-1" />
                            {{ $server['tool_count'] ?? 0 }} tools
                        </span>
                        <span class="flex items-center">
                            <flux:icon.document class="w-4 h-4 mr-1" />
                            {{ $server['resource_count'] ?? 0 }} resources
                        </span>
                        @if(($server['workflow_count'] ?? 0) > 0)
                            <span class="flex items-center">
                                <flux:icon.arrow-path class="w-4 h-4 mr-1" />
                                {{ $server['workflow_count'] }} workflows
                            </span>
                        @endif
                    </div>
                </div>

                <div class="border-t border-zinc-200 dark:border-zinc-700 px-6 py-4 bg-zinc-50 dark:bg-zinc-800/50">
                    <div class="flex items-center justify-between">
                        <a href="{{ route('mcp.servers.show', $server['id']) }}"
                           class="text-cyan-600 dark:text-cyan-400 hover:underline text-sm font-medium">
                            View Documentation →
                        </a>
                        <a href="{{ route('mcp.servers.analytics', $server['id']) }}"
                           class="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300 text-sm">
                            Analytics
                        </a>
                    </div>
                </div>
            </div>
        @empty
            <div class="col-span-2 text-center py-12 bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700">
                <flux:icon.server class="w-12 h-12 text-zinc-400 mx-auto mb-4" />
                <p class="text-zinc-500 dark:text-zinc-400">No MCP servers available.</p>
            </div>
        @endforelse
    </div>

    <!-- Planned Servers -->
    @if($plannedServers->isNotEmpty())
        <div class="mb-8">
            <h2 class="text-2xl font-semibold text-zinc-900 dark:text-white mb-4">Planned Servers</h2>
            <div class="grid md:grid-cols-3 gap-4">
                @foreach($plannedServers as $server)
                    <div class="p-4 bg-zinc-100 dark:bg-zinc-800/50 rounded-lg border border-zinc-200 dark:border-zinc-700 opacity-75">
                        <div class="flex items-center space-x-3 mb-2">
                            <div class="p-1.5 bg-zinc-200 dark:bg-zinc-700 rounded-lg">
                                @switch($server['id'])
                                    @case('upstream')
                                        <flux:icon.arrows-up-down class="w-4 h-4 text-zinc-400" />
                                        @break
                                    @case('analyticshost')
                                        <flux:icon.chart-bar class="w-4 h-4 text-zinc-400" />
                                        @break
                                    @default
                                        <flux:icon.server class="w-4 h-4 text-zinc-400" />
                                @endswitch
                            </div>
                            <h3 class="font-medium text-zinc-600 dark:text-zinc-300">{{ $server['name'] }}</h3>
                        </div>
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">{{ $server['tagline'] ?? '' }}</p>
                    </div>
                @endforeach
            </div>
        </div>
    @endif
</x-layouts::mcp>
