@php
    $appName = config('core.app.name', 'Core PHP');
    $appUrl = config('app.url', 'https://core.test');
    $privacyUrl = config('core.urls.privacy', '/privacy');
    $termsUrl = config('core.urls.terms', '/terms');
@endphp
<!DOCTYPE html>
<html lang="{{ str_replace('_', '-', app()->getLocale()) }}" class="dark">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">

    <title>{{ $title ?? 'MCP Portal' }} - {{ $appName }}</title>
    <meta name="description" content="{{ $description ?? 'Connect AI agents via Model Context Protocol' }}">

    @vite(['resources/css/app.css', 'resources/js/app.js'])
    @fluxAppearance
</head>
<body class="min-h-screen bg-white dark:bg-zinc-900">
    <!-- Header -->
    <header class="border-b border-zinc-200 dark:border-zinc-800">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
            <div class="flex items-center justify-between h-16">
                <div class="flex items-center space-x-8">
                    <a href="{{ route('mcp.landing') }}" class="flex items-center space-x-2">
                        <span class="text-xl font-bold text-zinc-900 dark:text-white">MCP Portal</span>
                    </a>
                    <nav class="hidden md:flex items-center space-x-6 text-sm">
                        <a href="{{ route('mcp.servers.index') }}" class="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                            Servers
                        </a>
                        <a href="{{ route('mcp.connect') }}" class="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                            Setup Guide
                        </a>
                    </nav>
                </div>
                <div class="flex items-center space-x-4">
                    @php
                        $workspace = request()->attributes->get('mcp_workspace');
                    @endphp
                    @if($workspace)
                        <a href="{{ route('mcp.dashboard') }}" class="text-sm text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                            Dashboard
                        </a>
                        <a href="{{ route('mcp.keys') }}" class="text-sm text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                            API Keys
                        </a>
                        <span class="text-sm text-zinc-500 dark:text-zinc-400">
                            {{ $workspace->name }}
                        </span>
                    @else
                        <a href="{{ route('login') }}" class="text-sm text-cyan-600 hover:text-cyan-700 dark:text-cyan-400 dark:hover:text-cyan-300">
                            Sign in
                        </a>
                    @endif
                    <a href="{{ $appUrl }}" class="text-sm text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white">
                        &larr; {{ $appName }}
                    </a>
                </div>
            </div>
        </div>
    </header>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        {{ $slot }}
    </main>

    <!-- Footer -->
    <footer class="border-t border-zinc-200 dark:border-zinc-800 mt-auto">
        <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
            <div class="flex items-center justify-between">
                <p class="text-sm text-zinc-500 dark:text-zinc-400">
                    &copy; {{ date('Y') }} {{ $appName }}. All rights reserved.
                </p>
                <div class="flex items-center space-x-6 text-sm text-zinc-500 dark:text-zinc-400">
                    <a href="{{ $privacyUrl }}" class="hover:text-zinc-900 dark:hover:text-white">Privacy</a>
                    <a href="{{ $termsUrl }}" class="hover:text-zinc-900 dark:hover:text-white">Terms</a>
                </div>
            </div>
        </div>
    </footer>

    @fluxScripts
</body>
</html>
