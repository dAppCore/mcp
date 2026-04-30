<div class="min-h-screen bg-gray-100 py-8">
    <div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
        <!-- Header -->
        <div class="mb-8">
            <h1 class="text-3xl font-bold text-gray-900">Search</h1>
            <p class="mt-2 text-gray-600">Find tools, endpoints, patterns, and more across the system</p>
        </div>

        <!-- Search Box -->
        <div class="bg-white rounded-lg shadow mb-6">
            <div class="p-4">
                <div class="relative">
                    <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <svg class="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                        </svg>
                    </div>
                    <input
                        wire:model.live.debounce.300ms="query"
                        type="text"
                        placeholder="Search for tools, endpoints, patterns..."
                        class="block w-full pl-10 pr-3 py-3 border border-gray-300 rounded-lg leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-lg"
                        autofocus
                    >
                </div>
            </div>

            <!-- Type Filters -->
            <div class="border-t px-4 py-3">
                <div class="flex items-center gap-2 flex-wrap">
                    <span class="text-sm text-gray-500">Filter:</span>
                    @foreach($this->types as $typeKey => $typeInfo)
                        <button
                            wire:click="toggleType('{{ $typeKey }}')"
                            class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm transition
                                {{ in_array($typeKey, $selectedTypes)
                                    ? 'bg-indigo-100 text-indigo-800 ring-2 ring-indigo-500'
                                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200' }}"
                        >
                            @if($typeInfo['icon'] === 'wrench')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
                            @elseif($typeInfo['icon'] === 'document')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
                            @elseif($typeInfo['icon'] === 'globe-alt')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                            @elseif($typeInfo['icon'] === 'puzzle-piece')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 4a2 2 0 114 0v1a1 1 0 001 1h3a1 1 0 011 1v3a1 1 0 01-1 1h-1a2 2 0 100 4h1a1 1 0 011 1v3a1 1 0 01-1 1h-3a1 1 0 01-1-1v-1a2 2 0 10-4 0v1a1 1 0 01-1 1H7a1 1 0 01-1-1v-3a1 1 0 00-1-1H4a2 2 0 110-4h1a1 1 0 001-1V7a1 1 0 011-1h3a1 1 0 001-1V4z"/></svg>
                            @elseif($typeInfo['icon'] === 'cube')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
                            @elseif($typeInfo['icon'] === 'clipboard-list')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"/></svg>
                            @elseif($typeInfo['icon'] === 'map')
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7"/></svg>
                            @endif
                            {{ $typeInfo['name'] }}
                            @if(isset($this->resultCountsByType[$typeKey]))
                                <span class="text-xs bg-white/50 px-1.5 rounded">{{ $this->resultCountsByType[$typeKey] }}</span>
                            @endif
                        </button>
                    @endforeach

                    @if(count($selectedTypes) > 0)
                        <button
                            wire:click="clearFilters"
                            class="text-sm text-gray-500 hover:text-gray-700 ml-2"
                        >
                            Clear filters
                        </button>
                    @endif
                </div>
            </div>
        </div>

        <!-- Results -->
        @if(strlen($query) >= 2)
            <div class="space-y-3">
                @forelse($this->results as $result)
                    <a
                        href="{{ $result['url'] }}"
                        class="block bg-white rounded-lg shadow hover:shadow-md transition p-4"
                    >
                        <div class="flex items-start gap-3">
                            <!-- Icon -->
                            <div class="flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center
                                {{ $result['type'] === 'mcp_tool' ? 'bg-purple-100 text-purple-600' : '' }}
                                {{ $result['type'] === 'mcp_resource' ? 'bg-blue-100 text-blue-600' : '' }}
                                {{ $result['type'] === 'api_endpoint' ? 'bg-green-100 text-green-600' : '' }}
                                {{ $result['type'] === 'pattern' ? 'bg-yellow-100 text-yellow-600' : '' }}
                                {{ $result['type'] === 'asset' ? 'bg-pink-100 text-pink-600' : '' }}
                                {{ $result['type'] === 'todo' ? 'bg-orange-100 text-orange-600' : '' }}
                                {{ $result['type'] === 'plan' ? 'bg-indigo-100 text-indigo-600' : '' }}
                            ">
                                @if($result['icon'] === 'wrench')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>
                                @elseif($result['icon'] === 'document')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
                                @elseif($result['icon'] === 'globe-alt')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
                                @elseif($result['icon'] === 'puzzle-piece')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 4a2 2 0 114 0v1a1 1 0 001 1h3a1 1 0 011 1v3a1 1 0 01-1 1h-1a2 2 0 100 4h1a1 1 0 011 1v3a1 1 0 01-1 1h-3a1 1 0 01-1-1v-1a2 2 0 10-4 0v1a1 1 0 01-1 1H7a1 1 0 01-1-1v-3a1 1 0 00-1-1H4a2 2 0 110-4h1a1 1 0 001-1V7a1 1 0 011-1h3a1 1 0 001-1V4z"/></svg>
                                @elseif($result['icon'] === 'cube')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
                                @elseif($result['icon'] === 'clipboard-list')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01"/></svg>
                                @elseif($result['icon'] === 'map')
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 20l-5.447-2.724A1 1 0 013 16.382V5.618a1 1 0 011.447-.894L9 7m0 13l6-3m-6 3V7m6 10l4.553 2.276A1 1 0 0021 18.382V7.618a1 1 0 00-.553-.894L15 4m0 13V4m0 0L9 7"/></svg>
                                @endif
                            </div>

                            <!-- Content -->
                            <div class="flex-1 min-w-0">
                                <div class="flex items-center gap-2">
                                    <h3 class="text-base font-medium text-gray-900 truncate">{{ $result['title'] }}</h3>
                                    <span class="flex-shrink-0 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                                        {{ $result['subtitle'] }}
                                    </span>
                                </div>
                                @if($result['description'])
                                    <p class="mt-1 text-sm text-gray-500 line-clamp-2">{{ $result['description'] }}</p>
                                @endif

                                <!-- Meta badges -->
                                @if(!empty($result['meta']))
                                    <div class="mt-2 flex items-center gap-2 flex-wrap">
                                        @if(isset($result['meta']['method']))
                                            <span class="px-2 py-0.5 rounded text-xs font-medium
                                                {{ $result['meta']['method'] === 'GET' ? 'bg-green-100 text-green-800' : '' }}
                                                {{ $result['meta']['method'] === 'POST' ? 'bg-blue-100 text-blue-800' : '' }}
                                                {{ $result['meta']['method'] === 'PUT' ? 'bg-yellow-100 text-yellow-800' : '' }}
                                                {{ $result['meta']['method'] === 'DELETE' ? 'bg-red-100 text-red-800' : '' }}
                                            ">{{ $result['meta']['method'] }}</span>
                                        @endif
                                        @if(isset($result['meta']['status']))
                                            <span class="px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                                                {{ $result['meta']['status'] }}
                                            </span>
                                        @endif
                                        @if(isset($result['meta']['priority']))
                                            <span class="px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-700">
                                                Priority: {{ $result['meta']['priority'] }}
                                            </span>
                                        @endif
                                        @if(isset($result['meta']['progress']))
                                            <span class="px-2 py-0.5 rounded text-xs font-medium bg-indigo-100 text-indigo-700">
                                                {{ $result['meta']['progress'] }}%
                                            </span>
                                        @endif
                                    </div>
                                @endif
                            </div>

                            <!-- Arrow -->
                            <div class="flex-shrink-0 text-gray-400">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                                </svg>
                            </div>
                        </div>
                    </a>
                @empty
                    <div class="bg-white rounded-lg shadow p-8 text-center">
                        <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
                        </svg>
                        <h3 class="mt-2 text-sm font-medium text-gray-900">No results found</h3>
                        <p class="mt-1 text-sm text-gray-500">Try a different search term or clear filters.</p>
                    </div>
                @endforelse
            </div>

            <!-- Results count -->
            @if($this->results->count() > 0)
                <div class="mt-4 text-center text-sm text-gray-500">
                    Showing {{ $this->results->count() }} result{{ $this->results->count() !== 1 ? 's' : '' }}
                </div>
            @endif
        @else
            <!-- Empty state -->
            <div class="bg-white rounded-lg shadow p-12 text-center">
                <svg class="mx-auto h-16 w-16 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                </svg>
                <h3 class="mt-4 text-lg font-medium text-gray-900">Start searching</h3>
                <p class="mt-2 text-gray-500">Type at least 2 characters to search across all system components.</p>
                <div class="mt-6 flex justify-center gap-2 flex-wrap">
                    @foreach($this->types as $typeKey => $typeInfo)
                        <span class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-sm bg-gray-100 text-gray-600">
                            {{ $typeInfo['name'] }}
                        </span>
                    @endforeach
                </div>
            </div>
        @endif

        <!-- Back Link -->
        <div class="mt-8">
            <a href="{{ url('/') }}" class="text-indigo-600 hover:text-indigo-800">
                &larr; Back to Dashboard
            </a>
        </div>
    </div>
</div>
