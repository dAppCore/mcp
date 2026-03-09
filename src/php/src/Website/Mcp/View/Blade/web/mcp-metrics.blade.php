<div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <flux:heading size="xl">MCP Agent Metrics</flux:heading>
            <flux:subheading>Monitor tool usage, performance, and errors</flux:subheading>
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
            <flux:heading size="xl">{{ number_format($this->overview['total_calls']) }}</flux:heading>
            @if($this->overview['calls_trend_percent'] != 0)
                <span class="text-sm {{ $this->overview['calls_trend_percent'] > 0 ? 'text-green-600' : 'text-red-600' }}">
                    {{ $this->overview['calls_trend_percent'] > 0 ? '+' : '' }}{{ $this->overview['calls_trend_percent'] }}%
                </span>
            @endif
        </div>
        <div class="p-4 {{ $this->overview['success_rate'] >= 95 ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800' : ($this->overview['success_rate'] >= 80 ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800' : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800') }} rounded-lg border">
            <flux:subheading>Success Rate</flux:subheading>
            <flux:heading size="xl" class="{{ $this->overview['success_rate'] >= 95 ? 'text-green-600' : ($this->overview['success_rate'] >= 80 ? 'text-yellow-600' : 'text-red-600') }}">{{ $this->overview['success_rate'] }}%</flux:heading>
        </div>
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Successful</flux:subheading>
            <flux:heading size="xl" class="text-green-600">{{ number_format($this->overview['success_calls']) }}</flux:heading>
        </div>
        <div class="p-4 {{ $this->overview['error_calls'] > 0 ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800' : 'bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700' }} rounded-lg border">
            <flux:subheading>Errors</flux:subheading>
            <flux:heading size="xl" class="{{ $this->overview['error_calls'] > 0 ? 'text-red-600' : '' }}">{{ number_format($this->overview['error_calls']) }}</flux:heading>
        </div>
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Avg Duration</flux:subheading>
            <flux:heading size="xl">{{ $this->overview['avg_duration_ms'] < 1000 ? $this->overview['avg_duration_ms'] . 'ms' : round($this->overview['avg_duration_ms'] / 1000, 2) . 's' }}</flux:heading>
        </div>
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Unique Tools</flux:subheading>
            <flux:heading size="xl">{{ $this->overview['unique_tools'] }}</flux:heading>
        </div>
    </div>

    <!-- Tabs -->
    <div class="border-b border-zinc-200 dark:border-zinc-700">
        <nav class="-mb-px flex gap-4">
            <button wire:click="setTab('overview')" class="px-4 py-2 text-sm font-medium {{ $activeTab === 'overview' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Overview
            </button>
            <button wire:click="setTab('performance')" class="px-4 py-2 text-sm font-medium {{ $activeTab === 'performance' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Performance
            </button>
            <button wire:click="setTab('errors')" class="px-4 py-2 text-sm font-medium {{ $activeTab === 'errors' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Errors
            </button>
            <button wire:click="setTab('activity')" class="px-4 py-2 text-sm font-medium {{ $activeTab === 'activity' ? 'border-b-2 border-blue-500 text-blue-600' : 'text-zinc-500 hover:text-zinc-700' }}">
                Activity Feed
            </button>
        </nav>
    </div>

    @if($activeTab === 'overview')
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <!-- Daily Trend -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Daily Call Volume</flux:heading>
                </div>
                <div class="p-6">
                    <div class="space-y-2">
                        @foreach($this->dailyTrend as $day)
                            <div class="flex items-center gap-4">
                                <span class="w-16 text-sm text-zinc-500">{{ $day['date_formatted'] }}</span>
                                <div class="flex-1 flex items-center gap-2">
                                    <div class="flex-1 bg-zinc-100 dark:bg-zinc-700 rounded-full h-4 overflow-hidden">
                                        @php
                                            $maxCalls = collect($this->dailyTrend)->max('total_calls') ?: 1;
                                            $successWidth = ($day['total_success'] / $maxCalls) * 100;
                                            $errorWidth = ($day['total_errors'] / $maxCalls) * 100;
                                        @endphp
                                        <div class="h-full flex">
                                            <div class="bg-green-500 h-full" style="width: {{ $successWidth }}%"></div>
                                            <div class="bg-red-500 h-full" style="width: {{ $errorWidth }}%"></div>
                                        </div>
                                    </div>
                                    <span class="w-16 text-sm text-right">{{ $day['total_calls'] }}</span>
                                </div>
                            </div>
                        @endforeach
                    </div>
                </div>
            </div>

            <!-- Top Tools -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Top Tools</flux:heading>
                </div>
                <div class="p-6">
                    <div class="space-y-3">
                        @forelse($this->topTools as $tool)
                            <div class="flex items-center justify-between">
                                <div>
                                    <span class="font-mono text-sm">{{ $tool->tool_name }}</span>
                                    <span class="text-xs text-zinc-500 ml-2">{{ $tool->server_id }}</span>
                                </div>
                                <div class="flex items-center gap-4">
                                    <span class="text-sm {{ $tool->success_rate >= 95 ? 'text-green-600' : ($tool->success_rate >= 80 ? 'text-yellow-600' : 'text-red-600') }}">
                                        {{ $tool->success_rate }}%
                                    </span>
                                    <span class="text-sm font-medium">{{ number_format($tool->total_calls) }}</span>
                                </div>
                            </div>
                        @empty
                            <div class="text-zinc-500 text-center py-4">No tool calls recorded yet</div>
                        @endforelse
                    </div>
                </div>
            </div>

            <!-- Server Stats -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Server Breakdown</flux:heading>
                </div>
                <div class="p-6">
                    @forelse($this->serverStats as $server)
                        <div class="flex items-center justify-between py-2 border-b border-zinc-100 dark:border-zinc-700 last:border-0">
                            <div>
                                <span class="font-medium">{{ $server->server_id }}</span>
                                <span class="text-xs text-zinc-500 ml-2">{{ $server->unique_tools }} tools</span>
                            </div>
                            <div class="flex items-center gap-4">
                                <span class="text-sm text-green-600">{{ number_format($server->total_success) }}</span>
                                <span class="text-sm text-red-600">{{ number_format($server->total_errors) }}</span>
                                <span class="text-sm font-medium">{{ number_format($server->total_calls) }}</span>
                            </div>
                        </div>
                    @empty
                        <div class="text-zinc-500 text-center py-4">No servers active yet</div>
                    @endforelse
                </div>
            </div>

            <!-- Plan Activity -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Plan Activity</flux:heading>
                </div>
                <div class="p-6">
                    @forelse($this->planActivity as $plan)
                        <div class="flex items-center justify-between py-2 border-b border-zinc-100 dark:border-zinc-700 last:border-0">
                            <div>
                                <span class="font-mono text-sm">{{ $plan->plan_slug }}</span>
                                <span class="text-xs text-zinc-500 ml-2">{{ $plan->unique_tools }} tools</span>
                            </div>
                            <div class="flex items-center gap-4">
                                <span class="text-sm {{ $plan->success_rate >= 95 ? 'text-green-600' : 'text-yellow-600' }}">
                                    {{ $plan->success_rate }}%
                                </span>
                                <span class="text-sm font-medium">{{ number_format($plan->call_count) }}</span>
                            </div>
                        </div>
                    @empty
                        <div class="text-zinc-500 text-center py-4">No plan activity recorded</div>
                    @endforelse
                </div>
            </div>
        </div>
    @endif

    @if($activeTab === 'performance')
        <div class="grid grid-cols-1 gap-6">
            <!-- Tool Performance -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Tool Performance (p50 / p95 / p99)</flux:heading>
                </div>
                <div class="overflow-x-auto">
                    <table class="w-full">
                        <thead class="bg-zinc-50 dark:bg-zinc-700">
                            <tr>
                                <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Tool</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Calls</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Min</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Avg</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">p50</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">p95</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">p99</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Max</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-zinc-100 dark:divide-zinc-700">
                            @forelse($this->toolPerformance as $tool)
                                <tr>
                                    <td class="px-6 py-4 whitespace-nowrap font-mono text-sm">{{ $tool['tool_name'] }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm">{{ number_format($tool['call_count']) }}</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm text-zinc-500">{{ $tool['min_ms'] }}ms</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm">{{ round($tool['avg_ms']) }}ms</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-green-600">{{ round($tool['p50_ms']) }}ms</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-yellow-600">{{ round($tool['p95_ms']) }}ms</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-red-600">{{ round($tool['p99_ms']) }}ms</td>
                                    <td class="px-6 py-4 whitespace-nowrap text-right text-sm text-zinc-500">{{ $tool['max_ms'] }}ms</td>
                                </tr>
                            @empty
                                <tr>
                                    <td colspan="8" class="px-6 py-8 text-center text-zinc-500">No performance data recorded yet</td>
                                </tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Hourly Distribution -->
            <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <flux:heading>Hourly Distribution (Last 24 Hours)</flux:heading>
                </div>
                <div class="p-6">
                    <div class="flex items-end gap-1 h-32">
                        @php $maxHourly = collect($this->hourlyDistribution)->max('call_count') ?: 1; @endphp
                        @foreach($this->hourlyDistribution as $hour)
                            <div class="flex-1 flex flex-col items-center">
                                <div class="w-full bg-blue-500 rounded-t" style="height: {{ ($hour['call_count'] / $maxHourly) * 100 }}%"></div>
                                <span class="text-xs text-zinc-500 mt-1">{{ $hour['hour_formatted'] }}</span>
                            </div>
                        @endforeach
                    </div>
                </div>
            </div>
        </div>
    @endif

    @if($activeTab === 'errors')
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <flux:heading>Error Breakdown</flux:heading>
            </div>
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead class="bg-zinc-50 dark:bg-zinc-700">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Tool</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider">Error Code</th>
                            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">Count</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-100 dark:divide-zinc-700">
                        @forelse($this->errorBreakdown as $error)
                            <tr>
                                <td class="px-6 py-4 whitespace-nowrap font-mono text-sm">{{ $error->tool_name }}</td>
                                <td class="px-6 py-4 whitespace-nowrap">
                                    <span class="px-2 py-1 text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 rounded">
                                        {{ $error->error_code ?? 'unknown' }}
                                    </span>
                                </td>
                                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium text-red-600">{{ number_format($error->error_count) }}</td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="3" class="px-6 py-8 text-center text-green-600">No errors recorded - all systems healthy</td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    @endif

    @if($activeTab === 'activity')
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <flux:heading>Recent Activity</flux:heading>
            </div>
            <div class="divide-y divide-zinc-100 dark:divide-zinc-700">
                @forelse($this->recentCalls as $call)
                    <div class="px-6 py-4 flex items-center justify-between">
                        <div class="flex items-center gap-4">
                            <span class="w-2 h-2 rounded-full {{ $call['success'] ? 'bg-green-500' : 'bg-red-500' }}"></span>
                            <div>
                                <span class="font-mono text-sm">{{ $call['tool_name'] }}</span>
                                @if($call['plan_slug'])
                                    <span class="text-xs text-zinc-500 ml-2">@ {{ $call['plan_slug'] }}</span>
                                @endif
                                @if(!$call['success'] && $call['error_message'])
                                    <div class="text-xs text-red-600 mt-1">{{ Str::limit($call['error_message'], 80) }}</div>
                                @endif
                            </div>
                        </div>
                        <div class="flex items-center gap-4 text-sm text-zinc-500">
                            <span>{{ $call['duration'] }}</span>
                            <span>{{ $call['created_at'] }}</span>
                        </div>
                    </div>
                @empty
                    <div class="px-6 py-8 text-center text-zinc-500">No activity recorded yet</div>
                @endforelse
            </div>
        </div>
    @endif
</div>
