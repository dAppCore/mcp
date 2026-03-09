<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mcp\Models\McpToolVersion;
use Core\Mcp\Services\ToolVersionService;
use Illuminate\Contracts\Pagination\LengthAwarePaginator;
use Illuminate\Support\Carbon;
use Illuminate\Support\Collection;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Attributes\Url;
use Livewire\Component;
use Livewire\WithPagination;

/**
 * MCP Tool Version Manager.
 *
 * Admin interface for managing tool version lifecycles,
 * viewing schema changes, and setting deprecation schedules.
 */
#[Title('Tool Versions')]
#[Layout('hub::admin.layouts.app')]
class ToolVersionManager extends Component
{
    use WithPagination;

    #[Url]
    public string $search = '';

    #[Url]
    public string $server = '';

    #[Url]
    public string $status = '';

    public int $perPage = 25;

    // Modal state
    public bool $showVersionDetail = false;

    public ?int $selectedVersionId = null;

    public bool $showCompareModal = false;

    public ?int $compareFromId = null;

    public ?int $compareToId = null;

    public bool $showDeprecateModal = false;

    public ?int $deprecateVersionId = null;

    public string $deprecateSunsetDate = '';

    public bool $showRegisterModal = false;

    public string $registerServer = '';

    public string $registerTool = '';

    public string $registerVersion = '';

    public string $registerDescription = '';

    public string $registerChangelog = '';

    public string $registerMigrationNotes = '';

    public string $registerInputSchema = '';

    public bool $registerMarkLatest = false;

    public function mount(): void
    {
        $this->checkHadesAccess();
    }

    #[Computed]
    public function versions(): LengthAwarePaginator
    {
        $query = McpToolVersion::query()
            ->orderByDesc('created_at');

        if ($this->search) {
            $query->where(function ($q) {
                $q->where('tool_name', 'like', "%{$this->search}%")
                    ->orWhere('server_id', 'like', "%{$this->search}%")
                    ->orWhere('version', 'like', "%{$this->search}%")
                    ->orWhere('description', 'like', "%{$this->search}%");
            });
        }

        if ($this->server) {
            $query->forServer($this->server);
        }

        if ($this->status === 'latest') {
            $query->latest();
        } elseif ($this->status === 'deprecated') {
            $query->deprecated();
        } elseif ($this->status === 'sunset') {
            $query->sunset();
        } elseif ($this->status === 'active') {
            $query->active()->where('is_latest', false);
        }

        return $query->paginate($this->perPage);
    }

    #[Computed]
    public function servers(): Collection
    {
        return app(ToolVersionService::class)->getServersWithVersions();
    }

    #[Computed]
    public function stats(): array
    {
        return app(ToolVersionService::class)->getStats();
    }

    #[Computed]
    public function selectedVersion(): ?McpToolVersion
    {
        if (! $this->selectedVersionId) {
            return null;
        }

        return McpToolVersion::find($this->selectedVersionId);
    }

    #[Computed]
    public function versionHistory(): Collection
    {
        if (! $this->selectedVersion) {
            return collect();
        }

        return app(ToolVersionService::class)->getVersionHistory(
            $this->selectedVersion->server_id,
            $this->selectedVersion->tool_name
        );
    }

    #[Computed]
    public function schemaComparison(): ?array
    {
        if (! $this->compareFromId || ! $this->compareToId) {
            return null;
        }

        $from = McpToolVersion::find($this->compareFromId);
        $to = McpToolVersion::find($this->compareToId);

        if (! $from || ! $to) {
            return null;
        }

        return [
            'from' => $from,
            'to' => $to,
            'changes' => $from->compareSchemaWith($to),
        ];
    }

    // -------------------------------------------------------------------------
    // Actions
    // -------------------------------------------------------------------------

    public function viewVersion(int $id): void
    {
        $this->selectedVersionId = $id;
        $this->showVersionDetail = true;
    }

    public function closeVersionDetail(): void
    {
        $this->showVersionDetail = false;
        $this->selectedVersionId = null;
    }

