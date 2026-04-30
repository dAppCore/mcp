<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mcp\Models\McpAuditLog;
use Core\Mcp\Services\AuditLogService;
use Core\Tenant\Models\Workspace;
use Illuminate\Contracts\Pagination\LengthAwarePaginator;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithPagination;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * MCP Audit Log Viewer.
 *
 * Admin interface for viewing and exporting immutable tool execution logs.
 * Includes integrity verification and compliance export features.
 */
#[Title('MCP Audit Log')]
#[Layout('hub::admin.layouts.app')]
class AuditLogViewer extends Component
{
    use WithPagination;

    #[Url]
    public string $search = '';

    #[Url]
    public string $tool = '';

    #[Url]
    public string $workspace = '';

    #[Url]
    public string $status = '';

    #[Url]
    public string $sensitivity = '';

    #[Url]
    public string $dateFrom = '';

    #[Url]
    public string $dateTo = '';

    public int $perPage = 25;

    public ?int $selectedEntryId = null;

    public ?array $integrityStatus = null;

    public bool $showIntegrityModal = false;

    public bool $showExportModal = false;

    public string $exportFormat = 'json';

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function entries(): LengthAwarePaginator
    {
        $query = McpAuditLog::query()
            ->with('workspace')
            ->orderByDesc('id');

        if ($this->search) {
            $query->where(function ($q) {
                $q->where('tool_name', 'like', "%{$this->search}%")
                    ->orWhere('server_id', 'like', "%{$this->search}%")
                    ->orWhere('session_id', 'like', "%{$this->search}%")
                    ->orWhere('error_message', 'like', "%{$this->search}%");
            });
        }

        if ($this->tool) {
            $query->where('tool_name', $this->tool);
        }

        if ($this->workspace) {
            $query->where('workspace_id', $this->workspace);
        }

        if ($this->status === 'success') {
            $query->where('success', true);
        } elseif ($this->status === 'failed') {
            $query->where('success', false);
        }

        if ($this->sensitivity === 'sensitive') {
            $query->where('is_sensitive', true);
        } elseif ($this->sensitivity === 'normal') {
            $query->where('is_sensitive', false);
        }

        if ($this->dateFrom) {
            $query->where('created_at', '>=', Carbon::parse($this->dateFrom)->startOfDay());
        }

        if ($this->dateTo) {
            $query->where('created_at', '<=', Carbon::parse($this->dateTo)->endOfDay());
        }

        return $query->paginate($this->perPage);
    }

    #[Computed]
    public function workspaces(): Collection
    {
        return Workspace::orderBy('name')->get(['id', 'name']);
    }

    #[Computed]
    public function tools(): Collection
    {
        return McpAuditLog::query()
            ->select('tool_name')
            ->distinct()
            ->orderBy('tool_name')
            ->pluck('tool_name');
    }

    #[Computed]
    public function selectedEntry(): ?McpAuditLog
    {
        if (! $this->selectedEntryId) {
            return null;
        }

        return McpAuditLog::with('workspace')->find($this->selectedEntryId);
    }

    #[Computed]
    public function stats(): array
    {
        return app(AuditLogService::class)->getStats(
            workspaceId: $this->workspace ? (int) $this->workspace : null,
            days: 30
        );
    }

    public function viewEntry(int $id): void
    {
        $this->selectedEntryId = $id;
    }

    public function closeEntryDetail(): void
    {
        $this->selectedEntryId = null;
    }

    public function verifyIntegrity(): void
    {
        $this->integrityStatus = app(AuditLogService::class)->verifyChain();
        $this->showIntegrityModal = true;
    }

    public function closeIntegrityModal(): void
    {
        $this->showIntegrityModal = false;
        $this->integrityStatus = null;
    }

    public function openExportModal(): void
    {
        $this->showExportModal = true;
    }

    public function closeExportModal(): void
    {
        $this->showExportModal = false;
    }

    public function export(): StreamedResponse
    {
        $auditLogService = app(AuditLogService::class);

        $workspaceId = $this->workspace ? (int) $this->workspace : null;
        $from = $this->dateFrom ? Carbon::parse($this->dateFrom) : null;
        $to = $this->dateTo ? Carbon::parse($this->dateTo) : null;
        $tool = $this->tool ?: null;
        $sensitiveOnly = $this->sensitivity === 'sensitive';

        if ($this->exportFormat === 'csv') {
            $content = $auditLogService->exportToCsv($workspaceId, $from, $to, $tool, $sensitiveOnly);
            $filename = 'mcp-audit-log-'.now()->format('Y-m-d-His').'.csv';
            $contentType = 'text/csv';
        } else {
            $content = $auditLogService->exportToJson($workspaceId, $from, $to, $tool, $sensitiveOnly);
            $filename = 'mcp-audit-log-'.now()->format('Y-m-d-His').'.json';
            $contentType = 'application/json';
        }

        return response()->streamDownload(function () use ($content) {
            echo $content;
        }, $filename, [
            'Content-Type' => $contentType,
        ]);
    }

    public function clearFilters(): void
    {
        $this->search = '';
        $this->tool = '';
        $this->workspace = '';
        $this->status = '';
        $this->sensitivity = '';
        $this->dateFrom = '';
        $this->dateTo = '';
        $this->resetPage();
    }

    public function getStatusBadgeClass(bool $success): string
    {
        return $success
            ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300'
            : 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300';
    }

    public function getSensitivityBadgeClass(bool $isSensitive): string
    {
        return $isSensitive
            ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300'
            : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300';
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render()
    {
        return view('mcp::admin.audit-log-viewer');
    }
}
