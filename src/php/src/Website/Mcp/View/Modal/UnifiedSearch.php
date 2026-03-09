<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Core\Search\Unified as UnifiedSearchService;
use Illuminate\Support\Collection;
use Livewire\Attributes\Layout;
use Livewire\Component;

/**
 * Unified Search Component
 *
 * Single search interface across all system components:
 * MCP tools, API endpoints, patterns, assets, todos, and plans.
 */
#[Layout('layouts::mcp')]
class UnifiedSearch extends Component
{
    public string $query = '';

    public array $selectedTypes = [];

    public int $limit = 50;

    protected UnifiedSearchService $searchService;

    public function boot(UnifiedSearchService $searchService): void
    {
        $this->searchService = $searchService;
    }

    public function updatedQuery(): void
    {
        // Debounce handled by wire:model.debounce
    }

    public function toggleType(string $type): void
    {
        if (in_array($type, $this->selectedTypes)) {
            $this->selectedTypes = array_values(array_diff($this->selectedTypes, [$type]));
        } else {
            $this->selectedTypes[] = $type;
        }
    }

    public function clearFilters(): void
    {
        $this->selectedTypes = [];
    }

    public function getResultsProperty(): Collection
    {
        if (strlen($this->query) < 2) {
            return collect();
        }

        return $this->searchService->search($this->query, $this->selectedTypes, $this->limit);
    }

    public function getTypesProperty(): array
    {
        return UnifiedSearchService::getTypes();
    }

    public function getResultCountsByTypeProperty(): array
    {
        if (strlen($this->query) < 2) {
            return [];
        }

        $allResults = $this->searchService->search($this->query, [], 200);

        return $allResults->groupBy('type')->map->count()->toArray();
    }

    public function render()
    {
        return view('mcp::web.unified-search');
    }
}
