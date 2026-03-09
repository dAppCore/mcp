<div class="min-h-screen bg-gray-100 py-8">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <!-- Header -->
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-gray-900">MCP Tool Playground</h1>
            <p class="mt-2 text-gray-600">Test MCP tool calls with custom parameters</p>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <!-- Input Panel -->
            <div class="bg-white rounded-lg shadow">
                <div class="p-6">
                    <h2 class="text-lg font-semibold mb-4">Request</h2>

                    <!-- Server Selection -->
                    <div class="mb-4">
                        <label for="server" class="block text-sm font-medium text-gray-700 mb-1">Server</label>
                        <select
                            wire:model.live="selectedServer"
                            id="server"
                            class="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
                        >
                            <option value="">Select a server...</option>
                            @foreach($servers as $server)
                                <option value="{{ $server['id'] }}">
                                    {{ $server['name'] }} ({{ $server['tool_count'] }} tools)
                                </option>
                            @endforeach
                        </select>
                    </div>

                    <!-- Tool Selection -->
                    <div class="mb-4">
                        <label for="tool" class="block text-sm font-medium text-gray-700 mb-1">Tool</label>
                        <select
                            wire:model.live="selectedTool"
                            id="tool"
                            class="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500"
                            @if(empty($selectedServer)) disabled @endif
                        >
                            <option value="">Select a tool...</option>
                            @foreach($tools as $tool)
                                <option value="{{ $tool['name'] }}">
                                    {{ $tool['name'] }}
                                </option>
                            @endforeach
                        </select>
                        @if($selectedTool)
                            @php $currentTool = collect($tools)->firstWhere('name', $selectedTool); @endphp
                            @if($currentTool && !empty($currentTool['purpose']))
                                <p class="mt-1 text-sm text-gray-500">{{ $currentTool['purpose'] }}</p>
                            @endif
                        @endif
                    </div>

                    <!-- Parameters JSON -->
                    <div class="mb-4">
                        <div class="flex justify-between items-center mb-1">
                            <label for="params" class="block text-sm font-medium text-gray-700">Parameters (JSON)</label>
                            <button
                                wire:click="formatJson"
                                type="button"
                                class="text-xs text-indigo-600 hover:text-indigo-800"
                            >
                                Format JSON
                            </button>
                        </div>
                        <textarea
                            wire:model="inputJson"
                            id="params"
                            rows="10"
                            class="w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 font-mono text-sm"
                            placeholder="{}"
                        ></textarea>
                        @error('inputJson')
                            <p class="mt-1 text-sm text-red-600">{{ $message }}</p>
                        @enderror
                    </div>

                    <!-- Execute Button -->
                    <button
                        wire:click="execute"
                        wire:loading.attr="disabled"
                        type="button"
                        class="w-full inline-flex justify-center items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        @if(empty($selectedServer) || empty($selectedTool)) disabled @endif
                    >
                        <span wire:loading.remove wire:target="execute">Execute Tool</span>
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

            <!-- Output Panel -->
            <div class="bg-white rounded-lg shadow">
                <div class="p-6">
                    <div class="flex justify-between items-center mb-4">
                        <h2 class="text-lg font-semibold">Response</h2>
                        @if($executionTime > 0)
                            <span class="text-sm text-gray-500">{{ $executionTime }}ms</span>
                        @endif
                    </div>

                    @if($lastError)
                        <div class="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
                            <div class="flex">
                                <div class="flex-shrink-0">
                                    <svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
                                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd"/>
                                    </svg>
                                </div>
                                <div class="ml-3">
                                    <h3 class="text-sm font-medium text-red-800">Error</h3>
                                    <p class="mt-1 text-sm text-red-700">{{ $lastError }}</p>
                                </div>
                            </div>
                        </div>
                    @endif

                    <div class="bg-gray-900 rounded-md overflow-hidden">
                        <pre class="p-4 text-sm text-gray-100 overflow-x-auto" style="max-height: 500px;">@if($lastResult){{ json_encode($lastResult, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE) }}@else<span class="text-gray-500">// Response will appear here...</span>@endif</pre>
                    </div>
                </div>
            </div>
        </div>

        <!-- Tool Parameters Reference -->
        @if($selectedTool && !empty($tools))
            @php $currentTool = collect($tools)->firstWhere('name', $selectedTool); @endphp
            @if($currentTool && !empty($currentTool['parameters']))
                <div class="mt-6 bg-white rounded-lg shadow">
                    <div class="p-6">
                        <h2 class="text-lg font-semibold mb-4">Parameter Reference</h2>
                        <div class="overflow-x-auto">
                            <table class="min-w-full divide-y divide-gray-200">
                                <thead class="bg-gray-50">
                                    <tr>
                                        <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                                        <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
                                        <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Required</th>
                                        <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">Description</th>
                                    </tr>
                                </thead>
                                <tbody class="bg-white divide-y divide-gray-200">
                                    @foreach($currentTool['parameters'] as $paramName => $paramDef)
                                        <tr>
                                            <td class="px-4 py-2 text-sm font-mono text-gray-900">{{ $paramName }}</td>
                                            <td class="px-4 py-2 text-sm text-gray-500">{{ is_array($paramDef) ? ($paramDef['type'] ?? 'string') : 'string' }}</td>
                                            <td class="px-4 py-2 text-sm">
                                                @if(is_array($paramDef) && ($paramDef['required'] ?? false))
                                                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">Required</span>
                                                @else
                                                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800">Optional</span>
                                                @endif
                                            </td>
                                            <td class="px-4 py-2 text-sm text-gray-500">{{ is_array($paramDef) ? ($paramDef['description'] ?? '-') : $paramDef }}</td>
                                        </tr>
                                    @endforeach
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            @endif
        @endif

        <!-- Back Link -->
        <div class="mt-6">
            <a href="{{ route('mcp.landing') }}" class="text-indigo-600 hover:text-indigo-800">
                &larr; Back to MCP Portal
            </a>
        </div>
    </div>
</div>
