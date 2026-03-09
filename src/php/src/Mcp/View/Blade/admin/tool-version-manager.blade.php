{{--
MCP Tool Version Manager.

Admin interface for managing tool version lifecycles,
viewing schema changes between versions, and setting deprecation schedules.
--}}

<div class="space-y-6">
    {{-- Header --}}
    <div class="flex items-center justify-between">
        <div>
            <core:heading size="xl">{{ __('Tool Versions') }}</core:heading>
            <core:subheading>Manage MCP tool version lifecycles and backwards compatibility</core:subheading>
        </div>
        <div class="flex items-center gap-2">
            <flux:button wire:click="openRegisterModal" icon="plus">
                Register Version
            </flux:button>
        </div>
    </div>

    {{-- Stats Cards --}}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Total Versions</div>
            <div class="mt-1 text-2xl font-semibold text-zinc-900 dark:text-white">
                {{ number_format($this->stats['total_versions']) }}
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Unique Tools</div>
            <div class="mt-1 text-2xl font-semibold text-zinc-900 dark:text-white">
                {{ number_format($this->stats['total_tools']) }}
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Servers</div>
            <div class="mt-1 text-2xl font-semibold text-zinc-900 dark:text-white">
                {{ number_format($this->stats['servers']) }}
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Deprecated</div>
            <div class="mt-1 text-2xl font-semibold text-amber-600 dark:text-amber-400">
                {{ number_format($this->stats['deprecated_count']) }}
            </div>
        </div>
        <div class="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-700 dark:bg-zinc-800">
            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Sunset</div>
            <div class="mt-1 text-2xl font-semibold text-red-600 dark:text-red-400">
                {{ number_format($this->stats['sunset_count']) }}
            </div>
        </div>
    </div>

    {{-- Filters --}}
    <div class="flex flex-wrap items-end gap-4">
        <div class="flex-1 min-w-[200px]">
            <flux:input wire:model.live.debounce.300ms="search" placeholder="Search tools, servers, versions..." icon="magnifying-glass" />
        </div>
        <flux:select wire:model.live="server" placeholder="All servers">
            <flux:select.option value="">All servers</flux:select.option>
            @foreach ($this->servers as $serverId)
                <flux:select.option value="{{ $serverId }}">{{ $serverId }}</flux:select.option>
            @endforeach
        </flux:select>
        <flux:select wire:model.live="status" placeholder="All statuses">
            <flux:select.option value="">All statuses</flux:select.option>
            <flux:select.option value="latest">Latest</flux:select.option>
            <flux:select.option value="active">Active (non-latest)</flux:select.option>
            <flux:select.option value="deprecated">Deprecated</flux:select.option>
            <flux:select.option value="sunset">Sunset</flux:select.option>
        </flux:select>
        @if($search || $server || $status)
            <flux:button wire:click="clearFilters" variant="ghost" size="sm" icon="x-mark">Clear</flux:button>
        @endif
    </div>

    {{-- Versions Table --}}
    <flux:table>
        <flux:table.columns>
            <flux:table.column>Tool</flux:table.column>
            <flux:table.column>Server</flux:table.column>
            <flux:table.column>Version</flux:table.column>
            <flux:table.column>Status</flux:table.column>
            <flux:table.column>Deprecated</flux:table.column>
            <flux:table.column>Sunset</flux:table.column>
            <flux:table.column>Created</flux:table.column>
            <flux:table.column></flux:table.column>
        </flux:table.columns>

        <flux:table.rows>
            @forelse ($this->versions as $version)
                <flux:table.row wire:key="version-{{ $version->id }}">
                    <flux:table.cell>
                        <div class="font-medium text-zinc-900 dark:text-white">{{ $version->tool_name }}</div>
                        @if($version->description)
                            <div class="text-xs text-zinc-500 truncate max-w-xs">{{ $version->description }}</div>
                        @endif
                    </flux:table.cell>
                    <flux:table.cell class="text-sm text-zinc-600 dark:text-zinc-400">
                        {{ $version->server_id }}
                    </flux:table.cell>
                    <flux:table.cell>
                        <code class="rounded bg-zinc-100 px-2 py-1 text-sm font-mono dark:bg-zinc-800">
                            {{ $version->version }}
                        </code>
                    </flux:table.cell>
                    <flux:table.cell>
                        <flux:badge size="sm" :color="$this->getStatusBadgeColor($version->status)">
                            {{ ucfirst($version->status) }}
                        </flux:badge>
                    </flux:table.cell>
                    <flux:table.cell class="text-sm text-zinc-500">
                        @if($version->deprecated_at)
                            {{ $version->deprecated_at->format('M j, Y') }}
                        @else
                            <span class="text-zinc-400">-</span>
                        @endif
                    </flux:table.cell>
                    <flux:table.cell class="text-sm text-zinc-500">
                        @if($version->sunset_at)
                            <span class="{{ $version->is_sunset ? 'text-red-600 dark:text-red-400' : '' }}">
                                {{ $version->sunset_at->format('M j, Y') }}
                            </span>
                        @else
                            <span class="text-zinc-400">-</span>
                        @endif
                    </flux:table.cell>
                    <flux:table.cell class="text-sm text-zinc-500">
                        {{ $version->created_at->format('M j, Y') }}
                    </flux:table.cell>
                    <flux:table.cell>
                        <flux:dropdown>
                            <flux:button variant="ghost" size="xs" icon="ellipsis-horizontal" />
                            <flux:menu>
                                <flux:menu.item wire:click="viewVersion({{ $version->id }})" icon="eye">
                                    View Details
                                </flux:menu.item>
                                @if(!$version->is_latest && !$version->is_sunset)
                                    <flux:menu.item wire:click="markAsLatest({{ $version->id }})" icon="star">
                                        Mark as Latest
                                    </flux:menu.item>
                                @endif
                                @if(!$version->is_deprecated && !$version->is_sunset)
                                    <flux:menu.item wire:click="openDeprecateModal({{ $version->id }})" icon="archive-box">
                                        Deprecate
                                    </flux:menu.item>
                                @endif
                            </flux:menu>
                        </flux:dropdown>
                    </flux:table.cell>
                </flux:table.row>
            @empty
                <flux:table.row>
                    <flux:table.cell colspan="8">
                        <div class="flex flex-col items-center py-12">
                            <div class="w-16 h-16 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center mb-4">
                                <flux:icon name="cube" class="size-8 text-zinc-400" />
                            </div>
                            <flux:heading size="lg">No tool versions found</flux:heading>
                            <flux:subheading class="mt-1">Register tool versions to enable backwards compatibility.</flux:subheading>
                        </div>
                    </flux:table.cell>
                </flux:table.row>
            @endforelse
        </flux:table.rows>
    </flux:table>

    @if($this->versions->hasPages())
        <div class="border-t border-zinc-200 px-6 py-4 dark:border-zinc-700">
            {{ $this->versions->links() }}
        </div>
    @endif

    {{-- Version Detail Modal --}}
    @if($showVersionDetail && $this->selectedVersion)
        <flux:modal wire:model="showVersionDetail" name="version-detail" class="max-w-4xl">
            <div class="space-y-6">
                <div class="flex items-center justify-between">
                    <div>
                        <flux:heading size="lg">{{ $this->selectedVersion->tool_name }}</flux:heading>
                        <div class="flex items-center gap-2 mt-1">
                            <code class="rounded bg-zinc-100 px-2 py-1 text-sm font-mono dark:bg-zinc-800">
                                {{ $this->selectedVersion->version }}
                            </code>
                            <flux:badge size="sm" :color="$this->getStatusBadgeColor($this->selectedVersion->status)">
                                {{ ucfirst($this->selectedVersion->status) }}
                            </flux:badge>
                        </div>
                    </div>
                    <flux:button wire:click="closeVersionDetail" variant="ghost" icon="x-mark" />
                </div>

                {{-- Metadata --}}
                <div class="grid grid-cols-2 gap-4">
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Server</div>
                        <div class="mt-1">{{ $this->selectedVersion->server_id }}</div>
                    </div>
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Created</div>
                        <div class="mt-1">{{ $this->selectedVersion->created_at->format('Y-m-d H:i:s') }}</div>
                    </div>
                    @if($this->selectedVersion->deprecated_at)
                        <div>
                            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Deprecated</div>
                            <div class="mt-1 text-amber-600 dark:text-amber-400">
                                {{ $this->selectedVersion->deprecated_at->format('Y-m-d') }}
                            </div>
                        </div>
                    @endif
                    @if($this->selectedVersion->sunset_at)
                        <div>
                            <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Sunset</div>
                            <div class="mt-1 {{ $this->selectedVersion->is_sunset ? 'text-red-600 dark:text-red-400' : 'text-zinc-600' }}">
                                {{ $this->selectedVersion->sunset_at->format('Y-m-d') }}
                            </div>
                        </div>
                    @endif
                </div>

                @if($this->selectedVersion->description)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Description</div>
                        <div class="mt-1">{{ $this->selectedVersion->description }}</div>
                    </div>
                @endif

                @if($this->selectedVersion->changelog)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400">Changelog</div>
                        <div class="mt-1 prose prose-sm dark:prose-invert">
                            {!! nl2br(e($this->selectedVersion->changelog)) !!}
                        </div>
                    </div>
                @endif

                @if($this->selectedVersion->migration_notes)
                    <div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
                        <div class="flex items-center gap-2 text-blue-700 dark:text-blue-300">
                            <flux:icon name="arrow-path" class="size-5" />
                            <span class="font-medium">Migration Notes</span>
                        </div>
                        <div class="mt-2 text-sm text-blue-600 dark:text-blue-400">
                            {!! nl2br(e($this->selectedVersion->migration_notes)) !!}
                        </div>
                    </div>
                @endif

                {{-- Input Schema --}}
                @if($this->selectedVersion->input_schema)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-2">Input Schema</div>
                        <pre class="overflow-auto rounded-lg bg-zinc-100 p-4 text-xs dark:bg-zinc-800 max-h-60">{{ $this->formatSchema($this->selectedVersion->input_schema) }}</pre>
                    </div>
                @endif

                {{-- Output Schema --}}
                @if($this->selectedVersion->output_schema)
                    <div>
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-2">Output Schema</div>
                        <pre class="overflow-auto rounded-lg bg-zinc-100 p-4 text-xs dark:bg-zinc-800 max-h-60">{{ $this->formatSchema($this->selectedVersion->output_schema) }}</pre>
                    </div>
                @endif

                {{-- Version History --}}
                @if($this->versionHistory->count() > 1)
                    <div class="border-t border-zinc-200 pt-4 dark:border-zinc-700">
                        <div class="text-sm font-medium text-zinc-500 dark:text-zinc-400 mb-3">Version History</div>
                        <div class="space-y-2">
                            @foreach($this->versionHistory as $index => $historyVersion)
                                <div class="flex items-center justify-between rounded-lg border border-zinc-200 p-3 dark:border-zinc-700 {{ $historyVersion->id === $this->selectedVersion->id ? 'bg-zinc-50 dark:bg-zinc-800/50' : '' }}">
                                    <div class="flex items-center gap-3">
                                        <code class="rounded bg-zinc-100 px-2 py-1 text-sm font-mono dark:bg-zinc-800">
                                            {{ $historyVersion->version }}
                                        </code>
                                        <flux:badge size="sm" :color="$this->getStatusBadgeColor($historyVersion->status)">
                                            {{ ucfirst($historyVersion->status) }}
                                        </flux:badge>
                                        <span class="text-xs text-zinc-500">
                                            {{ $historyVersion->created_at->format('M j, Y') }}
                                        </span>
                                    </div>
                                    @if($historyVersion->id !== $this->selectedVersion->id && $index < $this->versionHistory->count() - 1)
                                        @php $nextVersion = $this->versionHistory[$index + 1] @endphp
                                        <flux:button
                                            wire:click="openCompareModal({{ $nextVersion->id }}, {{ $historyVersion->id }})"
                                            variant="ghost"
                                            size="xs"
                                            icon="arrows-right-left"
                                        >
                                            Compare
                                        </flux:button>
                                    @endif
                                </div>
                            @endforeach
                        </div>
                    </div>
                @endif
            </div>
        </flux:modal>
    @endif

    {{-- Compare Schemas Modal --}}
    @if($showCompareModal && $this->schemaComparison)
        <flux:modal wire:model="showCompareModal" name="compare-modal" class="max-w-4xl">
            <div class="space-y-6">
                <div class="flex items-center justify-between">
                    <flux:heading size="lg">Schema Comparison</flux:heading>
                    <flux:button wire:click="closeCompareModal" variant="ghost" icon="x-mark" />
                </div>

                <div class="flex items-center justify-center gap-4">
                    <div class="text-center">
                        <code class="rounded bg-zinc-100 px-3 py-1 text-sm font-mono dark:bg-zinc-800">
                            {{ $this->schemaComparison['from']->version }}
                        </code>
                    </div>
                    <flux:icon name="arrow-right" class="size-5 text-zinc-400" />
                    <div class="text-center">
                        <code class="rounded bg-zinc-100 px-3 py-1 text-sm font-mono dark:bg-zinc-800">
                            {{ $this->schemaComparison['to']->version }}
                        </code>
                    </div>
                </div>

                @php $changes = $this->schemaComparison['changes'] @endphp

                @if(empty($changes['added']) && empty($changes['removed']) && empty($changes['changed']))
                    <div class="rounded-lg bg-green-50 p-4 dark:bg-green-900/20">
                        <div class="flex items-center gap-2 text-green-700 dark:text-green-300">
                            <flux:icon name="check-circle" class="size-5" />
                            <span>No schema changes between versions</span>
                        </div>
                    </div>
                @else
                    <div class="space-y-4">
                        @if(!empty($changes['added']))
                            <div class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20">
                                <div class="font-medium text-green-700 dark:text-green-300 mb-2">
                                    Added Properties ({{ count($changes['added']) }})
                                </div>
                                <ul class="list-disc list-inside text-sm text-green-600 dark:text-green-400">
                                    @foreach($changes['added'] as $prop)
                                        <li><code>{{ $prop }}</code></li>
                                    @endforeach
                                </ul>
                            </div>
                        @endif

                        @if(!empty($changes['removed']))
                            <div class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20">
                                <div class="font-medium text-red-700 dark:text-red-300 mb-2">
                                    Removed Properties ({{ count($changes['removed']) }})
                                </div>
                                <ul class="list-disc list-inside text-sm text-red-600 dark:text-red-400">
                                    @foreach($changes['removed'] as $prop)
                                        <li><code>{{ $prop }}</code></li>
                                    @endforeach
                                </ul>
                            </div>
                        @endif

                        @if(!empty($changes['changed']))
                            <div class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20">
                                <div class="font-medium text-amber-700 dark:text-amber-300 mb-2">
                                    Changed Properties ({{ count($changes['changed']) }})
                                </div>
                                <div class="space-y-3">
                                    @foreach($changes['changed'] as $prop => $change)
                                        <div class="text-sm">
                                            <code class="font-medium text-amber-700 dark:text-amber-300">{{ $prop }}</code>
                                            <div class="mt-1 grid grid-cols-2 gap-2 text-xs">
                                                <div class="rounded bg-red-100 p-2 dark:bg-red-900/30">
                                                    <div class="text-red-600 dark:text-red-400 mb-1">Before:</div>
                                                    <pre class="text-red-700 dark:text-red-300 overflow-auto">{{ json_encode($change['from'], JSON_PRETTY_PRINT) }}</pre>
                                                </div>
                                                <div class="rounded bg-green-100 p-2 dark:bg-green-900/30">
                                                    <div class="text-green-600 dark:text-green-400 mb-1">After:</div>
                                                    <pre class="text-green-700 dark:text-green-300 overflow-auto">{{ json_encode($change['to'], JSON_PRETTY_PRINT) }}</pre>
                                                </div>
                                            </div>
                                        </div>
                                    @endforeach
                                </div>
                            </div>
                        @endif
                    </div>
                @endif

                <div class="flex justify-end">
                    <flux:button wire:click="closeCompareModal" variant="primary">Close</flux:button>
                </div>
            </div>
        </flux:modal>
    @endif

    {{-- Deprecate Modal --}}
    @if($showDeprecateModal)
        @php $deprecateVersion = \Core\Mcp\Models\McpToolVersion::find($deprecateVersionId) @endphp
        @if($deprecateVersion)
            <flux:modal wire:model="showDeprecateModal" name="deprecate-modal" class="max-w-md">
                <div class="space-y-6">
                    <div class="flex items-center justify-between">
                        <flux:heading size="lg">Deprecate Version</flux:heading>
                        <flux:button wire:click="closeDeprecateModal" variant="ghost" icon="x-mark" />
                    </div>

                    <div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
                        <div class="flex items-center gap-2 text-amber-700 dark:text-amber-300">
                            <flux:icon name="exclamation-triangle" class="size-5" />
                            <span class="font-medium">{{ $deprecateVersion->tool_name }} v{{ $deprecateVersion->version }}</span>
                        </div>
                        <p class="mt-2 text-sm text-amber-600 dark:text-amber-400">
                            Deprecated versions will show warnings to agents but remain usable until sunset.
                        </p>
                    </div>

                    <div>
                        <flux:label>Sunset Date (optional)</flux:label>
                        <flux:input
                            type="date"
                            wire:model="deprecateSunsetDate"
                            :min="now()->addDay()->format('Y-m-d')"
                        />
                        <flux:description class="mt-1">
                            After this date, the version will be blocked and return errors.
                        </flux:description>
                    </div>

                    <div class="flex justify-end gap-2">
                        <flux:button wire:click="closeDeprecateModal" variant="ghost">Cancel</flux:button>
                        <flux:button wire:click="deprecateVersion" variant="primary" color="amber">
                            Deprecate Version
                        </flux:button>
                    </div>
                </div>
            </flux:modal>
        @endif
    @endif

    {{-- Register Version Modal --}}
    @if($showRegisterModal)
        <flux:modal wire:model="showRegisterModal" name="register-modal" class="max-w-2xl">
            <div class="space-y-6">
                <div class="flex items-center justify-between">
                    <flux:heading size="lg">Register Tool Version</flux:heading>
                    <flux:button wire:click="closeRegisterModal" variant="ghost" icon="x-mark" />
                </div>

                <form wire:submit="registerVersion" class="space-y-4">
                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <flux:label>Server ID</flux:label>
                            <flux:input
                                wire:model="registerServer"
                                placeholder="e.g., hub-agent"
                                required
                            />
                            @error('registerServer') <flux:error>{{ $message }}</flux:error> @enderror
                        </div>
                        <div>
                            <flux:label>Tool Name</flux:label>
                            <flux:input
                                wire:model="registerTool"
                                placeholder="e.g., query_database"
                                required
                            />
                            @error('registerTool') <flux:error>{{ $message }}</flux:error> @enderror
                        </div>
                    </div>

                    <div class="grid grid-cols-2 gap-4">
                        <div>
                            <flux:label>Version (semver)</flux:label>
                            <flux:input
                                wire:model="registerVersion"
                                placeholder="1.0.0"
                                required
                            />
                            @error('registerVersion') <flux:error>{{ $message }}</flux:error> @enderror
                        </div>
                        <div class="flex items-end">
                            <flux:checkbox wire:model="registerMarkLatest" label="Mark as latest version" />
                        </div>
                    </div>

                    <div>
                        <flux:label>Description</flux:label>
                        <flux:input
                            wire:model="registerDescription"
                            placeholder="Brief description of the tool"
                        />
                        @error('registerDescription') <flux:error>{{ $message }}</flux:error> @enderror
                    </div>

                    <div>
                        <flux:label>Changelog</flux:label>
                        <flux:textarea
                            wire:model="registerChangelog"
                            placeholder="What changed in this version..."
                            rows="3"
                        />
                        @error('registerChangelog') <flux:error>{{ $message }}</flux:error> @enderror
                    </div>

                    <div>
                        <flux:label>Migration Notes</flux:label>
                        <flux:textarea
                            wire:model="registerMigrationNotes"
                            placeholder="Guidance for upgrading from previous version..."
                            rows="3"
                        />
                        @error('registerMigrationNotes') <flux:error>{{ $message }}</flux:error> @enderror
                    </div>

                    <div>
                        <flux:label>Input Schema (JSON)</flux:label>
                        <flux:textarea
                            wire:model="registerInputSchema"
                            placeholder='{"type": "object", "properties": {...}}'
                            rows="6"
                            class="font-mono text-sm"
                        />
                        @error('registerInputSchema') <flux:error>{{ $message }}</flux:error> @enderror
                    </div>

                    <div class="flex justify-end gap-2 pt-4 border-t border-zinc-200 dark:border-zinc-700">
                        <flux:button type="button" wire:click="closeRegisterModal" variant="ghost">Cancel</flux:button>
                        <flux:button type="submit" variant="primary">Register Version</flux:button>
                    </div>
                </form>
            </div>
        </flux:modal>
    @endif
</div>
