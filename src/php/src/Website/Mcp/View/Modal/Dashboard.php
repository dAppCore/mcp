<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Livewire\Attributes\Layout;
use Livewire\Component;
use Livewire\WithPagination;
use Mod\Uptelligence\Models\AnalysisLog;
use Mod\Uptelligence\Models\Asset;
use Mod\Uptelligence\Models\Pattern;
use Mod\Uptelligence\Models\UpstreamTodo;
use Mod\Uptelligence\Models\Vendor;

/**
 * MCP Dashboard
 *
 * @todo This file incorrectly references Uptelligence models. Needs to be rewritten
 *       to show MCP servers, tools, and usage stats instead.
 *
 * Unified view of vendors, todos, assets, and patterns.
 */
#[Layout('hub::admin.layouts.app')]
class Dashboard extends Component
{
    use WithPagination;

    public string $vendorFilter = '';

    public string $typeFilter = '';

    public string $statusFilter = 'pending';

    public string $effortFilter = '';

    public bool $quickWinsOnly = false;

    public function updatingVendorFilter(): void
    {
        $this->resetPage();
    }

    public function updatingTypeFilter(): void
    {
        $this->resetPage();
    }

    public function updatingStatusFilter(): void
    {
        $this->resetPage();
    }

    public function getVendorsProperty()
    {
        try {
            return Vendor::active()->withCount(['todos', 'releases'])->get();
        } catch (\Illuminate\Database\QueryException $e) {
            return collect();
        }
    }

    public function getStatsProperty(): array
    {
        try {
            return [
                'total_vendors' => Vendor::active()->count(),
                'pending_todos' => UpstreamTodo::pending()->count(),
                'quick_wins' => UpstreamTodo::quickWins()->count(),
                'security_updates' => UpstreamTodo::pending()->where('type', 'security')->count(),
                'recent_releases' => \Mod\Uptelligence\Models\VersionRelease::recent(7)->count(),
                'in_progress' => UpstreamTodo::inProgress()->count(),
            ];
        } catch (\Illuminate\Database\QueryException $e) {
            return [
                'total_vendors' => 0,
                'pending_todos' => 0,
                'quick_wins' => 0,
                'security_updates' => 0,
                'recent_releases' => 0,
                'in_progress' => 0,
            ];
        }
    }

    public function getTodosProperty()
    {
        try {
            $query = UpstreamTodo::with('vendor')
                ->orderByDesc('priority')
                ->orderBy('effort');

            if ($this->vendorFilter) {
                $query->where('vendor_id', $this->vendorFilter);
            }

            if ($this->typeFilter) {
                $query->where('type', $this->typeFilter);
            }

            if ($this->statusFilter) {
                $query->where('status', $this->statusFilter);
            }

            if ($this->effortFilter) {
                $query->where('effort', $this->effortFilter);
            }

            if ($this->quickWinsOnly) {
                $query->where('effort', 'low')->where('priority', '>=', 5);
            }

            return $query->paginate(15);
        } catch (\Illuminate\Database\QueryException $e) {
            return new \Illuminate\Pagination\LengthAwarePaginator([], 0, 15);
        }
    }

    public function getRecentLogsProperty()
    {
        try {
            return AnalysisLog::with('vendor')
                ->latest()
                ->limit(10)
                ->get();
        } catch (\Illuminate\Database\QueryException $e) {
            return collect();
        }
    }

    public function getAssetsProperty()
    {
        try {
            return Asset::active()->orderBy('type')->get();
        } catch (\Illuminate\Database\QueryException $e) {
            return collect();
        }
    }

    public function getPatternsProperty()
    {
        try {
            return Pattern::active()->orderBy('category')->limit(6)->get();
        } catch (\Illuminate\Database\QueryException $e) {
            return collect();
        }
    }

    public function getAssetStatsProperty(): array
    {
        try {
            return [
                'total' => Asset::active()->count(),
                'updates_available' => Asset::active()->needsUpdate()->count(),
                'patterns' => Pattern::active()->count(),
            ];
        } catch (\Illuminate\Database\QueryException $e) {
            return [
                'total' => 0,
                'updates_available' => 0,
                'patterns' => 0,
            ];
        }
    }

    public function markInProgress(int $todoId): void
    {
        $todo = UpstreamTodo::findOrFail($todoId);
        $todo->markInProgress();
    }

    public function markPorted(int $todoId): void
    {
        $todo = UpstreamTodo::findOrFail($todoId);
        $todo->markPorted();
    }

    public function markSkipped(int $todoId): void
    {
        $todo = UpstreamTodo::findOrFail($todoId);
        $todo->markSkipped();
    }

    public function render()
    {
        return view('mcp::web.dashboard');
    }
}
