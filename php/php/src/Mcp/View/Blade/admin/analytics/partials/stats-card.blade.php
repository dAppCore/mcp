@props([
    'label',
    'value',
    'color' => 'default',
    'subtext' => null,
])

@php
    $colorClasses = match($color) {
        'red' => 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800',
        'yellow' => 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800',
        'green' => 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800',
        'blue' => 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800',
        default => 'bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700',
    };

    $valueClasses = match($color) {
        'red' => 'text-red-600 dark:text-red-400',
        'yellow' => 'text-yellow-600 dark:text-yellow-400',
        'green' => 'text-green-600 dark:text-green-400',
        'blue' => 'text-blue-600 dark:text-blue-400',
        default => '',
    };
@endphp

<div class="p-4 rounded-lg border {{ $colorClasses }}">
    <flux:subheading>{{ $label }}</flux:subheading>
    <flux:heading size="xl" class="{{ $valueClasses }}">{{ $value }}</flux:heading>
    @if($subtext)
        <span class="text-sm text-zinc-500">{{ $subtext }}</span>
    @endif
</div>
