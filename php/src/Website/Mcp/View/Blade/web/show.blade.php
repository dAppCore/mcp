<x-layouts::mcp>
    <x-slot:title>{{ $server['name'] }}</x-slot:title>
    <x-slot:description>{{ $server['tagline'] ?? $server['description'] ?? '' }}</x-slot:description>

    <!-- Header -->
    <div class="mb-8">
        <nav class="text-sm mb-4">
            <a href="{{ route('mcp.servers.index') }}" class="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200">
                ← Back to Servers
            </a>
        </nav>

        <div class="flex items-start justify-between">
            <div class="flex items-start space-x-4">
                <div class="p-3 bg-cyan-100 dark:bg-cyan-900/30 rounded-xl">
                    @switch($server['id'])
                        @case('hosthub-agent')
                            <flux:icon.clipboard-document-check class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('commerce')
                            <flux:icon.credit-card class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('socialhost')
                            <flux:icon.share class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('biohost')
                            <flux:icon.link class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('supporthost')
                            <flux:icon.chat-bubble-left-right class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('openbrain')
                            <flux:icon.light-bulb class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('analyticshost')
                            <flux:icon.chart-bar class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('eaas')
                            <flux:icon.shield-check class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @case('upstream')
                            <flux:icon.arrows-up-down class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                            @break
                        @default
                            <flux:icon.server class="w-8 h-8 text-cyan-600 dark:text-cyan-400" />
                    @endswitch
                </div>
                <div>
                    <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">{{ $server['name'] }}</h1>
                    <p class="text-zinc-500 dark:text-zinc-400">{{ $server['id'] }}</p>
                </div>
            </div>
            <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
                {{ ucfirst($server['status'] ?? 'available') }}
            </span>
        </div>

        <p class="mt-4 text-lg text-zinc-600 dark:text-zinc-400">
            {{ $server['tagline'] ?? '' }}
        </p>
    </div>

    <!-- Description -->
    @if(!empty($server['description']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-3">About</h2>
            <div class="prose dark:prose-invert max-w-none">
                {!! nl2br(e($server['description'])) !!}
            </div>
        </div>
    @endif

    <!-- Use When / Don't Use When -->
    <div class="grid md:grid-cols-2 gap-6 mb-8">
        @if(!empty($server['use_when']))
            <div class="bg-green-50 dark:bg-green-900/20 rounded-xl border border-green-200 dark:border-green-800 p-6">
                <h3 class="text-lg font-semibold text-green-800 dark:text-green-300 mb-3 flex items-center">
                    <flux:icon.check-circle class="w-5 h-5 mr-2" />
                    Use when
                </h3>
                <ul class="space-y-2">
                    @foreach($server['use_when'] as $item)
                        <li class="text-green-700 dark:text-green-400 text-sm">• {{ $item }}</li>
                    @endforeach
                </ul>
            </div>
        @endif

        @if(!empty($server['dont_use_when']))
            <div class="bg-red-50 dark:bg-red-900/20 rounded-xl border border-red-200 dark:border-red-800 p-6">
                <h3 class="text-lg font-semibold text-red-800 dark:text-red-300 mb-3 flex items-center">
                    <flux:icon.x-circle class="w-5 h-5 mr-2" />
                    Don't use when
                </h3>
                <ul class="space-y-2">
                    @foreach($server['dont_use_when'] as $item)
                        <li class="text-red-700 dark:text-red-400 text-sm">• {{ $item }}</li>
                    @endforeach
                </ul>
            </div>
        @endif
    </div>

    <!-- Connection -->
    @php
        $mcpUrl = request()->getSchemeAndHttpHost();
    @endphp
    <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
        <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Connection</h2>
        <p class="text-sm text-zinc-600 dark:text-zinc-400 mb-4">
            Call tools on this server via HTTP:
        </p>
        <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm"><code class="text-emerald-400">curl -X POST {{ $mcpUrl }}/tools/call \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "server": "{{ $server['id'] }}",
    "tool": "{{ !empty($server['tools']) ? $server['tools'][0]['name'] : 'tool_name' }}",
    "arguments": {}
  }'</code></pre>
        <p class="mt-3 text-xs text-zinc-500 dark:text-zinc-400">
            <a href="{{ route('mcp.connect') }}" class="text-cyan-600 dark:text-cyan-400 hover:underline">
                Full setup guide →
            </a>
        </p>
    </div>

    <!-- Tools -->
    @if(!empty($server['tools']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
                Tools ({{ count($server['tools']) }})
            </h2>
            <div class="space-y-4">
                @foreach($server['tools'] as $tool)
                    <div class="border border-zinc-200 dark:border-zinc-700 rounded-lg p-4">
                        <div class="flex items-start justify-between mb-2">
                            <h3 class="font-mono text-sm font-semibold text-cyan-600 dark:text-cyan-400">
                                {{ $tool['name'] }}
                            </h3>
                        </div>
                        <p class="text-sm text-zinc-600 dark:text-zinc-400 mb-3">
                            {{ $tool['purpose'] ?? '' }}
                        </p>

                        @if(!empty($tool['example_prompts']))
                            <div class="mb-3">
                                <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Example prompts:</p>
                                <ul class="text-xs text-zinc-500 dark:text-zinc-400 space-y-1">
                                    @foreach(array_slice($tool['example_prompts'], 0, 3) as $prompt)
                                        <li class="italic">"{{ $prompt }}"</li>
                                    @endforeach
                                </ul>
                            </div>
                        @endif

                        @if(!empty($tool['parameters']))
                            <details class="text-sm">
                                <summary class="cursor-pointer text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300">
                                    Parameters
                                </summary>
                                <div class="mt-2 pl-4 border-l-2 border-zinc-200 dark:border-zinc-700 space-y-2">
                                    @foreach($tool['parameters'] as $name => $param)
                                        <div>
                                            <span class="font-mono text-xs text-cyan-600 dark:text-cyan-400">{{ $name }}</span>
                                            @if(!empty($param['required']))
                                                <span class="text-xs text-red-500">*</span>
                                            @endif
                                            <span class="text-xs text-zinc-500 dark:text-zinc-400 ml-2">
                                                {{ is_array($param['type'] ?? '') ? implode('|', $param['type']) : ($param['type'] ?? 'string') }}
                                            </span>
                                            @if(!empty($param['description']))
                                                <p class="text-xs text-zinc-500 dark:text-zinc-400 mt-0.5">{{ $param['description'] }}</p>
                                            @endif
                                        </div>
                                    @endforeach
                                </div>
                            </details>
                        @endif
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    <!-- Resources -->
    @if(!empty($server['resources']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
                Resources ({{ count($server['resources']) }})
            </h2>
            <div class="space-y-3">
                @foreach($server['resources'] as $resource)
                    <div class="flex items-start space-x-3 p-3 bg-zinc-50 dark:bg-zinc-900 rounded-lg">
                        <flux:icon.document class="w-5 h-5 text-zinc-400 shrink-0 mt-0.5" />
                        <div>
                            <p class="font-mono text-sm text-zinc-700 dark:text-zinc-300">{{ $resource['uri'] }}</p>
                            <p class="text-xs text-zinc-500 dark:text-zinc-400 mt-1">{{ $resource['purpose'] ?? $resource['name'] ?? '' }}</p>
                        </div>
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    <!-- Workflows -->
    @if(!empty($server['workflows']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
                Workflows ({{ count($server['workflows']) }})
            </h2>
            <div class="space-y-6">
                @foreach($server['workflows'] as $workflow)
                    <div>
                        <h3 class="font-semibold text-zinc-800 dark:text-zinc-200 mb-2">{{ $workflow['name'] }}</h3>
                        <p class="text-sm text-zinc-600 dark:text-zinc-400 mb-3">{{ $workflow['description'] ?? '' }}</p>
                        @if(!empty($workflow['steps']))
                            <ol class="space-y-2 pl-4 border-l-2 border-cyan-500/50">
                                @foreach($workflow['steps'] as $index => $step)
                                    <li class="text-sm">
                                        <span class="font-mono text-cyan-600 dark:text-cyan-400">{{ $step['action'] }}</span>
                                        @if(!empty($step['note']))
                                            <span class="text-zinc-500 dark:text-zinc-400 ml-2">— {{ $step['note'] }}</span>
                                        @endif
                                    </li>
                                @endforeach
                            </ol>
                        @endif
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    <!-- Analytics Link -->
    <div class="text-center py-8">
        <a href="{{ route('mcp.servers.analytics', $server['id']) }}"
           class="inline-flex items-center text-sm text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300">
            <flux:icon.chart-bar class="w-4 h-4 mr-2" />
            View Usage Analytics
        </a>
    </div>
</x-layouts::mcp>
