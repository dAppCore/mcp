<div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <flux:heading size="xl">Tool Usage Analytics</flux:heading>
            <flux:subheading>Monitor MCP tool usage patterns, performance, and errors</flux:subheading>
        </div>
        <div class="flex gap-2">
            <flux:button.group>
                <flux:button size="sm" wire:click="setDays(7)" variant="{{ $days === 7 ? 'primary' : 'ghost' }}">7 Days</flux:button>
                <flux:button size="sm" wire:click="setDays(14)" variant="{{ $days === 14 ? 'primary' : 'ghost' }}">14 Days</flux:button>
                <flux:button size="sm" wire:click="setDays(30)" variant="{{ $days === 30 ? 'primary' : 'ghost' }}">30 Days</flux:button>
            </flux:button.group>
            <flux:button icon="arrow-path" wire:click="$refresh">Refresh</flux:button>
        </div>
    </div>

    <!-- Overview Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-4">
        @include('mcp::admin.analytics.partials.stats-card', [
            'label' => 'Total Calls',
            'value' => number_format($this->overview['total_calls']),
            'color' => 'default',
        ])

        @include('mcp::admin.analytics.partials.stats-card', [
            'label' => 'Error Rate',
            'value' => $this->overview['error_rate'] . '%',
            'color' => $this->overview['error_rate'] > 10 ? 'red' : ($this->overview['error_rate'] > 5 ? 'yellow' : 'green'),
        ])

        @include('mcp::admin.analytics.partials.stats-card', [
            'label' => 'Avg Response',
            'value' => $this->formatDuration($this->overview['avg_duration_ms']),
            'color' => $this->overview['avg_duration_ms'] > 5000 ? 'yellow' : 'default',
        ])

        @include('mcp::admin.analytics.partials.stats-card', [
            'label' => 'Total Errors',
            'value' => number_format($this->overview['total_errors']),
            'color' => $this->overview['total_errors'] > 0 ? 'red' : 'default',
        ])

        @include('mcp::admin.analytics.partials.stats-card', [
            'label' => 'Unique Tools',
            'value' => $this->overview['unique_tools'],
            'color' => 'default',
        ])
    </div>

    <!-- Tabs -->
    <div class="border-b border-zinc-200 dark:border-zinc-700">
        <nav class="-mb-px flex gap-4">
            <button wire:click="setTab('overview')" class="px-4 py-2 text-sm font-medium {{ $tab === 'overview' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Overview
            </button>
            <button wire:click="setTab('tools')" class="px-4 py-2 text-sm font-medium {{ $tab === 'tools' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                All Tools
            </button>
            <button wire:click="setTab('errors')" class="px-4 py-2 text-sm font-medium {{ $tab === 'errors' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Errors
            </button>
            <button wire:click="setTab('combinations')" class="px-4 py-2 text-sm font-medium {{ $tab === 'combinations' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Combinations
            </button>
        </nav>
    </div>

    @if($tab === 'overview')
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <!-- Top Tools Chart -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Top 10 Most Used Tools</flux:heading>
                </div>
                <div class="p-6">
                    @if($this->popularTools->isEmpty())
                        <div class="text-zinc-500 text-center py-8">No tool usage data available</div>
                    @else
                        <div class="space-y-3">
                            @php $maxCalls = $this->popularTools->first()->totalCalls ?: 1; @endphp
                            @foreach($this->popularTools as $tool)
                                <div class="flex items-center gap-4">
                                    <div class="w-32 truncate text-sm font-mono" title="{{ $tool->toolName }}">
                                        {{ $tool->toolName }}
                                    </div>
                                    <div class="flex-1">
                                        <div class="h-6 bg-zinc-100 dark:bg-zinc-700 rounded-full overflow-hidden">
                                            <div class="h-full {{ $tool->errorRate > 10 ? 'bg-red-500' : 'bg-blue-500' }} rounded-full transition-all"
                                                 style="width: {{ ($tool->totalCalls / $maxCalls) * 100 }}%">
                                            </div>
                                        </div>
                                    </div>
                                    <div class="w-20 text-right text-sm">
                                        {{ number_format($tool->totalCalls) }}
                                    </div>
                                    <div class="w-16 text-right text-sm {{ $tool->errorRate > 10 ? 'text-red-600' : ($tool->errorRate > 5 ? 'text-yellow-600' : 'text-green-600') }}">
                                        {{ $tool->errorRate }}%
                                    </div>
                                </div>
                            @endforeach
                        </div>
                    @endif
                </div>
            </div>

            <!-- Error-Prone Tools -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Tools with Highest Error Rates</flux:heading>
                </div>
                <div class="p-6">
                    @if($this->errorProneTools->isEmpty())
                        <div class="text-green-600 text-center py-8">All tools are healthy - no significant errors</div>
                    @else
                        <div class="space-y-3">
                            @foreach($this->errorProneTools as $tool)
                                <div class="flex items-center justify-between p-3 rounded-lg {{ $tool->errorRate > 20 ? 'bg-red-50 dark:bg-red-900/20' : 'bg-yellow-50 dark:bg-yellow-900/20' }}">
                                    <div>
                                        <a href="{{ route('admin.mcp.analytics.tool', ['name' => $tool->toolName]) }}"
                                           class="font-mono text-sm hover:underline">
                                            {{ $tool->toolName }}
                                        </a>
                                        <div class="text-xs text-zinc-500">
                                            {{ number_format($tool->errorCount) }} errors / {{ number_format($tool->totalCalls) }} calls
                                        </div>
                                    </div>
                                    <flux:badge :color="$tool->errorRate > 20 ? 'red' : 'yellow'">
                                        {{ $tool->errorRate }}% errors
                                    </flux:badge>
                                </div>
                            @endforeach
                        </div>
                    @endif
                </div>
            </div>
        </div>
    @endif

    @if($tab === 'tools')
        <!-- All Tools Table -->
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700 flex justify-between items-center">
                <flux:heading>All Tools</flux:heading>
                <flux:subheading>{{ $this->sortedTools->count() }} tools</flux:subheading>
            </div>
            <div class="overflow-x-auto">
                @include('mcp::admin.analytics.partials.tool-table', ['tools' => $this->sortedTools])
            </div>
        </div>
    @endif

    @if($tab === 'errors')
        <!-- Error-Prone Tools List -->
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <flux:heading>Error Analysis</flux:heading>
            </div>
            <div class="p-6">
                @if($this->errorProneTools->isEmpty())
                    <div class="text-green-600 text-center py-8">
                        <div class="text-4xl mb-2">&#10003;</div>
                        All tools are healthy - no significant errors detected
                    </div>
                @else
                    <div class="space-y-4">
                        @foreach($this->errorProneTools as $tool)
                            <div class="p-4 rounded-lg border {{ $tool->errorRate > 20 ? 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/10' : 'border-yellow-200 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/10' }}">
                                <div class="flex items-center justify-between mb-2">
                                    <a href="{{ route('admin.mcp.analytics.tool', ['name' => $tool->toolName]) }}"
                                       class="font-mono font-medium hover:underline">
                                        {{ $tool->toolName }}
                                    </a>
                                    <flux:badge :color="$tool->errorRate > 20 ? 'red' : 'yellow'" size="lg">
                                        {{ $tool->errorRate }}% Error Rate
                                    </flux:badge>
                                </div>
                                <div class="grid grid-cols-4 gap-4 text-sm">
                                    <div>
                                        <span class="text-zinc-500">Total Calls:</span>
                                        <span class="font-medium ml-1">{{ number_format($tool->totalCalls) }}</span>
                                    </div>
                                    <div>
                                        <span class="text-zinc-500">Errors:</span>
                                        <span class="font-medium text-red-600 ml-1">{{ number_format($tool->errorCount) }}</span>
                                    </div>
                                    <div>
                                        <span class="text-zinc-500">Avg Duration:</span>
                                        <span class="font-medium ml-1">{{ $this->formatDuration($tool->avgDurationMs) }}</span>
                                    </div>
                                    <div>
                                        <span class="text-zinc-500">Max Duration:</span>
                                        <span class="font-medium ml-1">{{ $this->formatDuration($tool->maxDurationMs) }}</span>
                                    </div>
                                </div>
                            </div>
                        @endforeach
                    </div>
                @endif
            </div>
        </div>
    @endif

    @if($tab === 'combinations')
        <!-- Tool Combinations -->
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <flux:heading>Popular Tool Combinations</flux:heading>
                <flux:subheading>Tools frequently used together in the same session</flux:subheading>
            </div>
            <div class="p-6">
                @if($this->toolCombinations->isEmpty())
                    <div class="text-zinc-500 text-center py-8">No tool combination data available yet</div>
                @else
                    <div class="space-y-3">
                        @foreach($this->toolCombinations as $combo)
                            <div class="flex items-center justify-between p-3 rounded-lg bg-zinc-50 dark:bg-zinc-700/50">
                                <div class="flex items-center gap-2">
                                    <span class="font-mono text-sm">{{ $combo['tool_a'] }}</span>
                                    <span class="text-zinc-400">+</span>
                                    <span class="font-mono text-sm">{{ $combo['tool_b'] }}</span>
                                </div>
                                <flux:badge>
                                    {{ number_format($combo['occurrences']) }} times
                                </flux:badge>
                            </div>
                        @endforeach
                    </div>
                @endif
            </div>
        </div>
    @endif
</div>
