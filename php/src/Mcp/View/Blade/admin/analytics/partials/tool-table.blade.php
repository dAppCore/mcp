@props(['tools'])

<table class="w-full">
    <thead class="bg-zinc-50 dark:bg-zinc-700">
        <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 uppercase tracking-wider cursor-pointer hover:text-zinc-700"
                wire:click="sort('toolName')">
                <div class="flex items-center gap-1">
                    Tool Name
                    @if($sortColumn === 'toolName')
                        <span class="text-blue-500">{{ $sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}</span>
                    @endif
                </div>
            </th>
            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider cursor-pointer hover:text-zinc-700"
                wire:click="sort('totalCalls')">
                <div class="flex items-center justify-end gap-1">
                    Total Calls
                    @if($sortColumn === 'totalCalls')
                        <span class="text-blue-500">{{ $sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}</span>
                    @endif
                </div>
            </th>
            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider cursor-pointer hover:text-zinc-700"
                wire:click="sort('errorCount')">
                <div class="flex items-center justify-end gap-1">
                    Errors
                    @if($sortColumn === 'errorCount')
                        <span class="text-blue-500">{{ $sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}</span>
                    @endif
                </div>
            </th>
            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider cursor-pointer hover:text-zinc-700"
                wire:click="sort('errorRate')">
                <div class="flex items-center justify-end gap-1">
                    Error Rate
                    @if($sortColumn === 'errorRate')
                        <span class="text-blue-500">{{ $sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}</span>
                    @endif
                </div>
            </th>
            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider cursor-pointer hover:text-zinc-700"
                wire:click="sort('avgDurationMs')">
                <div class="flex items-center justify-end gap-1">
                    Avg Duration
                    @if($sortColumn === 'avgDurationMs')
                        <span class="text-blue-500">{{ $sortDirection === 'asc' ? '&#9650;' : '&#9660;' }}</span>
                    @endif
                </div>
            </th>
            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">
                Min / Max
            </th>
            <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 uppercase tracking-wider">
                Actions
            </th>
        </tr>
    </thead>
    <tbody class="divide-y divide-zinc-100 dark:divide-zinc-700">
        @forelse($tools as $tool)
            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-700/50">
                <td class="px-6 py-4 whitespace-nowrap">
                    <a href="{{ route('admin.mcp.analytics.tool', ['name' => $tool->toolName]) }}"
                       class="font-mono text-sm text-blue-600 hover:underline">
                        {{ $tool->toolName }}
                    </a>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    {{ number_format($tool->totalCalls) }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm {{ $tool->errorCount > 0 ? 'text-red-600' : 'text-zinc-500' }}">
                    {{ number_format($tool->errorCount) }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                    <span class="px-2 py-1 text-xs font-medium rounded {{ $tool->errorRate > 10 ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' : ($tool->errorRate > 5 ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' : 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200') }}">
                        {{ $tool->errorRate }}%
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm {{ $tool->avgDurationMs > 5000 ? 'text-yellow-600' : '' }}">
                    {{ $this->formatDuration($tool->avgDurationMs) }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm text-zinc-500">
                    {{ $this->formatDuration($tool->minDurationMs) }} / {{ $this->formatDuration($tool->maxDurationMs) }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right">
                    <a href="{{ route('admin.mcp.analytics.tool', ['name' => $tool->toolName]) }}"
                       class="text-blue-600 hover:text-blue-800 text-sm">
                        View Details
                    </a>
                </td>
            </tr>
        @empty
            <tr>
                <td colspan="7" class="px-6 py-8 text-center text-zinc-500">
                    No tool usage data available
                </td>
            </tr>
        @endforelse
    </tbody>
</table>
