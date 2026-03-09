{{--
MCP Audit Log Viewer.

Displays immutable audit trail for MCP tool executions.
Includes integrity verification and compliance export features.
--}}

<div class="space-y-6">
    {{-- Header --}}
    <div class="flex items-center justify-between">
        <div>
            <core:heading size="xl">{{ __('MCP Audit Log') }}</core:heading>
            <core:subheading>Immutable audit trail for tool executions with hash chain integrity</core:subheading>
        </div>
        <div class="flex items-center gap-2">
            <flux:button wire:click="verifyIntegrity" variant="ghost" icon="shield-check">
                Verify Integrity
            </flux:button>
            <flux:button wire:click="openExportModal" icon="arrow-down-tray">
                Export
            </flux:button>
        </div>
    </div>

    {{-- Stats Cards --}}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Total Entries</div>
            <div class="mt-1 text-2xl font-semibold text-zinc-900 dark:text-white">
                {{ number_format($this->stats['total']) }}
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Success Rate</div>
            <div class="mt-1 text-2xl font-semibold text-green-600 dark:text-green-400">
                {{ $this->stats['success_rate'] }}%
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Failed Calls</div>
            <div class="mt-1 text-2xl font-semibold text-red-600 dark:text-red-400">
                {{ number_format($this->stats['failed']) }}
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Sensitive Calls</div>
            <div class="mt-1 text-2xl font-semibold text-amber-600 dark:text-amber-400">
                {{ number_format($this->stats['sensitive_calls']) }}
            </div>
        </div>
    </div>

    {{-- Filters --}}
    <div class="flex flex-wrap items-end gap-4">
        <div class="flex-1 min-w-[200px]">
            <flux:input wire:model.live.debounce.300ms="search" placeholder="Search..." icon="magnifying-glass" />
        </div>
        <flux:select wire:model.live="tool" placeholder="All tools">
            <flux:select.option value="">All tools</flux:select.option>
            @foreach ($this->tools as $toolName)
                <flux:select.option value="{{ $toolName }}">{{ $toolName }}</flux:select.option>
            @endforeach
        </flux:select>
        <flux:select wire:model.live="workspace" placeholder="All workspaces">
            <flux:select.option value="">All workspaces</flux:select.option>
            @foreach ($this->workspaces as $ws)
                <flux:select.option value="{{ $ws->id }}">{{ $ws->name }}</flux:select.option>
            @endforeach
        </flux:select>
        <flux:select wire:model.live="status" placeholder="All statuses">
            <flux:select.option value="">All statuses</flux:select.option>
            <flux:select.option value="success">Success</flux:select.option>
            <flux:select.option value="failed">Failed</flux:select.option>
        </flux:select>
        <flux:select wire:model.live="sensitivity" placeholder="All sensitivity">
            <flux:select.option value="">All sensitivity</flux:select.option>
            <flux:select.option value="sensitive">Sensitive only</flux:select.option>
            <flux:select.option value="normal">Normal only</flux:select.option>
        </flux:select>
        <flux:input type="date" wire:model.live="dateFrom" placeholder="From" />
        <flux:input type="date" wire:model.live="dateTo" placeholder="To" />
        @if($search || $tool || $workspace || $status || $sensitivity || $dateFrom || $dateTo)
            <flux:button wire:click="clearFilters" variant="ghost" size="sm" icon="x-mark">Clear</flux:button>
        @endif
    </div>

    {{-- Audit Log Table --}}
    <flux:table>
        <flux:table.columns>
            <flux:table.column>ID</flux:table.column>
            <flux:table.column>Time</flux:table.column>
            <flux:table.column>Tool</flux:table.column>
            <flux:table.column>Workspace</flux:table.column>
            <flux:table.column>Status</flux:table.column>
            <flux:table.column>Sensitivity</flux:table.column>
            <flux:table.column>Actor</flux:table.column>
            <flux:table.column>Duration</flux:table.column>
            <flux:table.column></flux:table.column>
        </flux:table.columns>

        <flux:table.rows>
            @forelse ($this->entries as $entry)
                <flux:table.row wire:key="entry-{{ $entry->id }}">
                    <flux:table.cell class="font-mono text-xs text-zinc-500">
                        #{{ $entry->id }}
                    </flux:table.cell>
                    <flux:table.cell class="text-sm text-zinc-500 whitespace-nowrap">
                        {{ $entry->created_at->format('M j, Y H:i:s') }}
                    </flux:table.cell>
                    <flux:table.cell>
                        <div class="font-medium text-zinc-900 dark:text-white">{{ $entry->tool_name }}</div>
                        <div class="text-xs text-zinc-500">{{ $entry->server_id }}</div>
                    </flux:table.cell>
                    <flux:table.cell class="text-sm">
                        @if($entry->workspace)
                            {{ $entry->workspace->name }}
                        @else
                            <span class="text-zinc-400">-</span>
                        @endif
                    </flux:table.cell>
                    <flux:table.cell>
                        <flux:badge size="sm" :color="$entry->success ? 'green' : 'red'">
                            {{ $entry->success ? 'Success' : 'Failed' }}
                        </flux:badge>
                    </flux:table.cell>
                    <flux:table.cell>
                        @if($entry->is_sensitive)
                            <flux:badge size="sm" color="amber" icon="exclamation-triangle">
                                Sensitive
                            </flux:badge>
                        @else
                            <span class="text-zinc-400 text-xs">-</span>
                        @endif
                    </flux:table.cell>
                    <flux:table.cell class="text-sm">
                        {{ $entry->getActorDisplay() }}
                        @if($entry->actor_ip)
                            <div class="text-xs text-zinc-400">{{ $entry->actor_ip }}</div>
                        @endif
                    </flux:table.cell>
                    <flux:table.cell class="text-sm text-zinc-500">
                        {{ $entry->getDurationForHumans() }}
                    </flux:table.cell>
                    <flux:table.cell>
                        <flux:button wire:click="viewEntry({{ $entry->id }})" variant="ghost" size="xs" icon="eye">
                            View
                        </flux:button>
                    </flux:table.cell>
                </flux:table.row>
            @empty
                <flux:table.row>
                    <flux:table.cell colspan="9">
                        <div class="flex flex-col items-center py-12">
                            <div class="w-16 h-16 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-4">
                                <flux:icon name="document-magnifying-glass" class="size-8 text-zinc-400" />
                            </div>
                            <flux:heading size="lg">No audit entries found</flux:heading>
                            <flux:subheading class="mt-1">Audit logs will appear here as tools are executed.</flux:subheading>
                        </div>
                    </flux:table.cell>
                </flux:table.row>
            @endforelse
        </flux:table.rows>
    </flux:table>

    @if($this->entries->hasPages())
        <div class="border-t border-zinc-200 px-6 py-4 dark:border-zinc-700">
            {{ $this->entries->links() }}
        </div>
    @endif

    {{-- Entry Detail Modal --}}
    @if($this->selectedEntry)
        <flux:modal wire:model="selectedEntryId" name="entry-detail" class="max-w-3xl">
            <div class="space-y-6">
                <div class="flex items-center justify-between">
                    <flux:heading size="lg">Audit Entry #{{ $this->selectedEntry->id }}</flux:heading>
                    <flux:button wire:click="closeEntryDetail" variant="ghost" icon="x-mark" />
                </div>

                {{-- Integrity Status --}}
                @php
                    $integrity = $this->selectedEntry->getIntegrityStatus();
                @endphp
                <div class="rounded-lg p-4 {{ $integrity['valid'] ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20' }}">
                    <div class="flex items-center gap-2">
                        <flux:icon name="{{ $integrity['valid'] ? 'shield-check' : 'shield-exclamation' }}"
                            class="size-5 {{ $integrity['valid'] ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400' }}" />
                        <span class="font-medium {{ $integrity['valid'] ? 'text-green-700 dark:text-green-300' : 'text-red-700 dark:text-red-300' }}">
                            {{ $integrity['valid'] ? 'Integrity Verified' : 'Integrity Issues Detected' }}
                        </span>
                    </div>
                    @if(!$integrity['valid'])
                        <ul class="mt-2 text-sm text-red-600 dark:text-red-400 list-disc list-inside">
                            @foreach($integrity['issues'] as $issue)
                                <li>{{ $issue }}</li>
                            @endforeach
                        </ul>
                    @endif
                </div>

                {{-- Entry Details --}}
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Tool</div>
                        <div class="mt-1 font-medium">{{ $this->selectedEntry->tool_name }}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Server</div>
                        <div class="mt-1">{{ $this->selectedEntry->server_id }}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Timestamp</div>
                        <div class="mt-1">{{ $this->selectedEntry->created_at->format('Y-m-d H:i:s.u') }}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Duration</div>
                        <div class="mt-1">{{ $this->selectedEntry->getDurationForHumans() }}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Status</div>
                        <div class="mt-1">
                            <flux:badge :color="$this->selectedEntry->success ? 'green' : 'red'">
                                {{ $this->selectedEntry->success ? 'Success' : 'Failed' }}
                            </flux:badge>
                        </div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Actor</div>
                        <div class="mt-1">{{ $this->selectedEntry->getActorDisplay() }}</div>
                    </div>
                </div>

                @if($this->selectedEntry->is_sensitive)
                    <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
                        <div class="flex items-center gap-2 text-amber-700 dark:text-amber-300">
                            <flux:icon name="exclamation-triangle" class="size-5" />
                            <span class="font-medium">Sensitive Tool</span>
                        </div>
                        <p class="mt-1 text-sm text-amber-600 dark:text-amber-400">
                            {{ $this->selectedEntry->sensitivity_reason ?? 'This tool is flagged as sensitive.' }}
                        </p>
                    </div>
                @endif

                @if($this->selectedEntry->error_message)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Error</div>
                        <div class="mt-1 rounded-lg bg-red-50 p-3 dark:bg-red-900/20">
                            @if($this->selectedEntry->error_code)
                                <div class="font-mono text-sm text-red-600 dark:text-red-400">
                                    {{ $this->selectedEntry->error_code }}
                                </div>
                            @endif
                            <div class="text-sm text-red-700 dark:text-red-300">
                                {{ $this->selectedEntry->error_message }}
                            </div>
                        </div>
                    </div>
                @endif

                @if($this->selectedEntry->input_params)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Input Parameters</div>
                        <pre class="mt-1 overflow-auto rounded-lg bg-zinc-100 p-3 text-xs dark:bg-zinc-800">{{ json_encode($this->selectedEntry->input_params, JSON_PRETTY_PRINT) }}</pre>
                    </div>
                @endif

                @if($this->selectedEntry->output_summary)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Output Summary</div>
                        <pre class="mt-1 overflow-auto rounded-lg bg-zinc-100 p-3 text-xs dark:bg-zinc-800">{{ json_encode($this->selectedEntry->output_summary, JSON_PRETTY_PRINT) }}</pre>
                    </div>
                @endif

                {{-- Hash Chain Info --}}
                <div class="border-t border-zinc-200 pt-4 dark:border-zinc-700">
                    <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-2">Hash Chain</div>
                    <div class="space-y-2 font-mono text-xs">
                        <div>
                            <span class="text-zinc-500">Entry Hash:</span>
                            <span class="text-zinc-700 dark:text-zinc-300 break-all">{{ $this->selectedEntry->entry_hash }}</span>
                        </div>
                        <div>
                            <span class="text-zinc-500">Previous Hash:</span>
                            <span class="text-zinc-700 dark:text-zinc-300 break-all">{{ $this->selectedEntry->previous_hash ?? '(first entry)' }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </flux:modal>
    @endif

    {{-- Integrity Verification Modal --}}
    @if($showIntegrityModal && $integrityStatus)
        <flux:modal wire:model="showIntegrityModal" name="integrity-modal" class="max-w-lg">
            <div class="space-y-6">
                <div class="flex items-center justify-between">
                    <flux:heading size="lg">Integrity Verification</flux:heading>
                    <flux:button wire:click="closeIntegrityModal" variant="ghost" icon="x-mark" />
                </div>

                <div class="rounded-lg p-6 {{ $integrityStatus['valid'] ? 'bg-green-50 dark:bg-green-900/20' : 'bg-red-50 dark:bg-red-900/20' }}">
                    <div class="flex items-center gap-3">
                        <flux:icon name="{{ $integrityStatus['valid'] ? 'shield-check' : 'shield-exclamation' }}"
                            class="size-10 {{ $integrityStatus['valid'] ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400' }}" />
                        <div>
                            <div class="text-lg font-semibold {{ $integrityStatus['valid'] ? 'text-green-700 dark:text-green-300' : 'text-red-700 dark:text-red-300' }}">
                                {{ $integrityStatus['valid'] ? 'Audit Log Verified' : 'Integrity Issues Detected' }}
                            </div>
                            <div class="text-sm {{ $integrityStatus['valid'] ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400' }}">
                                {{ number_format($integrityStatus['verified']) }} of {{ number_format($integrityStatus['total']) }} entries verified
                            </div>
                        </div>
                    </div>
                </div>

                @if(!$integrityStatus['valid'] && !empty($integrityStatus['issues']))
                    <div class="space-y-2">
                        <div class="text-sm font-medium text-zinc-700 dark:text-zinc-300">Issues Found:</div>
                        <div class="max-h-60 overflow-auto rounded-lg border border-red-200 dark:border-red-800">
                            @foreach($integrityStatus['issues'] as $issue)
                                <div class="border-b border-red-100 p-3 last:border-0 dark:border-red-900">
                                    <div class="font-medium text-red-700 dark:text-red-300">
                                        Entry #{{ $issue['id'] }}: {{ $issue['type'] }}
                                    </div>
                                    <div class="text-sm text-red-600 dark:text-red-400">
                                        {{ $issue['message'] }}
                                    </div>
                                </div>
                            @endforeach
                        </div>
                    </div>
                @endif

                <div class="flex justify-end">
                    <flux:button wire:click="closeIntegrityModal" variant="primary">
                        Close
                    </flux:button>
                </div>
            </div>
        </flux:modal>
    @endif

    {{-- Export Modal --}}
    @if($showExportModal)
        <flux:modal wire:model="showExportModal" name="export-modal" class="max-w-md">
            <div class="space-y-6">
                <div class="flex items-center justify-between">
                    <flux:heading size="lg">Export Audit Log</flux:heading>
                    <flux:button wire:click="closeExportModal" variant="ghost" icon="x-mark" />
                </div>

                <div class="space-y-4">
                    <p class="text-sm text-zinc-600 dark:text-zinc-400">
                        Export the audit log with current filters applied. The export includes integrity verification metadata.
                    </p>

                    <div>
                        <flux:label>Export Format</flux:label>
                        <flux:select wire:model="exportFormat">
                            <flux:select.option value="json">JSON (with integrity metadata)</flux:select.option>
                            <flux:select.option value="csv">CSV (data only)</flux:select.option>
                        </flux:select>
                    </div>

                    <div class="rounded-lg bg-zinc-100 p-3 text-sm dark:bg-zinc-800">
                        <div class="font-medium text-zinc-700 dark:text-zinc-300">Current Filters:</div>
                        <ul class="mt-1 text-zinc-600 dark:text-zinc-400">
                            @if($tool)
                                <li>Tool: {{ $tool }}</li>
                            @endif
                            @if($workspace)
                                <li>Workspace: {{ $this->workspaces->firstWhere('id', $workspace)?->name }}</li>
                            @endif
                            @if($dateFrom || $dateTo)
                                <li>Date: {{ $dateFrom ?: 'start' }} to {{ $dateTo ?: 'now' }}</li>
                            @endif
                            @if($sensitivity === 'sensitive')
                                <li>Sensitive only</li>
                            @endif
                            @if(!$tool && !$workspace && !$dateFrom && !$dateTo && !$sensitivity)
                                <li>All entries</li>
                            @endif
                        </ul>
                    </div>
                </div>

                <div class="flex justify-end gap-2">
                    <flux:button wire:click="closeExportModal" variant="ghost">
                        Cancel
                    </flux:button>
                    <flux:button wire:click="export" variant="primary" icon="arrow-down-tray">
                        Download
                    </flux:button>
                </div>
            </div>
        </flux:modal>
    @endif
</div>
