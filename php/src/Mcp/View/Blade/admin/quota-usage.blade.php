<div class="space-y-6">
    {{-- Header --}}
    <div class="flex items-center justify-between">
        <div>
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white">MCP Usage Quota</h2>
            <p class="text-sm text-zinc-500 dark:text-zinc-400">
                Current billing period resets {{ $this->resetDate }}
            </p>
        </div>
        <button wire:click="loadQuotaData" class="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-zinc-700 bg-white border border-zinc-300 rounded-lg hover:bg-zinc-50 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-600 dark:hover:bg-zinc-700">
            <x-heroicon-o-arrow-path class="w-4 h-4" />
            Refresh
        </button>
    </div>

    {{-- Current Usage Cards --}}
    <div class="grid gap-6 md:grid-cols-2">
        {{-- Tool Calls Card --}}
        <div class="p-6 bg-white border border-zinc-200 rounded-xl dark:bg-zinc-800 dark:border-zinc-700">
            <div class="flex items-center justify-between mb-4">
                <div class="flex items-center gap-3">
                    <div class="p-2 bg-indigo-100 rounded-lg dark:bg-indigo-900/30">
                        <x-heroicon-o-wrench-screwdriver class="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                    </div>
                    <div>
                        <h3 class="font-medium text-zinc-900 dark:text-white">Tool Calls</h3>
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">Monthly usage</p>
                    </div>
                </div>
            </div>

            @if($quotaLimits['tool_calls_unlimited'] ?? false)
                <div class="flex items-baseline gap-2">
                    <span class="text-3xl font-bold text-zinc-900 dark:text-white">
                        {{ number_format($currentUsage['tool_calls_count'] ?? 0) }}
                    </span>
                    <span class="text-sm font-medium text-emerald-600 dark:text-emerald-400">Unlimited</span>
                </div>
            @else
                <div class="space-y-3">
                    <div class="flex items-baseline justify-between">
                        <span class="text-3xl font-bold text-zinc-900 dark:text-white">
                            {{ number_format($currentUsage['tool_calls_count'] ?? 0) }}
                        </span>
                        <span class="text-sm text-zinc-500 dark:text-zinc-400">
                            of {{ number_format($quotaLimits['tool_calls_limit'] ?? 0) }}
                        </span>
                    </div>
                    <div class="w-full h-2 bg-zinc-200 rounded-full dark:bg-zinc-700">
                        <div
                            class="h-2 rounded-full transition-all duration-300 {{ $this->toolCallsPercentage >= 90 ? 'bg-red-500' : ($this->toolCallsPercentage >= 75 ? 'bg-amber-500' : 'bg-indigo-500') }}"
                            style="width: {{ $this->toolCallsPercentage }}%"
                        ></div>
                    </div>
                    <p class="text-sm text-zinc-500 dark:text-zinc-400">
                        {{ number_format($remaining['tool_calls'] ?? 0) }} remaining
                    </p>
                </div>
            @endif
        </div>

        {{-- Tokens Card --}}
        <div class="p-6 bg-white border border-zinc-200 rounded-xl dark:bg-zinc-800 dark:border-zinc-700">
            <div class="flex items-center justify-between mb-4">
                <div class="flex items-center gap-3">
                    <div class="p-2 bg-purple-100 rounded-lg dark:bg-purple-900/30">
                        <x-heroicon-o-cube class="w-5 h-5 text-purple-600 dark:text-purple-400" />
                    </div>
                    <div>
                        <h3 class="font-medium text-zinc-900 dark:text-white">Tokens</h3>
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">Monthly consumption</p>
                    </div>
                </div>
            </div>

            @if($quotaLimits['tokens_unlimited'] ?? false)
                <div class="flex items-baseline gap-2">
                    <span class="text-3xl font-bold text-zinc-900 dark:text-white">
                        {{ number_format($currentUsage['total_tokens'] ?? 0) }}
                    </span>
                    <span class="text-sm font-medium text-emerald-600 dark:text-emerald-400">Unlimited</span>
                </div>
                <div class="mt-3 grid grid-cols-2 gap-4 text-sm">
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">Input:</span>
                        <span class="ml-1 font-medium text-zinc-700 dark:text-zinc-300">
                            {{ number_format($currentUsage['input_tokens'] ?? 0) }}
                        </span>
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">Output:</span>
                        <span class="ml-1 font-medium text-zinc-700 dark:text-zinc-300">
                            {{ number_format($currentUsage['output_tokens'] ?? 0) }}
                        </span>
                    </div>
                </div>
            @else
                <div class="space-y-3">
                    <div class="flex items-baseline justify-between">
                        <span class="text-3xl font-bold text-zinc-900 dark:text-white">
                            {{ number_format($currentUsage['total_tokens'] ?? 0) }}
                        </span>
                        <span class="text-sm text-zinc-500 dark:text-zinc-400">
                            of {{ number_format($quotaLimits['tokens_limit'] ?? 0) }}
                        </span>
                    </div>
                    <div class="w-full h-2 bg-zinc-200 rounded-full dark:bg-zinc-700">
                        <div
                            class="h-2 rounded-full transition-all duration-300 {{ $this->tokensPercentage >= 90 ? 'bg-red-500' : ($this->tokensPercentage >= 75 ? 'bg-amber-500' : 'bg-purple-500') }}"
                            style="width: {{ $this->tokensPercentage }}%"
                        ></div>
                    </div>
                    <div class="flex justify-between text-sm">
                        <p class="text-zinc-500 dark:text-zinc-400">
                            {{ number_format($remaining['tokens'] ?? 0) }} remaining
                        </p>
                        <div class="flex gap-3">
                            <span class="text-zinc-400 dark:text-zinc-500">
                                In: {{ number_format($currentUsage['input_tokens'] ?? 0) }}
                            </span>
                            <span class="text-zinc-400 dark:text-zinc-500">
                                Out: {{ number_format($currentUsage['output_tokens'] ?? 0) }}
                            </span>
                        </div>
                    </div>
                </div>
            @endif
        </div>
    </div>

    {{-- Usage History --}}
    @if($usageHistory->count() > 0)
        <div class="p-6 bg-white border border-zinc-200 rounded-xl dark:bg-zinc-800 dark:border-zinc-700">
            <h3 class="mb-4 font-medium text-zinc-900 dark:text-white">Usage History</h3>
            <div class="overflow-x-auto">
                <table class="w-full text-sm">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="px-4 py-3 text-left font-medium text-zinc-500 dark:text-zinc-400">Month</th>
                            <th class="px-4 py-3 text-right font-medium text-zinc-500 dark:text-zinc-400">Tool Calls</th>
                            <th class="px-4 py-3 text-right font-medium text-zinc-500 dark:text-zinc-400">Input Tokens</th>
                            <th class="px-4 py-3 text-right font-medium text-zinc-500 dark:text-zinc-400">Output Tokens</th>
                            <th class="px-4 py-3 text-right font-medium text-zinc-500 dark:text-zinc-400">Total Tokens</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($usageHistory as $record)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-700/50">
                                <td class="px-4 py-3 font-medium text-zinc-900 dark:text-white">
                                    {{ $record->month_label }}
                                </td>
                                <td class="px-4 py-3 text-right text-zinc-600 dark:text-zinc-300">
                                    {{ number_format($record->tool_calls_count) }}
                                </td>
                                <td class="px-4 py-3 text-right text-zinc-600 dark:text-zinc-300">
                                    {{ number_format($record->input_tokens) }}
                                </td>
                                <td class="px-4 py-3 text-right text-zinc-600 dark:text-zinc-300">
                                    {{ number_format($record->output_tokens) }}
                                </td>
                                <td class="px-4 py-3 text-right font-medium text-zinc-900 dark:text-white">
                                    {{ number_format($record->total_tokens) }}
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        </div>
    @endif

    {{-- Upgrade Prompt (shown when near limit) --}}
    @if(($this->toolCallsPercentage >= 80 || $this->tokensPercentage >= 80) && !($quotaLimits['tool_calls_unlimited'] ?? false))
        <div class="p-4 bg-amber-50 border border-amber-200 rounded-xl dark:bg-amber-900/20 dark:border-amber-800">
            <div class="flex items-start gap-3">
                <x-heroicon-o-exclamation-triangle class="w-5 h-5 mt-0.5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
                <div>
                    <h4 class="font-medium text-amber-800 dark:text-amber-200">Approaching usage limit</h4>
                    <p class="mt-1 text-sm text-amber-700 dark:text-amber-300">
                        You're nearing your monthly MCP quota. Consider upgrading your plan for higher limits.
                    </p>
                </div>
            </div>
        </div>
    @endif
</div>