    public function openCompareModal(int $fromId, int $toId): void
    {
        $this->compareFromId = $fromId;
        $this->compareToId = $toId;
        $this->showCompareModal = true;
    }

    public function closeCompareModal(): void
    {
        $this->showCompareModal = false;
        $this->compareFromId = null;
        $this->compareToId = null;
    }

    public function openDeprecateModal(int $versionId): void
    {
        $this->deprecateVersionId = $versionId;
        $this->deprecateSunsetDate = '';
        $this->showDeprecateModal = true;
    }

    public function closeDeprecateModal(): void
    {
        $this->showDeprecateModal = false;
        $this->deprecateVersionId = null;
        $this->deprecateSunsetDate = '';
    }

    public function deprecateVersion(): void
    {
        $version = McpToolVersion::find($this->deprecateVersionId);
        if (! $version) {
            return;
        }

        $sunsetAt = $this->deprecateSunsetDate
            ? Carbon::parse($this->deprecateSunsetDate)
            : null;

        app(ToolVersionService::class)->deprecateVersion(
            $version->server_id,
            $version->tool_name,
            $version->version,
            $sunsetAt
        );

        $this->closeDeprecateModal();
        $this->dispatch('version-deprecated');
    }

    public function markAsLatest(int $versionId): void
    {
        $version = McpToolVersion::find($versionId);
        if (! $version) {
            return;
        }

        $version->markAsLatest();
        $this->dispatch('version-marked-latest');
    }

    public function openRegisterModal(): void
    {
        $this->resetRegisterForm();
        $this->showRegisterModal = true;
    }

    public function closeRegisterModal(): void
    {
        $this->showRegisterModal = false;
        $this->resetRegisterForm();
    }

    public function registerVersion(): void
    {
        $this->validate([
            'registerServer' => 'required|string|max:64',
            'registerTool' => 'required|string|max:128',
            'registerVersion' => 'required|string|max:32|regex:/^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?$/',
            'registerDescription' => 'nullable|string|max:1000',
            'registerChangelog' => 'nullable|string|max:5000',
            'registerMigrationNotes' => 'nullable|string|max:5000',
            'registerInputSchema' => 'nullable|string',
        ]);

        $inputSchema = null;
        if ($this->registerInputSchema) {
            $inputSchema = json_decode($this->registerInputSchema, true);
            if (json_last_error() !== JSON_ERROR_NONE) {
                $this->addError('registerInputSchema', 'Invalid JSON');

                return;
            }
        }

        app(ToolVersionService::class)->registerVersion(
            serverId: $this->registerServer,
            toolName: $this->registerTool,
            version: $this->registerVersion,
            inputSchema: $inputSchema,
            description: $this->registerDescription ?: null,
            options: [
                'changelog' => $this->registerChangelog ?: null,
                'migration_notes' => $this->registerMigrationNotes ?: null,
                'mark_latest' => $this->registerMarkLatest,
            ]
        );

        $this->closeRegisterModal();
        $this->dispatch('version-registered');
    }

    public function clearFilters(): void
    {
        $this->search = '';
        $this->server = '';
        $this->status = '';
        $this->resetPage();
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    public function getStatusBadgeColor(string $status): string
    {
        return match ($status) {
            'latest' => 'green',
            'active' => 'zinc',
            'deprecated' => 'amber',
            'sunset' => 'red',
            default => 'zinc',
        };
    }

    public function formatSchema(array $schema): string
    {
        return json_encode($schema, JSON_PRETTY_PRINT);
    }

    private function resetRegisterForm(): void
    {
        $this->registerServer = '';
        $this->registerTool = '';
        $this->registerVersion = '';
        $this->registerDescription = '';
        $this->registerChangelog = '';
        $this->registerMigrationNotes = '';
        $this->registerInputSchema = '';
        $this->registerMarkLatest = false;
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    public function render()
    {
        return view('mcp::admin.tool-version-manager');
    }
}
