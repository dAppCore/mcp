<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mcp\Models\McpApiRequest;
use Livewire\Attributes\Layout;
use Livewire\Component;
use Livewire\WithPagination;

/**
 * MCP Request Log - view and replay API requests.
 */
#[Layout('hub::admin.layouts.app')]
class RequestLog extends Component
{
    use WithPagination;

    public string $serverFilter = '';

    public string $statusFilter = '';

    public ?int $selectedRequestId = null;

    public ?McpApiRequest $selectedRequest = null;

    public function updatedServerFilter(): void
    {
        $this->resetPage();
    }

    public function updatedStatusFilter(): void
    {
        $this->resetPage();
    }

    public function selectRequest(int $id): void
    {
        $this->selectedRequestId = $id;
        $this->selectedRequest = McpApiRequest::find($id);
    }

    public function closeDetail(): void
    {
        $this->selectedRequestId = null;
        $this->selectedRequest = null;
    }

    public function render()
    {
        $workspace = auth()->user()?->defaultHostWorkspace();

        $query = McpApiRequest::query()
            ->orderByDesc('created_at');

        if ($workspace) {
            $query->forWorkspace($workspace->id);
        }

        if ($this->serverFilter) {
            $query->forServer($this->serverFilter);
        }

        if ($this->statusFilter === 'success') {
            $query->successful();
        } elseif ($this->statusFilter === 'failed') {
            $query->failed();
        }

        $requests = $query->paginate(20);

        // Get unique servers for filter dropdown
        $servers = McpApiRequest::query()
            ->when($workspace, fn ($q) => $q->forWorkspace($workspace->id))
            ->distinct()
            ->pluck('server_id')
            ->filter()
            ->values();

        return view('mcp::admin.request-log', [
            'requests' => $requests,
            'servers' => $servers,
        ]);
    }
}
