<div class="max-w-6xl mx-auto">
    <div class="mb-8">
        <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">Playground</h1>
        <p class="mt-2 text-lg text-zinc-600 dark:text-zinc-400">
            Test MCP tools interactively and execute requests live.
        </p>
    </div>

    {{-- Error Display --}}
    @if($error)
        <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl">
            <div class="flex items-start gap-3">
                <flux:icon.x-circle class="w-5 h-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
                <p class="text-sm text-red-700 dark:text-red-300">{{ $error }}</p>
            </div>
        </div>
    @endif

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <!-- Request Builder -->
        <div class="space-y-6">
            <!-- API Key Input -->
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Authentication</h2>

                <div class="space-y-4">
                    <div>
                        <flux:input
                            wire:model="apiKey"
                            type="password"
                            label="API Key"
                            placeholder="hk_xxxxxxxx_xxxxxxxxxxxx..."
                            description="Paste your API key to execute requests live"
                        />
                    </div>

                    <div class="flex items-center gap-3">
                        <flux:button wire:click="validateKey" size="sm" variant="ghost">
                            Validate Key
                        </flux:button>

                        @if($keyStatus === 'valid')
                            <span class="inline-flex items-center gap-1 text-sm text-green-600 dark:text-green-400">
                                <flux:icon.check-circle class="w-4 h-4" />
                                Valid
                            </span>
                        @elseif($keyStatus === 'invalid')
                            <span class="inline-flex items-center gap-1 text-sm text-red-600 dark:text-red-400">
                                <flux:icon.x-circle class="w-4 h-4" />
                                Invalid key
                            </span>
                        @elseif($keyStatus === 'expired')
                            <span class="inline-flex items-center gap-1 text-sm text-amber-600 dark:text-amber-400">
                                <flux:icon.clock class="w-4 h-4" />
                                Expired
                            </span>
                        @elseif($keyStatus === 'empty')
                            <span class="text-sm text-zinc-500 dark:text-zinc-400">
                                Enter a key to validate
                            </span>
                        @endif
                    </div>

                    @if($keyInfo)
                        <div class="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                            <div class="grid grid-cols-2 gap-2 text-sm">
                                <div>
                                    <span class="text-green-600 dark:text-green-400">Name:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ $keyInfo['name'] }}</span>
                                </div>
                                <div>
                                    <span class="text-green-600 dark:text-green-400">Workspace:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ $keyInfo['workspace'] }}</span>
                                </div>
                                <div>
                                    <span class="text-green-600 dark:text-green-400">Scopes:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ implode(', ', $keyInfo['scopes'] ?? []) }}</span>
                                </div>
                                <div>
                                    <span class="text-green-600 dark:text-green-400">Last used:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ $keyInfo['last_used'] }}</span>
                                </div>
                            </div>
                        </div>
                    @elseif(!$isAuthenticated && !$apiKey)
                        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                            <p class="text-sm text-amber-700 dark:text-amber-300">
                                <a href="{{ route('login') }}" class="underline hover:no-underline">Sign in</a>
                                to create API keys, or paste an existing key above.
                            </p>
                        </div>
                    @endif
                </div>
            </div>

            <!-- Server & Tool Selection -->
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Select Tool</h2>

                <div class="space-y-4">
                    <flux:select wire:model.live="selectedServer" label="Server" placeholder="Choose a server...">
                        @foreach($servers as $server)
                            <flux:select.option value="{{ $server['id'] }}">{{ $server['name'] }}</flux:select.option>
                        @endforeach
                    </flux:select>

                    @if($selectedServer && count($tools) > 0)
                        <flux:select wire:model.live="selectedTool" label="Tool" placeholder="Choose a tool...">
                            @foreach($tools as $tool)
                                <flux:select.option value="{{ $tool['name'] }}">{{ $tool['name'] }}</flux:select.option>
                            @endforeach
                        </flux:select>
                    @endif
                </div>
            </div>

            <!-- Tool Info & Arguments -->
            @if($toolSchema)
                <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                    <div class="mb-4">
                        <h3 class="text-lg font-semibold text-zinc-900 dark:text-white">{{ $toolSchema['name'] }}</h3>
                        <p class="text-sm text-zinc-600 dark:text-zinc-400">{{ $toolSchema['description'] ?? $toolSchema['purpose'] ?? '' }}</p>
                    </div>

                    @php
                        $params = $toolSchema['inputSchema']['properties'] ?? $toolSchema['parameters'] ?? [];
                        $required = $toolSchema['inputSchema']['required'] ?? [];
                    @endphp

                    @if(count($params) > 0)
                        <div class="space-y-4">
                            <h4 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300">Arguments</h4>

                            @foreach($params as $name => $schema)
                                <div>
                                    @php
                                        $paramRequired = in_array($name, $required) || ($schema['required'] ?? false);
                                        $paramType = is_array($schema['type'] ?? 'string') ? ($schema['type'][0] ?? 'string') : ($schema['type'] ?? 'string');
                                    @endphp

                                    @if(isset($schema['enum']))
                                        <flux:select
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="Select..."
                                            description="{{ $schema['description'] ?? '' }}"
                                        >
                                            @foreach($schema['enum'] as $option)
                                                <flux:select.option value="{{ $option }}">{{ $option }}</flux:select.option>
                                            @endforeach
                                        </flux:select>
                                    @elseif($paramType === 'boolean')
                                        <flux:select
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="Default"
                                            description="{{ $schema['description'] ?? '' }}"
                                        >
                                            <flux:select.option value="true">true</flux:select.option>
                                            <flux:select.option value="false">false</flux:select.option>
                                        </flux:select>
                                    @elseif($paramType === 'integer' || $paramType === 'number')
                                        <flux:input
                                            type="number"
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="{{ $schema['default'] ?? '' }}"
                                            description="{{ $schema['description'] ?? '' }}"
                                        />
                                    @else
                                        <flux:input
                                            type="text"
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="{{ $schema['default'] ?? '' }}"
                                            description="{{ $schema['description'] ?? '' }}"
                                        />
                                    @endif
                                </div>
                            @endforeach
                        </div>
                    @else
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">This tool has no arguments.</p>
                    @endif

                    <div class="mt-6">
                        <flux:button
                            wire:click="execute"
                            wire:loading.attr="disabled"
                            variant="primary"
                            class="w-full"
                        >
                            <span wire:loading.remove wire:target="execute">
                                @if($keyStatus === 'valid')
                                    Execute Request
                                @else
                                    Generate Request
                                @endif
                            </span>
                            <span wire:loading wire:target="execute">Executing...</span>
                        </flux:button>
                    </div>
                </div>
            @endif
        </div>

        <!-- Response -->
        <div class="space-y-6">
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">Response</h2>

                @if($response)
                    <div x-data="{ copied: false }">
                        <div class="flex justify-end mb-2">
                            <button
                                x-on:click="navigator.clipboard.writeText($refs.response.textContent); copied = true; setTimeout(() => copied = false, 2000)"
                                class="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                            >
                                <span x-show="!copied">Copy</span>
                                <span x-show="copied" x-cloak>Copied</span>
                            </button>
                        </div>
                        <pre x-ref="response" class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm text-emerald-400 whitespace-pre-wrap">{{ $response }}</pre>
                    </div>
                @else
                    <div class="text-center py-12 text-zinc-500 dark:text-zinc-400">
                        <flux:icon.code-bracket-square class="w-12 h-12 mx-auto mb-4 opacity-50" />
                        <p>Select a server and tool to get started.</p>
                    </div>
                @endif
            </div>

            <!-- Quick Reference -->
            <div class="bg-zinc-50 dark:bg-zinc-900/50 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-3">API Reference</h3>
                <div class="space-y-3 text-sm">
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">Endpoint:</span>
                        <code class="ml-2 text-zinc-800 dark:text-zinc-200 break-all">{{ request()->getSchemeAndHttpHost() }}/tools/call</code>
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">Method:</span>
                        <code class="ml-2 text-zinc-800 dark:text-zinc-200">POST</code>
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">Auth:</span>
                        @if($keyStatus === 'valid')
                            <code class="ml-2 text-green-600 dark:text-green-400">Bearer {{ Str::limit($apiKey, 20, '...') }}</code>
                        @else
                            <code class="ml-2 text-zinc-800 dark:text-zinc-200">Bearer &lt;your-api-key&gt;</code>
                        @endif
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">Content-Type:</span>
                        <code class="ml-2 text-zinc-800 dark:text-zinc-200">application/json</code>
                    </div>
                </div>

            </div>
        </div>
    </div>
</div>

@script
<script>
    // Suppress Livewire request errors to prevent modal popups
    // Errors are handled gracefully in the component
    document.addEventListener('admin:request-error', (event) => {
        // Prevent the default Livewire error modal
        event.preventDefault();
        console.warn('MCP Playground: Request failed', event.detail);
    });
</script>
@endscript
