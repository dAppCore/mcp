<div class="min-h-screen bg-gray-100 py-8">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <!-- Header -->
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-gray-900">API Explorer</h1>
            <p class="mt-2 text-gray-600">Interactive documentation with code snippets in 11 languages</p>
        </div>

        <!-- API Key Input -->
        <div class="bg-yellow-50 border border-yellow-200 rounded-lg p-4 mb-6">
            <div class="flex items-start gap-3">
                <svg class="w-5 h-5 text-yellow-600 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/>
                </svg>
                <div class="flex-1">
                    <label for="apiKey" class="block text-sm font-medium text-yellow-800 mb-1">API Key</label>
                    <div class="flex gap-2">
                        <input
                            wire:model="apiKey"
                            type="password"
                            id="apiKey"
                            placeholder="hk_live_..."
                            class="flex-1 rounded-md border-yellow-300 shadow-sm focus:border-yellow-500 focus:ring-yellow-500 text-sm"
                        >
                        <select
                            wire:model="baseUrl"
                            class="rounded-md border-yellow-300 shadow-sm focus:border-yellow-500 focus:ring-yellow-500 text-sm"
                        >
                            <option value="https://mcp.lthn.ai">Production</option>
                            <option value="https://mcp.lthn.sh">Homelab</option>
                            <option value="http://localhost">Local</option>
                        </select>
                    </div>
                    <p class="mt-1 text-xs text-yellow-700">Enter your API key to enable live testing. Keys are not stored.</p>
                </div>
            </div>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <!-- Endpoint Selector -->
            <div class="lg:col-span-1">
                <div class="bg-white rounded-lg shadow">
                    <div class="p-4 border-b">
                        <h2 class="text-lg font-semibold">Endpoints</h2>
                    </div>
                    <div class="divide-y max-h-[600px] overflow-y-auto">
                        @foreach($endpoints as $index => $endpoint)
                            <button
                                wire:click="selectEndpoint({{ $index }})"
                                class="w-full text-left p-4 hover:bg-gray-50 transition {{ $selectedEndpoint == $index ? 'bg-indigo-50 border-l-4 border-indigo-500' : '' }}"
                            >
                                <div class="flex items-center gap-2 mb-1">
                                    <span class="px-2 py-0.5 rounded text-xs font-medium
                                        {{ $endpoint['method'] === 'GET' ? 'bg-green-100 text-green-800' : '' }}
                                        {{ $endpoint['method'] === 'POST' ? 'bg-blue-100 text-blue-800' : '' }}
                                        {{ $endpoint['method'] === 'PUT' ? 'bg-yellow-100 text-yellow-800' : '' }}
                                        {{ $endpoint['method'] === 'DELETE' ? 'bg-red-100 text-red-800' : '' }}
                                    ">
                                        {{ $endpoint['method'] }}
                                    </span>
                                    <span class="font-medium text-gray-900">{{ $endpoint['name'] }}</span>
                                </div>
                                <code class="text-xs text-gray-500">{{ $endpoint['path'] }}</code>
                            </button>
                        @endforeach
                    </div>
                </div>
            </div>

            <!-- Code Snippets & Testing -->
            <div class="lg:col-span-2 space-y-6">
                <!-- Request Configuration -->
                <div class="bg-white rounded-lg shadow">
                    <div class="p-4 border-b">
                        <h2 class="text-lg font-semibold">Request</h2>
                    </div>
                    <div class="p-4 space-y-4">
                        <div class="flex gap-2">
                            <select
                                wire:model.live="method"
                                class="rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
                            >
                                <option value="GET">GET</option>
                                <option value="POST">POST</option>
                                <option value="PUT">PUT</option>
                                <option value="PATCH">PATCH</option>
                                <option value="DELETE">DELETE</option>
                            </select>
                            <input
                                wire:model.live="path"
                                type="text"
                                class="flex-1 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 font-mono text-sm"
                                placeholder="/api/v1/..."
                            >
                        </div>

                        @if(in_array($method, ['POST', 'PUT', 'PATCH']))
                            <div>
                                <div class="flex justify-between items-center mb-1">
                                    <label class="block text-sm font-medium text-gray-700">Request Body (JSON)</label>
                                    <button wire:click="formatBody" class="text-xs text-indigo-600 hover:text-indigo-800">Format</button>
                                </div>
                                <textarea
                                    wire:model="bodyJson"
                                    rows="4"
                                    class="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 font-mono text-sm"
                                ></textarea>
                            </div>
                        @endif

                        <button
                            wire:click="sendRequest"
                            wire:loading.attr="disabled"
                            class="w-full inline-flex justify-center items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
                        >
                            <span wire:loading.remove wire:target="sendRequest">Send Request</span>
                            <span wire:loading wire:target="sendRequest" class="flex items-center">
                                <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24">
                                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                </svg>
                                Sending...
                            </span>
                        </button>
                    </div>
                </div>

                <!-- Code Snippets -->
                <div class="bg-white rounded-lg shadow">
                    <div class="p-4 border-b">
                        <div class="flex justify-between items-center">
                            <h2 class="text-lg font-semibold">Code Snippet</h2>
                            <button
                                wire:click="copyToClipboard"
                                class="text-sm text-indigo-600 hover:text-indigo-800 flex items-center gap-1"
                            >
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/>
                                </svg>
                                Copy
                            </button>
                        </div>
                    </div>

                    <!-- Language Tabs -->
                    <div class="border-b overflow-x-auto">
                        <div class="flex px-4">
                            @foreach($languages as $lang)
                                <button
                                    wire:click="$set('selectedLanguage', '{{ $lang['code'] }}')"
                                    class="px-3 py-2 text-sm font-medium border-b-2 whitespace-nowrap {{ $selectedLanguage === $lang['code'] ? 'border-indigo-500 text-indigo-600' : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300' }}"
                                >
                                    {{ $lang['name'] }}
                                </button>
                            @endforeach
                        </div>
                    </div>

                    <!-- Code Display -->
                    <div class="bg-gray-900 rounded-b-lg">
                        <pre class="p-4 text-sm text-gray-100 overflow-x-auto" style="max-height: 300px;"><code>{{ $snippet }}</code></pre>
                    </div>
                </div>

                <!-- Response -->
                @if($error)
                    <div class="bg-red-50 border border-red-200 rounded-lg p-4">
                        <div class="flex">
                            <svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                            </svg>
                            <div class="ml-3">
                                <h3 class="text-sm font-medium text-red-800">Error</h3>
                                <p class="mt-1 text-sm text-red-700">{{ $error }}</p>
                            </div>
                        </div>
                    </div>
                @endif

                @if($response)
                    <div class="bg-white rounded-lg shadow">
                        <div class="p-4 border-b flex justify-between items-center">
                            <div class="flex items-center gap-3">
                                <h2 class="text-lg font-semibold">Response</h2>
                                <span class="px-2 py-0.5 rounded text-xs font-medium
                                    {{ $response['status'] >= 200 && $response['status'] < 300 ? 'bg-green-100 text-green-800' : '' }}
                                    {{ $response['status'] >= 400 ? 'bg-red-100 text-red-800' : '' }}
                                ">
                                    {{ $response['status'] }}
                                </span>
                            </div>
                            <span class="text-sm text-gray-500">{{ $responseTime }}ms</span>
                        </div>
                        <div class="bg-gray-900 rounded-b-lg">
                            <pre class="p-4 text-sm text-gray-100 overflow-x-auto" style="max-height: 400px;"><code>{{ json_encode($response['body'], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) }}</code></pre>
                        </div>
                    </div>
                @endif
            </div>
        </div>

        <!-- Back Link -->
        <div class="mt-8">
            <a href="{{ url('/') }}" class="text-indigo-600 hover:text-indigo-800">
                &larr; Back to Home
            </a>
        </div>
    </div>

    @script
    <script>
        $wire.on('copy-to-clipboard', ({ code }) => {
            navigator.clipboard.writeText(code).then(() => {
                // Could show a toast notification here
            });
        });
    </script>
    @endscript
</div>
