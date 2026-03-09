<div class="max-w-6xl mx-auto">
    <div class="mb-8">
        <h1 class="text-3xl font-bold text-zinc-900 dark:text-white">Request Log</h1>
        <p class="mt-2 text-lg text-zinc-600 dark:text-zinc-400">
            View API requests and generate curl commands to replay them.
        </p>
    </div>

    <!-- Filters -->
    <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-4 mb-6">
        <div class="flex flex-wrap gap-4">
            <div>
                <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Server</label>
                <select
                    wire:model.live="serverFilter"
                    class="px-3 py-1.5 bg-zinc-50 dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-600 rounded-lg text-sm"
                >
                    <option value="">All servers</option>
                    @foreach($servers as $server)
                        <option value="{{ $server }}">{{ $server }}</option>
                    @endforeach
                </select>
            </div>
            <div>
                <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Status</label>
                <select
                    wire:model.live="statusFilter"
                    class="px-3 py-1.5 bg-zinc-50 dark:bg-zinc-900 border border-zinc-300 dark:border-zinc-600 rounded-lg text-sm"
                >
                    <option value="">All</option>
                    <option value="success">Success</option>
                    <option value="failed">Failed</option>
                </select>
            </div>
        </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Request List -->
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 overflow-hidden">
            <div class="divide-y divide-zinc-200 dark:divide-zinc-700">
                @forelse($requests as $request)
                    <button
                        wire:click="selectRequest({{ $request->id }})"
                        class="w-full text-left p-4 hover:bg-zinc-50 dark:hover:bg-zinc-700/50 transition-colors {{ $selectedRequestId === $request->id ? 'bg-cyan-50 dark:bg-cyan-900/20' : '' }}"
                    >
                        <div class="flex items-center justify-between mb-1">
                            <div class="flex items-center space-x-2">
                                <span class="text-xs font-mono px-1.5 py-0.5 rounded {{ $request->isSuccessful() ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' }}">
                                    {{ $request->response_status }}
                                </span>
                                <span class="text-sm font-medium text-zinc-900 dark:text-white">
                                    {{ $request->server_id }}/{{ $request->tool_name }}
                                </span>
                            </div>
                            <span class="text-xs text-zinc-500 dark:text-zinc-400">
                                {{ $request->duration_for_humans }}
                            </span>
                        </div>
                        <div class="text-xs text-zinc-500 dark:text-zinc-400">
                            {{ $request->created_at->diffForHumans() }}
                            <span class="mx-1">&middot;</span>
                            {{ $request->request_id }}
                        </div>
                    </button>
                @empty
                    <div class="p-8 text-center text-zinc-500 dark:text-zinc-400">
                        No requests found.
                    </div>
                @endforelse
            </div>

            @if($requests->hasPages())
                <div class="p-4 border-t border-zinc-200 dark:border-zinc-700">
                    {{ $requests->links() }}
                </div>
            @endif
        </div>

        <!-- Request Detail -->
        <div class="bg-white dark:bg-zinc-800 rounded-xl border border-zinc-200 dark:border-zinc-700 p-6">
            @if($selectedRequest)
                <div class="flex items-center justify-between mb-4">
                    <h2 class="text-lg font-semibold text-zinc-900 dark:text-white">Request Detail</h2>
                    <button
                        wire:click="closeDetail"
                        class="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                    >
                        <flux:icon.x-mark class="w-5 h-5" />
                    </button>
                </div>

                <div class="space-y-4">
                    <!-- Status -->
                    <div>
                        <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Status</label>
                        <span class="inline-flex items-center px-2 py-1 rounded text-sm font-medium {{ $selectedRequest->isSuccessful() ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' }}">
                            {{ $selectedRequest->response_status }}
                            {{ $selectedRequest->isSuccessful() ? 'OK' : 'Error' }}
                        </span>
                    </div>

                    <!-- Request Body -->
                    <div>
                        <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Request</label>
                        <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-3 text-xs overflow-x-auto">{{ json_encode($selectedRequest->request_body, JSON_PRETTY_PRINT) }}</pre>
                    </div>

                    <!-- Response Body -->
                    <div>
                        <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Response</label>
                        <pre class="bg-zinc-100 dark:bg-zinc-900 rounded-lg p-3 text-xs overflow-x-auto max-h-48">{{ json_encode($selectedRequest->response_body, JSON_PRETTY_PRINT) }}</pre>
                    </div>

                    @if($selectedRequest->error_message)
                        <div>
                            <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">Error</label>
                            <pre class="bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400 rounded-lg p-3 text-xs">{{ $selectedRequest->error_message }}</pre>
                        </div>
                    @endif

                    <!-- Curl Command -->
                    <div x-data="{ copied: false }">
                        <label class="block text-xs font-medium text-zinc-500 dark:text-zinc-400 mb-1">
                            Replay Command
                            <button
                                x-on:click="navigator.clipboard.writeText($refs.curl.textContent); copied = true; setTimeout(() => copied = false, 2000)"
                                class="ml-2 text-cyan-600 hover:text-cyan-700"
                            >
                                <span x-show="!copied">Copy</span>
                                <span x-show="copied" x-cloak>Copied</span>
                            </button>
                        </label>
                        <pre x-ref="curl" class="bg-zinc-900 dark:bg-zinc-950 text-emerald-400 rounded-lg p-3 text-xs overflow-x-auto">{{ $selectedRequest->toCurl() }}</pre>
                    </div>

                    <!-- Metadata -->
                    <div class="pt-4 border-t border-zinc-200 dark:border-zinc-700 text-xs text-zinc-500 dark:text-zinc-400 space-y-1">
                        <div>Request ID: {{ $selectedRequest->request_id }}</div>
                        <div>Duration: {{ $selectedRequest->duration_for_humans }}</div>
                        <div>IP: {{ $selectedRequest->ip_address ?? 'N/A' }}</div>
                        <div>Time: {{ $selectedRequest->created_at->format('Y-m-d H:i:s') }}</div>
                    </div>
                </div>
            @else
                <div class="text-center py-12 text-zinc-500 dark:text-zinc-400">
                    <flux:icon.document-text class="w-12 h-12 mx-auto mb-4 opacity-50" />
                    <p>Select a request to view details and generate replay commands.</p>
                </div>
            @endif
        </div>
    </div>
</div>
