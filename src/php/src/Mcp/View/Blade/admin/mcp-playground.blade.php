<div class="min-h-screen" x-data="{ showHistory: false }">
    {{-- Header --}}
    <div class="mb-8">
        <div class="flex items-center justify-between">
            <div>
                <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">MCP Playground</h1>
                <p class="mt-2 text-zinc-600 dark:text-zinc-400">
                    Interactive tool testing with documentation and examples
                </p>
            </div>
            <div class="flex items-center gap-3">
                <button
                    x-on:click="showHistory = !showHistory"
                    class="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-colors"
                    :class="showHistory ? 'bg-cyan-50 dark:bg-cyan-900/20 border-cyan-200 dark:border-cyan-800 text-cyan-700 dark:text-cyan-300' : 'bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-700'"
                >
                    <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    History
                    @if(count($conversationHistory) > 0)
                        <span class="inline-flex items-center justify-center w-5 h-5 text-xs font-medium rounded-full bg-cyan-100 dark:bg-cyan-900 text-cyan-700 dark:text-cyan-300">
                            {{ count($conversationHistory) }}
                        </span>
                    @endif
                </button>
            </div>
        </div>
    </div>

    {{-- Error Display --}}
    @if($error)
        <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-xl">
            <div class="flex items-start gap-3">
                <svg class="w-5 h-5 text-red-600 dark:text-red-400 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
                </svg>
                <p class="text-sm text-red-700 dark:text-red-300">{{ $error }}</p>
            </div>
        </div>
    @endif

    <div class="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {{-- Left Sidebar: Tool Browser --}}
        <div class="lg:col-span-3">
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden sticky top-6">
                {{-- Server Selection --}}
                <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                    <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">Server</label>
                    <select
                        wire:model.live="selectedServer"
                        class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm focus:border-cyan-500 focus:ring-cyan-500"
                    >
                        <option value="">Select a server...</option>
                        @foreach($servers as $server)
                            <option value="{{ $server['id'] }}">{{ $server['name'] }} ({{ $server['tool_count'] }})</option>
                        @endforeach
                    </select>
                </div>

                @if($selectedServer)
                    {{-- Search --}}
                    <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                        <div class="relative">
                            <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z" />
                            </svg>
                            <input
                                type="text"
                                wire:model.live.debounce.300ms="searchQuery"
                                placeholder="Search tools..."
                                class="w-full pl-10 pr-4 py-2 text-sm rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 focus:border-cyan-500 focus:ring-cyan-500"
                            >
                        </div>
                    </div>

                    {{-- Category Filter --}}
                    @if($categories->isNotEmpty())
                        <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                            <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider mb-2">Category</label>
                            <div class="flex flex-wrap gap-1">
                                <button
                                    wire:click="$set('selectedCategory', '')"
                                    class="px-2 py-1 text-xs font-medium rounded-md transition-colors {{ empty($selectedCategory) ? 'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-700 dark:text-cyan-300' : 'bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-600' }}"
                                >
                                    All
                                </button>
                                @foreach($categories as $category)
                                    <button
                                        wire:click="$set('selectedCategory', '{{ $category }}')"
                                        class="px-2 py-1 text-xs font-medium rounded-md transition-colors {{ $selectedCategory === $category ? 'bg-cyan-100 dark:bg-cyan-900/30 text-cyan-700 dark:text-cyan-300' : 'bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-400 hover:bg-zinc-200 dark:hover:bg-zinc-600' }}"
                                    >
                                        {{ $category }}
                                    </button>
                                @endforeach
                            </div>
                        </div>
                    @endif

                    {{-- Tools List --}}
                    <div class="max-h-[400px] overflow-y-auto">
                        @forelse($toolsByCategory as $category => $categoryTools)
                            <div class="px-4 py-2 bg-zinc-50 dark:bg-zinc-900/50 border-b border-zinc-200 dark:border-zinc-700">
                                <h4 class="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wider">{{ $category }}</h4>
                            </div>
                            @foreach($categoryTools as $tool)
                                <button
                                    wire:click="selectTool('{{ $tool['name'] }}')"
                                    class="w-full text-left px-4 py-3 border-b border-zinc-100 dark:border-zinc-700/50 transition-colors {{ $selectedTool === $tool['name'] ? 'bg-cyan-50 dark:bg-cyan-900/20' : 'hover:bg-zinc-50 dark:hover:bg-zinc-700/50' }}"
                                >
                                    <div class="flex items-center justify-between">
                                        <span class="text-sm font-medium text-zinc-900 dark:text-white">{{ $tool['name'] }}</span>
                                        @if($selectedTool === $tool['name'])
                                            <svg class="w-4 h-4 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                                                <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
                                            </svg>
                                        @endif
                                    </div>
                                    @if(!empty($tool['description']))
                                        <p class="text-xs text-zinc-500 dark:text-zinc-400 mt-1 line-clamp-2">{{ Str::limit($tool['description'], 80) }}</p>
                                    @endif
                                </button>
                            @endforeach
                        @empty
                            <div class="p-8 text-center">
                                <p class="text-sm text-zinc-500 dark:text-zinc-400">No tools found</p>
                            </div>
                        @endforelse
                    </div>
                @else
                    <div class="p-8 text-center">
                        <svg class="w-12 h-12 mx-auto mb-4 text-zinc-300 dark:text-zinc-600" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M5.25 14.25h13.5m-13.5 0a3 3 0 01-3-3m3 3a3 3 0 100 6h13.5a3 3 0 100-6m-16.5-3a3 3 0 013-3h13.5a3 3 0 013 3m-19.5 0a4.5 4.5 0 01.9-2.7L5.737 5.1a3.375 3.375 0 012.7-1.35h7.126c1.062 0 2.062.5 2.7 1.35l2.587 3.45a4.5 4.5 0 01.9 2.7m0 0a3 3 0 01-3 3m0 3h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008zm-3 6h.008v.008h-.008v-.008zm0-6h.008v.008h-.008v-.008z" />
                        </svg>
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">Select a server to browse tools</p>
                    </div>
                @endif
            </div>
        </div>

        {{-- Center: Tool Details & Input Form --}}
        <div class="lg:col-span-5">
            {{-- API Key Authentication --}}
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6 mb-6">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white mb-4 flex items-center gap-2">
                    <svg class="w-5 h-5 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z" />
                    </svg>
                    Authentication
                </h2>
                <div class="space-y-4">
                    <div>
                        <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">API Key</label>
                        <input
                            type="password"
                            wire:model="apiKey"
                            placeholder="hk_xxxxxxxx_xxxxxxxxxxxx..."
                            class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm focus:border-cyan-500 focus:ring-cyan-500"
                        >
                        <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">Paste your API key to execute requests live</p>
                    </div>
                    <div class="flex items-center gap-3">
                        <button
                            wire:click="validateKey"
                            class="px-3 py-1.5 text-sm font-medium text-zinc-700 dark:text-zinc-300 bg-zinc-100 dark:bg-zinc-700 hover:bg-zinc-200 dark:hover:bg-zinc-600 rounded-lg transition-colors"
                        >
                            Validate Key
                        </button>
                        @if($keyStatus === 'valid')
                            <span class="inline-flex items-center gap-1 text-sm text-emerald-600 dark:text-emerald-400">
                                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                Valid
                            </span>
                        @elseif($keyStatus === 'invalid')
                            <span class="inline-flex items-center gap-1 text-sm text-red-600 dark:text-red-400">
                                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M9.75 9.75l4.5 4.5m0-4.5l-4.5 4.5M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                Invalid key
                            </span>
                        @elseif($keyStatus === 'expired')
                            <span class="inline-flex items-center gap-1 text-sm text-amber-600 dark:text-amber-400">
                                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                                </svg>
                                Expired
                            </span>
                        @endif
                    </div>
                    @if($keyInfo)
                        <div class="p-3 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg">
                            <div class="grid grid-cols-2 gap-2 text-sm">
                                <div>
                                    <span class="text-emerald-600 dark:text-emerald-400">Name:</span>
                                    <span class="text-emerald-800 dark:text-emerald-200 ml-1">{{ $keyInfo['name'] }}</span>
                                </div>
                                <div>
                                    <span class="text-emerald-600 dark:text-emerald-400">Workspace:</span>
                                    <span class="text-emerald-800 dark:text-emerald-200 ml-1">{{ $keyInfo['workspace'] }}</span>
                                </div>
                            </div>
                        </div>
                    @endif
                </div>
            </div>

            {{-- Tool Form --}}
            @if($currentTool)
                <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
                    <div class="mb-6">
                        <div class="flex items-start justify-between">
                            <div>
                                <h3 class="text-lg font-semibold text-zinc-900 dark:text-white">{{ $currentTool['name'] }}</h3>
                                <p class="text-sm text-zinc-600 dark:text-zinc-400 mt-1">{{ $currentTool['description'] }}</p>
                            </div>
                            <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-zinc-100 dark:bg-zinc-700 text-zinc-800 dark:text-zinc-200">
                                {{ $currentTool['category'] }}
                            </span>
                        </div>
                    </div>

                    @php
                        $properties = $currentTool['inputSchema']['properties'] ?? [];
                        $required = $currentTool['inputSchema']['required'] ?? [];
                    @endphp

                    @if(count($properties) > 0)
                        <div class="space-y-4">
                            <div class="flex items-center justify-between">
                                <h4 class="text-sm font-semibold text-zinc-700 dark:text-zinc-300">Parameters</h4>
                                <button
                                    wire:click="loadExampleInputs"
                                    class="text-xs text-cyan-600 dark:text-cyan-400 hover:text-cyan-700 dark:hover:text-cyan-300"
                                >
                                    Load examples
                                </button>
                            </div>

                            @foreach($properties as $name => $schema)
                                @php
                                    $isRequired = in_array($name, $required) || ($schema['required'] ?? false);
                                    $type = is_array($schema['type'] ?? 'string') ? ($schema['type'][0] ?? 'string') : ($schema['type'] ?? 'string');
                                    $description = $schema['description'] ?? '';
                                @endphp

                                <div>
                                    <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-1">
                                        {{ $name }}
                                        @if($isRequired)
                                            <span class="text-red-500">*</span>
                                        @endif
                                    </label>

                                    @if(isset($schema['enum']))
                                        <select
                                            wire:model="toolInput.{{ $name }}"
                                            class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm focus:border-cyan-500 focus:ring-cyan-500"
                                        >
                                            <option value="">Select...</option>
                                            @foreach($schema['enum'] as $option)
                                                <option value="{{ $option }}">{{ $option }}</option>
                                            @endforeach
                                        </select>
                                    @elseif($type === 'boolean')
                                        <select
                                            wire:model="toolInput.{{ $name }}"
                                            class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm focus:border-cyan-500 focus:ring-cyan-500"
                                        >
                                            <option value="">Default</option>
                                            <option value="true">true</option>
                                            <option value="false">false</option>
                                        </select>
                                    @elseif($type === 'integer' || $type === 'number')
                                        <input
                                            type="number"
                                            wire:model="toolInput.{{ $name }}"
                                            placeholder="{{ $schema['default'] ?? '' }}"
                                            class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm focus:border-cyan-500 focus:ring-cyan-500"
                                            @if(isset($schema['minimum'])) min="{{ $schema['minimum'] }}" @endif
                                            @if(isset($schema['maximum'])) max="{{ $schema['maximum'] }}" @endif
                                        >
                                    @elseif($type === 'array' || $type === 'object')
                                        <textarea
                                            wire:model="toolInput.{{ $name }}"
                                            rows="3"
                                            placeholder="Enter JSON..."
                                            class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm font-mono focus:border-cyan-500 focus:ring-cyan-500"
                                        ></textarea>
                                    @else
                                        <input
                                            type="text"
                                            wire:model="toolInput.{{ $name }}"
                                            placeholder="{{ $schema['default'] ?? '' }}"
                                            class="w-full rounded-lg border-zinc-300 dark:border-zinc-600 dark:bg-zinc-700 text-sm focus:border-cyan-500 focus:ring-cyan-500"
                                        >
                                    @endif

                                    @if($description)
                                        <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">{{ $description }}</p>
                                    @endif
                                </div>
                            @endforeach
                        </div>
                    @else
                        <p class="text-sm text-zinc-500 dark:text-zinc-400">This tool has no parameters.</p>
                    @endif

                    <div class="mt-6">
                        <button
                            wire:click="execute"
                            wire:loading.attr="disabled"
                            class="w-full inline-flex justify-center items-center px-4 py-2.5 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-cyan-600 hover:bg-cyan-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            <span wire:loading.remove wire:target="execute">
                                @if($keyStatus === 'valid')
                                    Execute Request
                                @else
                                    Generate Request Preview
                                @endif
                            </span>
                            <span wire:loading wire:target="execute" class="flex items-center">
                                <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                                Executing...
                            </span>
                        </button>
                    </div>
                </div>
            @else
                <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-12 text-center">
                    <svg class="w-16 h-16 mx-auto mb-4 text-zinc-300 dark:text-zinc-600" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17L17.25 21A2.652 2.652 0 0021 17.25l-5.877-5.877M11.42 15.17l2.496-3.03c.317-.384.74-.626 1.208-.766M11.42 15.17l-4.655 5.653a2.548 2.548 0 11-3.586-3.586l6.837-5.63m5.108-.233c.55-.164 1.163-.188 1.743-.14a4.5 4.5 0 004.486-6.336l-3.276 3.277a3.004 3.004 0 01-2.25-2.25l3.276-3.276a4.5 4.5 0 00-6.336 4.486c.091 1.076-.071 2.264-.904 2.95l-.102.085m-1.745 1.437L5.909 7.5H4.5L2.25 3.75l1.5-1.5L7.5 4.5v1.409l4.26 4.26m-1.745 1.437l1.745-1.437m6.615 8.206L15.75 15.75M4.867 19.125h.008v.008h-.008v-.008z" />
                    </svg>
                    <h3 class="text-lg font-medium text-zinc-900 dark:text-white mb-2">Select a tool</h3>
                    <p class="text-sm text-zinc-500 dark:text-zinc-400">
                        Choose a tool from the sidebar to view its documentation and test it
                    </p>
                </div>
            @endif
        </div>

        {{-- Right: Response Viewer --}}
        <div class="lg:col-span-4">
            <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden sticky top-6">
                <div class="p-4 border-b border-zinc-200 dark:border-zinc-700 flex items-center justify-between">
                    <h2 class="text-lg font-semibold text-zinc-900 dark:text-white">Response</h2>
                    @if($executionTime > 0)
                        <span class="text-sm text-zinc-500 dark:text-zinc-400">{{ $executionTime }}ms</span>
                    @endif
                </div>

                <div class="p-4" x-data="{ copied: false }">
                    @if($lastResponse)
                        <div class="flex justify-end mb-2">
                            <button
                                x-on:click="navigator.clipboard.writeText($refs.response.textContent); copied = true; setTimeout(() => copied = false, 2000)"
                                class="inline-flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 transition-colors"
                            >
                                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                                    <path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0013.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 01-.75.75H9a.75.75 0 01-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 01-2.25 2.25H6.75A2.25 2.25 0 014.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 011.927-.184" />
                                </svg>
                                <span x-show="!copied">Copy</span>
                                <span x-show="copied" x-cloak>Copied!</span>
                            </button>
                        </div>

                        @if(isset($lastResponse['error']))
                            <div class="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                                <p class="text-sm text-red-700 dark:text-red-300">{{ $lastResponse['error'] }}</p>
                            </div>
                        @endif

                        <div class="bg-zinc-900 dark:bg-zinc-950 rounded-lg overflow-hidden">
                            <pre x-ref="response" class="p-4 text-sm text-emerald-400 overflow-x-auto max-h-[500px] whitespace-pre-wrap">{{ json_encode($lastResponse, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) }}</pre>
                        </div>

                        @if(isset($lastResponse['executed']) && !$lastResponse['executed'])
                            <div class="mt-4 p-3 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                                <p class="text-sm text-amber-700 dark:text-amber-300">
                                    This is a preview. Add a valid API key to execute requests live.
                                </p>
                            </div>
                        @endif
                    @else
                        <div class="py-12 text-center">
                            <svg class="w-12 h-12 mx-auto mb-4 text-zinc-300 dark:text-zinc-600" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5" />
                            </svg>
                            <p class="text-sm text-zinc-500 dark:text-zinc-400">Response will appear here</p>
                        </div>
                    @endif
                </div>

                {{-- API Reference --}}
                <div class="p-4 border-t border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-900/50">
                    <h4 class="text-xs font-semibold text-zinc-500 dark:text-zinc-400 uppercase tracking-wider mb-3">API Reference</h4>
                    <div class="space-y-2 text-sm">
                        <div class="flex justify-between">
                            <span class="text-zinc-500 dark:text-zinc-400">Endpoint</span>
                            <code class="text-zinc-800 dark:text-zinc-200 text-xs">/api/v1/mcp/tools/call</code>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-zinc-500 dark:text-zinc-400">Method</span>
                            <code class="text-zinc-800 dark:text-zinc-200 text-xs">POST</code>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-zinc-500 dark:text-zinc-400">Auth</span>
                            <code class="text-zinc-800 dark:text-zinc-200 text-xs">Bearer token</code>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    {{-- History Panel (Collapsible Bottom) --}}
    <div
        x-show="showHistory"
        x-transition:enter="transition ease-out duration-200"
        x-transition:enter-start="opacity-0 translate-y-4"
        x-transition:enter-end="opacity-100 translate-y-0"
        x-transition:leave="transition ease-in duration-150"
        x-transition:leave-start="opacity-100 translate-y-0"
        x-transition:leave-end="opacity-0 translate-y-4"
        class="mt-6"
    >
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden">
            <div class="p-4 border-b border-zinc-200 dark:border-zinc-700 flex items-center justify-between">
                <h2 class="text-lg font-semibold text-zinc-900 dark:text-white flex items-center gap-2">
                    <svg class="w-5 h-5 text-cyan-500" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    Conversation History
                </h2>
                @if(count($conversationHistory) > 0)
                    <button
                        wire:click="clearHistory"
                        wire:confirm="Are you sure you want to clear your history?"
                        class="text-sm text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300"
                    >
                        Clear All
                    </button>
                @endif
            </div>

            @if(count($conversationHistory) > 0)
                <div class="divide-y divide-zinc-200 dark:divide-zinc-700 max-h-[300px] overflow-y-auto">
                    @foreach($conversationHistory as $index => $entry)
                        <div class="p-4 hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition-colors">
                            <div class="flex items-start justify-between">
                                <div class="flex-1 min-w-0">
                                    <div class="flex items-center gap-2">
                                        @if($entry['success'] ?? true)
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-emerald-100 dark:bg-emerald-900/30 text-emerald-800 dark:text-emerald-300">
                                                Success
                                            </span>
                                        @else
                                            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300">
                                                Failed
                                            </span>
                                        @endif
                                        <span class="font-medium text-zinc-900 dark:text-white">{{ $entry['tool'] }}</span>
                                        <span class="text-zinc-400">on</span>
                                        <span class="text-zinc-600 dark:text-zinc-400">{{ $entry['server'] }}</span>
                                    </div>
                                    <div class="mt-1 flex items-center gap-4 text-xs text-zinc-500 dark:text-zinc-400">
                                        <span>{{ \Carbon\Carbon::parse($entry['timestamp'])->diffForHumans() }}</span>
                                        @if(isset($entry['duration_ms']))
                                            <span>{{ $entry['duration_ms'] }}ms</span>
                                        @endif
                                    </div>
                                </div>
                                <div class="flex items-center gap-2 ml-4">
                                    <button
                                        wire:click="viewFromHistory({{ $index }})"
                                        class="px-2 py-1 text-xs font-medium text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white bg-zinc-100 dark:bg-zinc-700 hover:bg-zinc-200 dark:hover:bg-zinc-600 rounded transition-colors"
                                    >
                                        View
                                    </button>
                                    <button
                                        wire:click="rerunFromHistory({{ $index }})"
                                        class="px-2 py-1 text-xs font-medium text-cyan-600 dark:text-cyan-400 hover:text-cyan-700 dark:hover:text-cyan-300 bg-cyan-50 dark:bg-cyan-900/20 hover:bg-cyan-100 dark:hover:bg-cyan-900/30 rounded transition-colors"
                                    >
                                        Re-run
                                    </button>
                                </div>
                            </div>
                        </div>
                    @endforeach
                </div>
            @else
                <div class="p-8 text-center">
                    <p class="text-sm text-zinc-500 dark:text-zinc-400">No history yet. Execute a tool to see it here.</p>
                </div>
            @endif
        </div>
    </div>
</div>
