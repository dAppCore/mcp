<x-layouts::mcp>
    <x-slot:title>{{ $server['name'] }} Analytics</x-slot:title>

    <div class="mb-8">
        <nav class="text-sm mb-4">
            <a href="{{ route('mcp.servers.show', $server['id']) }}" class="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200">
                ← Back to {{ $server['name'] }}
            </a>
        </nav>

        <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">{{ $server['name'] }} Analytics</h1>
        <p class="mt-2 text-zinc-600 dark:text-zinc-400">
            Tool usage statistics for the last {{ $days }} days.
        </p>
    </div>

    <!-- Overview Stats -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-4">
            <p class="text-sm text-zinc-500 dark:text-zinc-400">Total Calls</p>
            <p class="text-2xl font-bold text-zinc-900 dark:text-white">{{ number_format($stats['total_calls']) }}</p>
        </div>
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-4">
            <p class="text-sm text-zinc-500 dark:text-zinc-400">Success Rate</p>
            <p class="text-2xl font-bold {{ $stats['success_rate'] >= 95 ? 'text-green-600 dark:text-green-400' : ($stats['success_rate'] >= 80 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400') }}">
                {{ $stats['success_rate'] }}%
            </p>
        </div>
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-4">
            <p class="text-sm text-zinc-500 dark:text-zinc-400">Successful</p>
            <p class="text-2xl font-bold text-green-600 dark:text-green-400">{{ number_format($stats['successful_calls']) }}</p>
        </div>
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-4">
            <p class="text-sm text-zinc-500 dark:text-zinc-400">Failed</p>
            <p class="text-2xl font-bold text-red-600 dark:text-red-400">{{ number_format($stats['failed_calls']) }}</p>
        </div>
    </div>

    <!-- Tool Breakdown -->
    @if(!empty($stats['by_tool']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Tool Usage</h2>
            <div class="space-y-3">
                @foreach($stats['by_tool'] as $tool => $data)
                    <div class="flex items-center justify-between py-2 border-b border-zinc-100 dark:border-zinc-700 last:border-0">
                        <div class="flex items-center space-x-3">
                            <code class="text-sm font-mono text-cyan-600 dark:text-cyan-400">{{ $tool }}</code>
                        </div>
                        <div class="flex items-center space-x-6 text-sm">
                            <span class="text-zinc-600 dark:text-zinc-400">{{ $data['calls'] }} calls</span>
                            <span class="{{ $data['success_rate'] >= 95 ? 'text-green-600 dark:text-green-400' : 'text-yellow-600 dark:text-yellow-400' }}">
                                {{ $data['success_rate'] }}% success
                            </span>
                            <span class="text-zinc-500 dark:text-zinc-500">{{ $data['avg_duration_ms'] }}ms avg</span>
                        </div>
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    <!-- Daily Breakdown -->
    @if(!empty($stats['by_day']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Daily Activity</h2>
            <div class="space-y-2">
                @foreach($stats['by_day'] as $date => $count)
                    <div class="flex items-center space-x-4">
                        <span class="w-24 text-sm text-zinc-500 dark:text-zinc-400">{{ $date }}</span>
                        <div class="flex-1">
                            <div class="h-4 bg-zinc-100 dark:bg-zinc-700 rounded-full overflow-hidden">
                                @php
                                    $maxCalls = max($stats['by_day']);
                                    $width = $maxCalls > 0 ? ($count / $maxCalls) * 100 : 0;
                                @endphp
                                <div class="h-full bg-cyan-500" style="width: {{ $width }}%"></div>
                            </div>
                        </div>
                        <span class="w-12 text-right text-sm text-zinc-600 dark:text-zinc-400">{{ $count }}</span>
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    <!-- Errors -->
    @if(!empty($stats['errors']))
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-8">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Error Breakdown</h2>
            <div class="space-y-2">
                @foreach($stats['errors'] as $code => $count)
                    <div class="flex items-center justify-between py-2 border-b border-zinc-100 dark:border-zinc-700 last:border-0">
                        <code class="text-sm text-red-600 dark:text-red-400">{{ $code ?: 'Unknown' }}</code>
                        <span class="text-sm text-zinc-600 dark:text-zinc-400">{{ $count }} occurrences</span>
                    </div>
                @endforeach
            </div>
        </div>
    @endif

    <!-- Time Range Selector -->
    <div class="flex items-center justify-center space-x-2 text-sm">
        <span class="text-zinc-500 dark:text-zinc-400 mr-2">Time range:</span>
        <flux:button.group>
            @foreach([7, 14, 30] as $range)
                <flux:button
                    href="{{ route('mcp.servers.analytics', ['id' => $server['id'], 'days' => $range]) }}"
                    :variant="$days == $range ? 'primary' : 'ghost'"
                    size="sm">
                    {{ $range }} days
                </flux:button>
            @endforeach
        </flux:button.group>
    </div>
</x-layouts::mcp>
