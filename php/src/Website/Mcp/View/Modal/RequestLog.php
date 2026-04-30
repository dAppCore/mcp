<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Livewire\Attributes\Layout;
use Livewire\Component;
use Livewire\WithPagination;
use Core\Mcp\Models\McpApiRequest;

/**
 * MCP Request Log - view and replay API requests.
 */
#[Layout('layouts::mcp')]
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
        $workspace = auth()->user()?->defaultHostWorkspace();

        // Only allow selecting requests that belong to the user's workspace
        $request = McpApiRequest::query()
            ->when($workspace, fn ($q) => $q->forWorkspace($workspace->id))
            ->find($id);

        if (! $request) {
            $this->selectedRequestId = null;
            $this->selectedRequest = null;

            return;
        }

        $this->selectedRequestId = $id;
        $this->selectedRequest = $request;
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

        return view('mcp::web.request-log', [
            'requests' => $requests,
            'servers' => $servers,
        ]);
    }
}
