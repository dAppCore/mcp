<div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <div class="flex items-center gap-2 mb-1">
                <a href="{{ route('admin.mcp.analytics') }}" class="text-zinc-500 hover:text-zinc-700">
                    Analytics
                </a>
                <span class="text-zinc-400">/</span>
            </div>
            <flux:heading size="xl" class="font-mono">{{ $toolName }}</flux:heading>
            <flux:subheading>Detailed usage analytics for this tool</flux:subheading>
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

    <!-- Overview Stats -->
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Total Calls</flux:subheading>
            <flux:heading size="xl">{{ number_format($this->stats->totalCalls) }}</flux:heading>
        </div>

        <div class="p-4 {{ $this->stats->errorRate > 10 ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800' : ($this->stats->errorRate > 5 ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800' : 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800') }} rounded-lg border">
            <flux:subheading>Error Rate</flux:subheading>
            <flux:heading size="xl" class="{{ $this->stats->errorRate > 10 ? 'text-red-600' : ($this->stats->errorRate > 5 ? 'text-yellow-600' : 'text-green-600') }}">
                {{ $this->stats->errorRate }}%
            </flux:heading>
        </div>

        <div class="p-4 {{ $this->stats->errorCount > 0 ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800' : 'bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700' }} rounded-lg border">
            <flux:subheading>Total Errors</flux:subheading>
            <flux:heading size="xl" class="{{ $this->stats->errorCount > 0 ? 'text-red-600' : '' }}">
                {{ number_format($this->stats->errorCount) }}
            </flux:heading>
        </div>

        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Avg Duration</flux:subheading>
            <flux:heading size="xl">{{ $this->formatDuration($this->stats->avgDurationMs) }}</flux:heading>
        </div>

        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Min Duration</flux:subheading>
            <flux:heading size="xl">{{ $this->formatDuration($this->stats->minDurationMs) }}</flux:heading>
        </div>

        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Max Duration</flux:subheading>
            <flux:heading size="xl">{{ $this->formatDuration($this->stats->maxDurationMs) }}</flux:heading>
        </div>
    </div>

    <!-- Usage Trend Chart -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <flux:heading>Usage Trend</flux:heading>
        </div>
        <div class="p-6">
            @if(empty($this->trends) || array_sum(array_column($this->trends, 'calls')) === 0)
                <div class="text-zinc-500 text-center py-8">No usage data available for this period</div>
            @else
                <div class="space-y-2">
                    @php
                        $maxCalls = max(array_column($this->trends, 'calls')) ?: 1;
                    @endphp
                    @foreach($this->trends as $day)
                        <div class="flex items-center gap-4">
                            <span class="w-16 text-sm text-zinc-500">{{ $day['date_formatted'] }}</span>
                            <div class="flex-1 flex items-center gap-2">
                                <div class="flex-1 bg-zinc-100 dark:bg-zinc-700 rounded-full h-5 overflow-hidden">
                                    @php
                                        $callsWidth = ($day['calls'] / $maxCalls) * 100;
                                        $errorsWidth = $day['calls'] > 0 ? ($day['errors'] / $day['calls']) * $callsWidth : 0;
                                        $successWidth = $callsWidth - $errorsWidth;
                                    @endphp
                                    <div class="h-full flex">
                                        <div class="bg-green-500 h-full" style="width: {{ $successWidth }}%"></div>
                                        <div class="bg-red-500 h-full" style="width: {{ $errorsWidth }}%"></div>
                                    </div>
                                </div>
                                <span class="w-12 text-sm text-right">{{ $day['calls'] }}</span>
                            </div>
                            <div class="w-20 text-right">
                                @if($day['calls'] > 0)
                                    <span class="text-sm {{ $day['error_rate'] > 10 ? 'text-red-600' : ($day['error_rate'] > 5 ? 'text-yellow-600' : 'text-green-600') }}">
                                        {{ round($day['error_rate'], 1) }}%
                                    </span>
                                @else
                                    <span class="text-sm text-zinc-400">-</span>
                                @endif
                            </div>
                        </div>
                    @endforeach
                </div>

                <div class="mt-6 flex items-center justify-center gap-6 text-sm">
                    <div class="flex items-center gap-2">
                        <div class="w-4 h-4 rounded bg-green-500"></div>
                        <span class="text-zinc-600 dark:text-zinc-400">Successful</span>
                    </div>
                    <div class="flex items-center gap-2">
                        <div class="w-4 h-4 rounded bg-red-500"></div>
                        <span class="text-zinc-600 dark:text-zinc-400">Errors</span>
                    </div>
                </div>
            @endif
        </div>
    </div>

    <!-- Response Time Distribution -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <flux:heading>Response Time Distribution</flux:heading>
        </div>
        <div class="p-6">
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                <div class="text-center p-4 rounded-lg bg-green-50 dark:bg-green-900/20">
                    <div class="text-sm text-zinc-600 dark:text-zinc-400 mb-1">Fastest</div>
                    <div class="text-2xl font-bold text-green-600">{{ $this->formatDuration($this->stats->minDurationMs) }}</div>
                </div>
                <div class="text-center p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20">
                    <div class="text-sm text-zinc-600 dark:text-zinc-400 mb-1">Average</div>
                    <div class="text-2xl font-bold text-blue-600">{{ $this->formatDuration($this->stats->avgDurationMs) }}</div>
                </div>
                <div class="text-center p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20">
                    <div class="text-sm text-zinc-600 dark:text-zinc-400 mb-1">Slowest</div>
                    <div class="text-2xl font-bold text-yellow-600">{{ $this->formatDuration($this->stats->maxDurationMs) }}</div>
                </div>
            </div>
        </div>
    </div>

    <!-- Daily Breakdown Table -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <flux:heading>Daily Breakdown</flux:heading>
        </div>
        <div class="overflow-x-auto">
            <table class="w-full">
                <thead class="bg-zinc-50 dark:bg-zinc-700">
                    <tr>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Date</th>
                        <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Calls</th>
                        <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Errors</th>
                        <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Error Rate</th>
                        <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Avg Duration</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-zinc-100 dark:divide-zinc-700">
                    @forelse($this->trends as $day)
                        @if($day['calls'] > 0)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-700/50">
                                <td class="px-6 py-4 whitespace-nowrap text-sm">{{ $day['date'] }}</td>
                                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">{{ number_format($day['calls']) }}</td>
                                <td class="px-6 py-4 whitespace-nowrap text-right text-sm {{ $day['errors'] > 0 ? 'text-red-600' : 'text-zinc-500' }}">{{ number_format($day['errors']) }}</td>
                                <td class="px-6 py-4 whitespace-nowrap text-right">
                                    <span class="px-2 py-1 text-xs font-medium rounded {{ $day['error_rate'] > 10 ? 'bg-red-100 text-red-800' : ($day['error_rate'] > 5 ? 'bg-yellow-100 text-yellow-800' : 'bg-green-100 text-green-800') }}">
                                        {{ round($day['error_rate'], 1) }}%
                                    </span>
                                </td>
                                <td class="px-6 py-4 whitespace-nowrap text-right text-sm">{{ $this->formatDuration($day['avg_duration_ms']) }}</td>
                            </tr>
                        @endif
                    @empty
                        <tr>
                            <td colspan="5" class="px-6 py-8 text-center text-zinc-500">
                                No data available for this period
                            </td>
                        </tr>
                    @endforelse
                </tbody>
            </table>
        </div>
    </div>
</div>
