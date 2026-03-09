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
                API Keys
            </h1>
            <p class="text-zinc-600 dark:text-zinc-400">
                Create API keys to authenticate HTTP requests to MCP servers.
            </p>
        </div>
        <flux:button icon="plus" variant="primary" wire:click="openCreateModal">
            Create Key
        </flux:button>
    </div>

    <!-- Keys List -->
    <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden">
        @if($keys->isEmpty())
            <div class="p-12 text-center">
                <div class="p-4 bg-cyan-50 dark:bg-cyan-900/20 rounded-full w-16 h-16 mx-auto mb-4 flex items-center justify-center">
                    <flux:icon.key class="w-8 h-8 text-cyan-500" />
                </div>
                <h3 class="text-lg font-semibold text-zinc-900 dark:text-white mb-2">No API Keys Yet</h3>
                <p class="text-zinc-600 dark:text-zinc-400 mb-6 max-w-md mx-auto">
                    Create an API key to start making authenticated requests to MCP servers over HTTP.
                </p>
                <flux:button icon="plus" variant="primary" wire:click="openCreateModal">
                    Create Your First Key
                </flux:button>
            </div>
        @else
            <table class="w-full">
                <thead class="bg-zinc-50 dark:bg-zinc-900/50">
                    <tr>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            Name
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            Key
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            Scopes
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            Last Used
                        </th>
                        <th class="px-6 py-3 text-left text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            Expires
                        </th>
                        <th class="px-6 py-3 text-right text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">
                            Actions
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
                                {{ $key->last_used_at?->diffForHumans() ?? 'Never' }}
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-sm">
                                @if($key->expires_at)
                                    @if($key->expires_at->isPast())
                                        <span class="text-red-600 dark:text-red-400">Expired</span>
                                    @else
                                        <span class="text-zinc-600 dark:text-zinc-400">{{ $key->expires_at->diffForHumans() }}</span>
                                    @endif
                                @else
                                    <span class="text-zinc-500 dark:text-zinc-500">Never</span>
                                @endif
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
                                <flux:button
                                    variant="ghost"
                                    size="sm"
                                    icon="trash"
                                    class="text-red-600 hover:text-red-700"
                                    wire:click="revokeKey({{ $key->id }})"
                                    wire:confirm="Are you sure you want to revoke this API key? This cannot be undone."
                                >
                                    Revoke
                                </flux:button>
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
                <flux:icon.lock-closed class="w-5 h-5 text-cyan-500" />
                Authentication
            </h2>
            <p class="text-zinc-600 dark:text-zinc-400 mb-4 text-sm">
                Include your API key in HTTP requests using one of these methods:
            </p>
            <div class="space-y-3">
                <div>
                    <p class="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1">Authorization Header (recommended)</p>
                    <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-2 overflow-x-auto text-xs"><code class="text-zinc-800 dark:text-zinc-200">Authorization: Bearer hk_abc123_****</code></pre>
                </div>
                <div>
                    <p class="text-xs font-medium text-zinc-700 dark:text-zinc-300 mb-1">X-API-Key Header</p>
                    <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-2 overflow-x-auto text-xs"><code class="text-zinc-800 dark:text-zinc-200">X-API-Key: hk_abc123_****</code></pre>
                </div>
            </div>
        </div>

        <!-- Example Request -->
        <div class="p-6 bg-zinc-50 dark:bg-zinc-800/50 rounded-xl border border-zinc-200 dark:border-zinc-700">
            <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4 flex items-center gap-2">
                <flux:icon.code-bracket class="w-5 h-5 text-cyan-500" />
                Example Request
            </h2>
            <p class="text-zinc-600 dark:text-zinc-400 mb-4 text-sm">
                Call an MCP tool via HTTP POST:
            </p>
            @php $mcpUrl = request()->getSchemeAndHttpHost(); @endphp
            <pre class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-3 overflow-x-auto text-xs"><code class="text-emerald-400">curl -X POST {{ $mcpUrl }}/tools/call \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "server": "openbrain",
    "tool": "brain_recall",
    "arguments": { "query": "recent decisions" }
  }'</code></pre>
        </div>
    </div>

    <!-- Create Key Modal -->
    <flux:modal wire:model="showCreateModal" class="max-w-md">
        <div class="p-6">
            <h3 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Create API Key</h3>

            <div class="space-y-4">
                <!-- Key Name -->
                <div>
                    <flux:label for="keyName">Key Name</flux:label>
                    <flux:input
                        id="keyName"
                        wire:model="newKeyName"
                        placeholder="e.g., Production Server, Claude Agent"
                    />
                    @error('newKeyName')
                        <p class="mt-1 text-sm text-red-600">{{ $message }}</p>
                    @enderror
                </div>

                <!-- Scopes -->
                <div>
                    <flux:label>Permissions</flux:label>
                    <div class="mt-2 space-y-2">
                        <label class="flex items-center gap-2">
                            <input
                                type="checkbox"
                                wire:click="toggleScope('read')"
                                {{ in_array('read', $newKeyScopes) ? 'checked' : '' }}
                                class="rounded border-zinc-300 text-cyan-600 focus:ring-cyan-500"
                            >
                            <span class="text-sm text-zinc-700 dark:text-zinc-300">Read — Query tools and resources</span>
                        </label>
                        <label class="flex items-center gap-2">
                            <input
                                type="checkbox"
                                wire:click="toggleScope('write')"
                                {{ in_array('write', $newKeyScopes) ? 'checked' : '' }}
                                class="rounded border-zinc-300 text-cyan-600 focus:ring-cyan-500"
                            >
                            <span class="text-sm text-zinc-700 dark:text-zinc-300">Write — Create and update data</span>
                        </label>
                        <label class="flex items-center gap-2">
                            <input
                                type="checkbox"
                                wire:click="toggleScope('delete')"
                                {{ in_array('delete', $newKeyScopes) ? 'checked' : '' }}
                                class="rounded border-zinc-300 text-zinc-600 focus:ring-cyan-500"
                            >
                            <span class="text-sm text-zinc-700 dark:text-zinc-300">Delete — Remove data</span>
                        </label>
                    </div>
                </div>

                <!-- Expiry -->
                <div>
                    <flux:label for="keyExpiry">Expiration</flux:label>
                    <flux:select id="keyExpiry" wire:model="newKeyExpiry">
                        <option value="never">Never expires</option>
                        <option value="30days">30 days</option>
                        <option value="90days">90 days</option>
                        <option value="1year">1 year</option>
                    </flux:select>
                </div>
            </div>

            <div class="mt-6 flex justify-end gap-3">
                <flux:button variant="ghost" wire:click="closeCreateModal">Cancel</flux:button>
                <flux:button variant="primary" wire:click="createKey">Create Key</flux:button>
            </div>
        </div>
    </flux:modal>

    <!-- New Key Display Modal -->
    <flux:modal wire:model="showNewKeyModal" class="max-w-lg">
        <div class="p-6">
            <div class="flex items-center gap-3 mb-4">
                <div class="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-full">
                    <flux:icon.check class="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                </div>
                <h3 class="text-lg font-semibold text-zinc-900 dark:text-white">API Key Created</h3>
            </div>

            <div class="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg mb-4">
                <p class="text-sm text-amber-800 dark:text-amber-200">
                    <strong>Copy this key now.</strong> You won't be able to see it again.
                </p>
            </div>

            <div class="relative" x-data="{ copied: false }">
                <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-4 overflow-x-auto text-sm font-mono break-all pr-12"><code class="text-zinc-800 dark:text-zinc-200">{{ $newPlainKey }}</code></pre>
                <button
                    type="button"
                    x-on:click="navigator.clipboard.writeText('{{ $newPlainKey }}'); copied = true; setTimeout(() => copied = false, 2000)"
                    class="absolute top-3 right-3 p-2 rounded-lg bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-700 transition-colors"
                >
                    <flux:icon.clipboard x-show="!copied" class="w-5 h-5 text-zinc-500" />
                    <flux:icon.check x-show="copied" x-cloak class="w-5 h-5 text-emerald-500" />
                </button>
            </div>

            <div class="mt-6 flex justify-end">
                <flux:button variant="primary" wire:click="closeNewKeyModal">Done</flux:button>
            </div>
        </div>
    </flux:modal>
</div>
