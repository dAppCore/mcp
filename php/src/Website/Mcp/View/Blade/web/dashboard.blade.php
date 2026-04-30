<div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <flux:heading size="xl">Upstream Intelligence</flux:heading>
            <flux:subheading>Track vendor updates and manage porting tasks</flux:subheading>
        </div>
        <div class="flex gap-2">
            <flux:button icon="arrow-path" wire:click="$refresh">Refresh</flux:button>
        </div>
    </div>

    <!-- Stats Cards -->
    <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Vendors</flux:subheading>
            <flux:heading size="xl">{{ $this->stats['total_vendors'] }}</flux:heading>
        </div>
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>Pending</flux:subheading>
            <flux:heading size="xl">{{ $this->stats['pending_todos'] }}</flux:heading>
        </div>
        <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg border border-green-200 dark:border-green-800">
            <flux:subheading>Quick Wins</flux:subheading>
            <flux:heading size="xl" class="text-green-600 dark:text-green-400">{{ $this->stats['quick_wins'] }}</flux:heading>
        </div>
        <div class="p-4 rounded-lg border {{ $this->stats['security_updates'] > 0 ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800' : 'bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700' }}">
            <flux:subheading>Security</flux:subheading>
            <flux:heading size="xl" class="{{ $this->stats['security_updates'] > 0 ? 'text-red-600 dark:text-red-400' : '' }}">{{ $this->stats['security_updates'] }}</flux:heading>
        </div>
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>In Progress</flux:subheading>
            <flux:heading size="xl">{{ $this->stats['in_progress'] }}</flux:heading>
        </div>
        <div class="p-4 bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <flux:subheading>This Week</flux:subheading>
            <flux:heading size="xl">{{ $this->stats['recent_releases'] }}</flux:heading>
        </div>
    </div>

    <!-- Vendors Overview -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <flux:heading>Tracked Vendors</flux:heading>
        </div>
        <div class="p-6">
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                @foreach($this->vendors as $vendor)
                    <div class="p-4 border rounded-lg dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition">
                        <div class="flex items-center gap-2 mb-2">
                            <span>{{ $vendor->getSourceTypeIcon() }}</span>
                            <flux:heading size="sm">{{ $vendor->name }}</flux:heading>
                        </div>
                        <div class="text-sm text-zinc-500 dark:text-zinc-400 space-y-1">
                            <div>{{ $vendor->vendor_name }} &middot; {{ $vendor->getSourceTypeLabel() }}</div>
                            <div>Version: <span class="font-mono">{{ $vendor->current_version ?? 'Not set' }}</span></div>
                            <div class="flex gap-4 mt-2">
                                <span>{{ $vendor->todos_count }} todos</span>
                                <span>{{ $vendor->releases_count }} releases</span>
                            </div>
                        </div>
                    </div>
                @endforeach
            </div>
        </div>
    </div>

    <!-- Filters -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-4">
        <div class="flex flex-wrap gap-4 items-center">
            <flux:select wire:model.live="vendorFilter" placeholder="All Vendors" class="w-48">
                <flux:select.option value="">All Vendors</flux:select.option>
                @foreach($this->vendors as $vendor)
                    <flux:select.option value="{{ $vendor->id }}">{{ $vendor->name }}</flux:select.option>
                @endforeach
            </flux:select>

            <flux:select wire:model.live="statusFilter" class="w-40">
                <flux:select.option value="">All Status</flux:select.option>
                <flux:select.option value="pending">Pending</flux:select.option>
                <flux:select.option value="in_progress">In Progress</flux:select.option>
                <flux:select.option value="ported">Ported</flux:select.option>
                <flux:select.option value="skipped">Skipped</flux:select.option>
            </flux:select>

            <flux:select wire:model.live="typeFilter" class="w-40">
                <flux:select.option value="">All Types</flux:select.option>
                <flux:select.option value="feature">Feature</flux:select.option>
                <flux:select.option value="bugfix">Bugfix</flux:select.option>
                <flux:select.option value="security">Security</flux:select.option>
                <flux:select.option value="ui">UI</flux:select.option>
                <flux:select.option value="block">Block</flux:select.option>
                <flux:select.option value="api">API</flux:select.option>
            </flux:select>

            <flux:select wire:model.live="effortFilter" class="w-40">
                <flux:select.option value="">All Effort</flux:select.option>
                <flux:select.option value="low">Low (&lt;1hr)</flux:select.option>
                <flux:select.option value="medium">Medium (1-4hr)</flux:select.option>
                <flux:select.option value="high">High (4+hr)</flux:select.option>
            </flux:select>

            <flux:checkbox wire:model.live="quickWinsOnly" label="Quick Wins Only" />
        </div>
    </div>

    <!-- Todos Table -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700 flex justify-between items-center">
            <flux:heading>Porting Tasks</flux:heading>
            <flux:subheading>{{ $this->todos->total() }} total</flux:subheading>
        </div>
        <div class="overflow-x-auto">
            <flux:table>
                <flux:table.columns>
                    <flux:table.column>Type</flux:table.column>
                    <flux:table.column>Title</flux:table.column>
                    <flux:table.column>Vendor</flux:table.column>
                    <flux:table.column>Priority</flux:table.column>
                    <flux:table.column>Effort</flux:table.column>
                    <flux:table.column>Status</flux:table.column>
                    <flux:table.column>Actions</flux:table.column>
                </flux:table.columns>
                <flux:table.rows>
                    @forelse($this->todos as $todo)
                        <flux:table.row>
                            <flux:table.cell>
                                <span title="{{ $todo->type }}">{{ $todo->getTypeIcon() }}</span>
                            </flux:table.cell>
                            <flux:table.cell>
                                <div class="max-w-md">
                                    <div class="font-medium">{{ $todo->title }}</div>
                                    @if($todo->description)
                                        <div class="text-sm text-zinc-500 truncate">{{ Str::limit($todo->description, 80) }}</div>
                                    @endif
                                </div>
                            </flux:table.cell>
                            <flux:table.cell>
                                <span class="text-sm">{{ $todo->vendor->name }}</span>
                            </flux:table.cell>
                            <flux:table.cell>
                                <flux:badge :color="$todo->priority >= 7 ? 'red' : ($todo->priority >= 4 ? 'yellow' : 'zinc')">
                                    {{ $todo->priority }}/10
                                </flux:badge>
                            </flux:table.cell>
                            <flux:table.cell>
                                <flux:badge :color="$todo->effort === 'low' ? 'green' : ($todo->effort === 'medium' ? 'yellow' : 'red')">
                                    {{ $todo->getEffortLabel() }}
                                </flux:badge>
                            </flux:table.cell>
                            <flux:table.cell>
                                <flux:badge :color="match($todo->status) { 'pending' => 'yellow', 'in_progress' => 'blue', 'ported' => 'green', default => 'zinc' }">
                                    {{ str_replace('_', ' ', $todo->status) }}
                                </flux:badge>
                            </flux:table.cell>
                            <flux:table.cell>
                                @if($todo->status === 'pending')
                                    <flux:button size="xs" wire:click="markInProgress({{ $todo->id }})">Start</flux:button>
                                @elseif($todo->status === 'in_progress')
                                    <div class="flex gap-1">
                                        <flux:button size="xs" variant="primary" wire:click="markPorted({{ $todo->id }})">Done</flux:button>
                                        <flux:button size="xs" variant="ghost" wire:click="markSkipped({{ $todo->id }})">Skip</flux:button>
                                    </div>
                                @endif
                            </flux:table.cell>
                        </flux:table.row>
                    @empty
                        <flux:table.row>
                            <flux:table.cell colspan="7" class="text-center py-8 text-zinc-500">
                                No todos found matching filters
                            </flux:table.cell>
                        </flux:table.row>
                    @endforelse
                </flux:table.rows>
            </flux:table>
        </div>
        @if($this->todos->hasPages())
            <div class="px-6 py-4 border-t border-zinc-200 dark:border-zinc-700">
                {{ $this->todos->links() }}
            </div>
        @endif
    </div>

    <!-- Asset Library -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Tracked Assets -->
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700 flex justify-between items-center">
                <flux:heading>Asset Library</flux:heading>
                <div class="flex gap-2 items-center">
                    @if($this->assetStats['updates_available'] > 0)
                        <flux:badge color="yellow">{{ $this->assetStats['updates_available'] }} updates</flux:badge>
                    @endif
                    <flux:badge>{{ $this->assetStats['total'] }} assets</flux:badge>
                </div>
            </div>
            <div class="p-4">
                <div class="space-y-3">
                    @foreach($this->assets as $asset)
                        <div class="flex items-center justify-between p-3 rounded-lg hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition">
                            <div class="flex items-center gap-3">
                                <span class="text-xl">{{ $asset->getTypeIcon() }}</span>
                                <div>
                                    <div class="font-medium">{{ $asset->name }}</div>
                                    <div class="text-sm text-zinc-500">
                                        @if($asset->package_name)
                                            <span class="font-mono text-xs">{{ $asset->package_name }}</span>
                                        @endif
                                    </div>
                                </div>
                            </div>
                            <div class="flex items-center gap-2">
                                <span class="text-sm">{{ $asset->getLicenseIcon() }}</span>
                                @if($asset->installed_version)
                                    <flux:badge :color="$asset->hasUpdate() ? 'yellow' : 'zinc'" size="sm">
                                        {{ $asset->installed_version }}
                                        @if($asset->hasUpdate())
                                            → {{ $asset->latest_version }}
                                        @endif
                                    </flux:badge>
                                @else
                                    <flux:badge color="zinc" size="sm">Not installed</flux:badge>
                                @endif
                            </div>
                        </div>
                    @endforeach
                </div>
            </div>
        </div>

        <!-- Pattern Library Preview -->
        <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700 flex justify-between items-center">
                <flux:heading>Pattern Library</flux:heading>
                <flux:badge>{{ $this->assetStats['patterns'] }} patterns</flux:badge>
            </div>
            <div class="p-4">
                <div class="grid grid-cols-2 gap-3">
                    @foreach($this->patterns as $pattern)
                        <div class="p-3 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:border-zinc-300 dark:hover:border-zinc-600 transition">
                            <div class="flex items-center gap-2 mb-1">
                                <span>{{ $pattern->getCategoryIcon() }}</span>
                                <span class="font-medium text-sm">{{ $pattern->name }}</span>
                            </div>
                            <div class="text-xs text-zinc-500 line-clamp-2">{{ $pattern->description }}</div>
                            <div class="flex gap-1 mt-2">
                                <flux:badge size="sm" color="zinc">{{ $pattern->language }}</flux:badge>
                                @if($pattern->is_vetted)
                                    <flux:badge size="sm" color="green">Vetted</flux:badge>
                                @endif
                            </div>
                        </div>
                    @endforeach
                </div>
            </div>
        </div>
    </div>

    <!-- Recent Activity -->
    <div class="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div class="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
            <flux:heading>Recent Activity</flux:heading>
        </div>
        <div class="p-6">
            <div class="space-y-3">
                @forelse($this->recentLogs as $log)
                    <div class="flex items-center gap-3 text-sm">
                        <span>{{ $log->getActionIcon() }}</span>
                        <span class="text-zinc-500">{{ $log->created_at->diffForHumans() }}</span>
                        <span>{{ $log->getActionLabel() }}</span>
                        <span class="text-zinc-400">&middot;</span>
                        <span>{{ $log->vendor->name }}</span>
                        @if($log->error_message)
                            <flux:badge color="red">Error</flux:badge>
                        @endif
                    </div>
                @empty
                    <div class="text-zinc-500 text-center py-4">No recent activity</div>
                @endforelse
            </div>
        </div>
    </div>
</div>
