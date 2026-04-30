<div>
    <!-- Flash Messages -->
    @if(session('message'))
        <div class="mb-6 p-4 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg">
            <p class="text-emerald-800 dark:text-emerald-200">{{ session('message') }}</p>
        </div>
    @endif

    <!-- Header -->
    <div class="mb-8 flex items-center justify-between">
        <div>
            <h1 class="text-3xl font-bold text-zinc-900 dark:text-white mb-2">
                {{ __('mcp::mcp.keys.title') }}
            </h1>
            <p class="text-zinc-600 dark:text-zinc-400">
                {{ __('mcp::mcp.keys.description') }}
            </p>
        </div>
        <core:button icon="plus" variant="primary" wire:click="openCreateModal">
            {{ __('mcp::mcp.keys.actions.create') }}
        </core:button>
    </div>

    <!-- Keys List -->
    <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden">
        @if($keys->isEmpty())
            <div class="p-12 text-center">
                <div class="p-4 bg-cyan-50 dark:bg-cyan-900/20 rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center">
                    <core:icon.key class="w-8 h-8 text-cyan-500" />
                </div>
                <h3 class="text-lg font-semibold text-zinc-900 dark:text-white mb-2">{{ __('mcp::mcp.keys.empty.title') }}</h3>
                <p class="text-zinc-600 dark:text-zinc-400 mb-6 max-w-md mx-auto">
                    {{ __('mcp::mcp.keys.empty.description') }}
                </p>
                <core:button icon="plus" variant="primary" wire:click="openCreateModal">
                    {{ __('mcp::mcp.keys.actions.create_first') }}
                </core:button>
            </div>
        @else
            <table class="w-full">
                <thead class="bg-zinc-50 dark:bg-zinc-900/50">
                    <tr>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            {{ __('mcp::mcp.keys.table.name') }}
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            {{ __('mcp::mcp.keys.table.key') }}
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            {{ __('mcp::mcp.keys.table.scopes') }}
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            {{ __('mcp::mcp.keys.table.last_used') }}
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            {{ __('mcp::mcp.keys.table.expires') }}
                        </th>
                        <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            {{ __('mcp::mcp.keys.table.actions') }}
                        </th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                    @foreach($keys as $key)
                        <tr>
                            <td class="px-6 py-4 whitespace-nowrap">
                                <span class="text-zinc-900 dark:text-white font-medium">{{ $key->name }}</span>
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap">
                                <code class="text-sm bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded font-mono">
                                    {{ $key->prefix }}_****
                                </code>
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap">
                                <div class="flex gap-1">
                                    @foreach($key->scopes ?? [] as $scope)
                                        <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium
                                            {{ $scope === 'read' ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300' : '' }}
                                            {{ $scope === 'write' ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300' : '' }}
                                            {{ $scope === 'delete' ? 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300' : '' }}
                                        ">
                                            {{ $scope }}
                                        </span>
                                    @endforeach
                                </div>
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-sm text-zinc-600 dark:text-zinc-400">
                                {{ $key->last_used_at?->diffForHumans() ?? __('mcp::mcp.keys.status.never') }}
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-sm">
                                @if($key->expires_at)
                                    @if($key->expires_at->isPast())
                                        <span class="text-red-600 dark:text-red-400">{{ __('mcp::mcp.keys.status.expired') }}</span>
                                    @else
                                        <span class="text-zinc-600 dark:text-zinc-400">{{ $key->expires_at->diffForHumans() }}</span>
                                    @endif
                                @else
                                    <span class="text-zinc-500 dark:text-zinc-500">{{ __('mcp::mcp.keys.status.never') }}</span>
                                @endif
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
                                <core:button
                                    variant="ghost"
                                    size="sm"
                                    icon="trash"
                                    class="text-red-600 hover:text-red-700"
                                    wire:click="revokeKey({{ $key->id }})"
                                    wire:confirm="{{ __('mcp::mcp.keys.confirm_revoke') }}"
                                >
                                    {{ __('mcp::mcp.keys.actions.revoke') }}
                                </core:button>
                            </td>
                        </tr>
                    @endforeach
                </tbody>
            </table>
        @endif
    </div>

    <!-- HTTP Usage Instructions -->
    <div class="mt-8 grid md:grid-cols-2 gap-6">
        <!-- Authentication -->
        <div class="p-6 bg-zinc-50 dark:bg-zinc-800/50 rounded-xl border border-zinc-200 dark:border-zinc-700">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4 flex items-center gap-2">
                <core:icon.lock-closed class="w-5 h-5 text-cyan-500" />
                {{ __('mcp::mcp.keys.auth.title') }}
            </h2>
            <p class="text-zinc-600 dark:text-zinc-400 mb-4 text-sm">
                {{ __('mcp::mcp.keys.auth.description') }}
            </p>
            <div class="space-y-3">
                <div>
                    <p class="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1">{{ __('mcp::mcp.keys.auth.header_recommended') }}</p>
                    <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-2 overflow-x-auto text-xs"><code class="text-zinc-800 dark:text-zinc-200">Authorization: Bearer hk_abc123_****</code></pre>
                </div>
                <div>
                    <p class="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1">{{ __('mcp::mcp.keys.auth.header_api_key') }}</p>
                    <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-2 overflow-x-auto text-xs"><code class="text-zinc-800 dark:text-zinc-200">X-API-Key: hk_abc123_****</code></pre>
                </div>
            </div>
        </div>

        <!-- Example Request -->
        <div class="p-6 bg-zinc-50 dark:bg-zinc-800/50 rounded-xl border border-zinc-200 dark:border-zinc-700">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4 flex items-center gap-2">
                <core:icon.code-bracket class="w-5 h-5 text-cyan-500" />
                {{ __('mcp::mcp.keys.example.title') }}
            </h2>
            <p class="text-zinc-600 dark:text-zinc-400 mb-4 text-sm">
                {{ __('mcp::mcp.keys.example.description') }}
            </p>
            <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-3 overflow-x-auto text-xs"><code class="text-emerald-400">curl -X POST https://mcp.host.uk.com/api/v1/tools/call \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "server": "commerce",
    "tool": "product_list",
    "arguments": {}
  }'</code></pre>
        </div>
    </div>

    <!-- Create Key Modal -->
    <core:modal wire:model="showCreateModal" class="max-w-md">
        <div class="p-6">
            <h3 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">{{ __('mcp::mcp.keys.create_modal.title') }}</h3>

            <div class="space-y-4">
                <!-- Key Name -->
                <div>
                    <core:label for="keyName">{{ __('mcp::mcp.keys.create_modal.name_label') }}</core:label>
                    <core:input
                        id="keyName"
                        wire:model="newKeyName"
                        placeholder="{{ __('mcp::mcp.keys.create_modal.name_placeholder') }}"
                    />
                    @error('newKeyName')
                        <p class="mt-1 text-sm text-red-600">{{ $message }}</p>
                    @enderror
                </div>

                <!-- Scopes -->
                <div>
                    <core:label>{{ __('mcp::mcp.keys.create_modal.permissions_label') }}</core:label>
                    <div class="mt-2 space-y-2">
                        <label class="flex items-center gap-2">
                            <input
                                type="checkbox"
                                wire:click="toggleScope('read')"
                                {{ in_array('read', $newKeyScopes) ? 'checked' : '' }}
                                class="rounded border-zinc-300 text-cyan-600 focus:ring-cyan-500"
                            >
                            <span class="text-sm text-zinc-700 dark:text-zinc-300">{{ __('mcp::mcp.keys.create_modal.permission_read') }}</span>
                        </label>
                        <label class="flex items-center gap-2">
                            <input
                                type="checkbox"
                                wire:click="toggleScope('write')"
                                {{ in_array('write', $newKeyScopes) ? 'checked' : '' }}
                                class="rounded border-zinc-300 text-cyan-600 focus:ring-cyan-500"
                            >
                            <span class="text-sm text-zinc-700 dark:text-zinc-300">{{ __('mcp::mcp.keys.create_modal.permission_write') }}</span>
                        </label>
                        <label class="flex items-center gap-2">
                            <input
                                type="checkbox"
                                wire:click="toggleScope('delete')"
                                {{ in_array('delete', $newKeyScopes) ? 'checked' : '' }}
                                class="rounded border-zinc-300 text-zinc-600 focus:ring-cyan-500"
                            >
                            <span class="text-sm text-zinc-700 dark:text-zinc-300">{{ __('mcp::mcp.keys.create_modal.permission_delete') }}</span>
                        </label>
                    </div>
                </div>

                <!-- Expiry -->
                <div>
                    <core:label for="keyExpiry">{{ __('mcp::mcp.keys.create_modal.expiry_label') }}</core:label>
                    <core:select id="keyExpiry" wire:model="newKeyExpiry">
                        <option value="never">{{ __('mcp::mcp.keys.create_modal.expiry_never') }}</option>
                        <option value="30days">{{ __('mcp::mcp.keys.create_modal.expiry_30') }}</option>
                        <option value="90days">{{ __('mcp::mcp.keys.create_modal.expiry_90') }}</option>
                        <option value="1year">{{ __('mcp::mcp.keys.create_modal.expiry_1year') }}</option>
                    </core:select>
                </div>
            </div>

            <div class="mt-6 flex justify-end gap-3">
                <core:button variant="ghost" wire:click="closeCreateModal">{{ __('mcp::mcp.keys.create_modal.cancel') }}</core:button>
                <core:button variant="primary" wire:click="createKey">{{ __('mcp::mcp.keys.create_modal.create') }}</core:button>
            </div>
        </div>
    </core:modal>

    <!-- New Key Display Modal -->
    <core:modal wire:model="showNewKeyModal" class="max-w-lg">
        <div class="p-6">
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-full">
                    <core:icon.check class="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                </div>
                <h3 class="text-lg font-semibold text-zinc-900 dark:text-white">{{ __('mcp::mcp.keys.new_key_modal.title') }}</h3>
            </div>

            <div class="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
                <p class="text-sm text-amber-800 dark:text-amber-200">
                    <strong>{{ __('mcp::mcp.keys.new_key_modal.warning') }}</strong> {{ __('mcp::mcp.keys.new_key_modal.warning_detail') }}
                </p>
            </div>

            <div class="relative" x-data="{ copied: false }">
                <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-4 overflow-x-auto text-sm font-mono break-all pr-12"><code class="text-zinc-800 dark:text-zinc-200">{{ $newPlainKey }}</code></pre>
                <button
                    type="button"
                    x-on:click="navigator.clipboard.writeText('{{ $newPlainKey }}'); copied = true; setTimeout(() => copied = false, 2000)"
                    class="absolute top-3 right-3 p-2 rounded-lg bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-700 transition-colors"
                >
                    <core:icon.clipboard x-show="!copied" class="w-5 h-5 text-zinc-500" />
                    <core:icon.check x-show="copied" x-cloak class="w-5 h-5 text-emerald-500" />
                </button>
            </div>

            <div class="mt-6 flex justify-end">
                <core:button variant="primary" wire:click="closeNewKeyModal">{{ __('mcp::mcp.keys.new_key_modal.done') }}</core:button>
            </div>
        </div>
    </core:modal>
</div>
