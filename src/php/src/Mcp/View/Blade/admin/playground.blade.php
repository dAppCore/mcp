<div>
    <div class="mb-8">
        <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">{{ __('mcp::mcp.playground.title') }}</h1>
        <p class="mt-2 text-lg text-zinc-600 dark:text-zinc-400">
            {{ __('mcp::mcp.playground.description') }}
        </p>
    </div>

    {{-- Error Display --}}
    @if($error)
        <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl">
            <div class="flex items-start gap-3">
                <core:icon.x-circle class="w-5 h-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" />
                <p class="text-sm text-red-700 dark:text-red-300">{{ $error }}</p>
            </div>
        </div>
    @endif

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <!-- Request Builder -->
        <div class="space-y-6">
            <!-- API Key Input -->
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">{{ __('mcp::mcp.playground.auth.title') }}</h2>

                <div class="space-y-4">
                    <div>
                        <core:input
                            wire:model="apiKey"
                            type="password"
                            label="{{ __('mcp::mcp.playground.auth.api_key_label') }}"
                            placeholder="{{ __('mcp::mcp.playground.auth.api_key_placeholder') }}"
                            description="{{ __('mcp::mcp.playground.auth.api_key_description') }}"
                        />
                    </div>

                    <div class="flex items-center gap-3">
                        <core:button wire:click="validateKey" size="sm" variant="ghost">
                            {{ __('mcp::mcp.playground.auth.validate') }}
                        </core:button>

                        @if($keyStatus === 'valid')
                            <span class="inline-flex items-center gap-1 text-sm text-green-600 dark:text-green-400">
                                <core:icon.check-circle class="w-4 h-4" />
                                {{ __('mcp::mcp.playground.auth.status.valid') }}
                            </span>
                        @elseif($keyStatus === 'invalid')
                            <span class="inline-flex items-center gap-1 text-sm text-red-600 dark:text-red-400">
                                <core:icon.x-circle class="w-4 h-4" />
                                {{ __('mcp::mcp.playground.auth.status.invalid') }}
                            </span>
                        @elseif($keyStatus === 'expired')
                            <span class="inline-flex items-center gap-1 text-sm text-amber-600 dark:text-amber-400">
                                <core:icon.clock class="w-4 h-4" />
                                {{ __('mcp::mcp.playground.auth.status.expired') }}
                            </span>
                        @elseif($keyStatus === 'empty')
                            <span class="text-sm text-zinc-500 dark:text-zinc-400">
                                {{ __('mcp::mcp.playground.auth.status.empty') }}
                            </span>
                        @endif
                    </div>

                    @if($keyInfo)
                        <div class="p-3 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                            <div class="grid grid-cols-2 gap-2 text-sm">
                                <div>
                                    <span class="text-green-600 dark:text-green-400">{{ __('mcp::mcp.playground.auth.key_info.name') }}:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ $keyInfo['name'] }}</span>
                                </div>
                                <div>
                                    <span class="text-green-600 dark:text-green-400">{{ __('mcp::mcp.playground.auth.key_info.workspace') }}:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ $keyInfo['workspace'] }}</span>
                                </div>
                                <div>
                                    <span class="text-green-600 dark:text-green-400">{{ __('mcp::mcp.playground.auth.key_info.scopes') }}:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ implode(', ', $keyInfo['scopes'] ?? []) }}</span>
                                </div>
                                <div>
                                    <span class="text-green-600 dark:text-green-400">{{ __('mcp::mcp.playground.auth.key_info.last_used') }}:</span>
                                    <span class="text-green-800 dark:text-green-200 ml-1">{{ $keyInfo['last_used'] }}</span>
                                </div>
                            </div>
                        </div>
                    @elseif(!$isAuthenticated && !$apiKey)
                        <div class="p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                            <p class="text-sm text-amber-700 dark:text-amber-300">
                                <a href="{{ route('login') }}" class="underline hover:no-underline">{{ __('mcp::mcp.playground.auth.sign_in_prompt') }}</a>
                                {{ __('mcp::mcp.playground.auth.sign_in_description') }}
                            </p>
                        </div>
                    @endif
                </div>
            </div>

            <!-- Server & Tool Selection -->
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">{{ __('mcp::mcp.playground.tools.title') }}</h2>

                <div class="space-y-4">
                    <core:select wire:model.live="selectedServer" label="{{ __('mcp::mcp.playground.tools.server_label') }}" placeholder="{{ __('mcp::mcp.playground.tools.server_placeholder') }}">
                        @foreach($servers as $server)
                            <core:select.option value="{{ $server['id'] }}">{{ $server['name'] }}</core:select.option>
                        @endforeach
                    </core:select>

                    @if($selectedServer && count($tools) > 0)
                        <core:select wire:model.live="selectedTool" label="{{ __('mcp::mcp.playground.tools.tool_label') }}" placeholder="{{ __('mcp::mcp.playground.tools.tool_placeholder') }}">
                            @foreach($tools as $tool)
                                <core:select.option value="{{ $tool['name'] }}">{{ $tool['name'] }}</core:select.option>
                            @endforeach
                        </core:select>
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
                            <h4 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300">{{ __('mcp::mcp.playground.tools.arguments') }}</h4>

                            @foreach($params as $name => $schema)
                                <div>
                                    @php
                                        $paramRequired = in_array($name, $required) || ($schema['required'] ?? false);
                                        $paramType = is_array($schema['type'] ?? 'string') ? ($schema['type'][0] ?? 'string') : ($schema['type'] ?? 'string');
                                    @endphp

                                    @if(isset($schema['enum']))
                                        <core:select
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="Select..."
                                            description="{{ $schema['description'] ?? '' }}"
                                        >
                                            @foreach($schema['enum'] as $option)
                                                <core:select.option value="{{ $option }}">{{ $option }}</core:select.option>
                                            @endforeach
                                        </core:select>
                                    @elseif($paramType === 'boolean')
                                        <core:select
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="Default"
                                            description="{{ $schema['description'] ?? '' }}"
                                        >
                                            <core:select.option value="true">true</core:select.option>
                                            <core:select.option value="false">false</core:select.option>
                                        </core:select>
                                    @elseif($paramType === 'integer' || $paramType === 'number')
                                        <core:input
                                            type="number"
                                            wire:model="arguments.{{ $name }}"
                                            label="{{ $name }}{{ $paramRequired ? ' *' : '' }}"
                                            placeholder="{{ $schema['default'] ?? '' }}"
                                            description="{{ $schema['description'] ?? '' }}"
                                        />
                                    @else
                                        <core:input
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
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">{{ __('mcp::mcp.playground.tools.no_arguments') }}</p>
                    @endif

                    <div class="mt-6">
                        <core:button
                            wire:click="execute"
                            wire:loading.attr="disabled"
                            variant="primary"
                            class="w-full"
                        >
                            <span wire:loading.remove wire:target="execute">
                                @if($keyStatus === 'valid')
                                    {{ __('mcp::mcp.playground.tools.execute') }}
                                @else
                                    {{ __('mcp::mcp.playground.tools.generate') }}
                                @endif
                            </span>
                            <span wire:loading wire:target="execute">{{ __('mcp::mcp.playground.tools.executing') }}</span>
                        </core:button>
                    </div>
                </div>
            @endif
        </div>

        <!-- Response -->
        <div class="space-y-6">
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4">{{ __('mcp::mcp.playground.response.title') }}</h2>

                @if($response)
                    <div x-data="{ copied: false }">
                        <div class="flex justify-end mb-2">
                            <button
                                x-on:click="navigator.clipboard.writeText($refs.response.textContent); copied = true; setTimeout(() => copied = false, 2000)"
                                class="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                            >
                                <span x-show="!copied">{{ __('mcp::mcp.playground.response.copy') }}</span>
                                <span x-show="copied" x-cloak>{{ __('mcp::mcp.playground.response.copied') }}</span>
                            </button>
                        </div>
                        <pre x-ref="response" class="bg-zinc-900 dark:bg-zinc-950 rounded-lg p-4 overflow-x-auto text-sm text-emerald-400 whitespace-pre-wrap">{{ $response }}</pre>
                    </div>
                @else
                    <div class="text-center py-12 text-zinc-500 dark:text-zinc-400">
                        <core:icon.code-bracket-square class="w-12 h-12 mx-auto mb-4 opacity-50" />
                        <p>{{ __('mcp::mcp.playground.response.empty') }}</p>
                    </div>
                @endif
            </div>

            <!-- Quick Reference -->
            <div class="bg-zinc-50 dark:bg-zinc-900/50 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                <h3 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300 mb-3">{{ __('mcp::mcp.playground.reference.title') }}</h3>
                <div class="space-y-3 text-sm">
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">{{ __('mcp::mcp.playground.reference.endpoint') }}:</span>
                        <code class="ml-2 text-zinc-800 dark:text-zinc-200 break-all">{{ config('app.url') }}/api/v1/mcp/tools/call</code>
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">{{ __('mcp::mcp.playground.reference.method') }}:</span>
                        <code class="ml-2 text-zinc-800 dark:text-zinc-200">POST</code>
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">{{ __('mcp::mcp.playground.reference.auth') }}:</span>
                        @if($keyStatus === 'valid')
                            <code class="ml-2 text-green-600 dark:text-green-400">Bearer {{ Str::limit($apiKey, 20, '...') }}</code>
                        @else
                            <code class="ml-2 text-zinc-800 dark:text-zinc-200">Bearer &lt;your-api-key&gt;</code>
                        @endif
                    </div>
                    <div>
                        <span class="text-zinc-500 dark:text-zinc-400">{{ __('mcp::mcp.playground.reference.content_type') }}:</span>
                        <code class="ml-2 text-zinc-800 dark:text-zinc-200">application/json</code>
                    </div>
                </div>

                @if($isAuthenticated)
                    <div class="mt-4 pt-4 border-t border-zinc-200 dark:border-zinc-700">
                        <core:button href="{{ route('mcp.keys') }}" size="sm" variant="ghost" icon="key">
                            {{ __('mcp::mcp.playground.reference.manage_keys') }}
                        </core:button>
                    </div>
                @endif
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
