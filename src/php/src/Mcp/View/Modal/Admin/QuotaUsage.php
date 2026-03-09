<?php

declare(strict_types=1);

namespace Core\Mcp\View\Modal\Admin;

use Core\Mcp\Services\McpQuotaService;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Collection;
use Livewire\Component;

/**
 * MCP Quota Usage Dashboard.
 *
 * Displays current workspace MCP usage against quota limits
 * and historical usage trends.
 */
class QuotaUsage extends Component
{
    public ?int $workspaceId = null;

    public array $currentUsage = [];

    public array $quotaLimits = [];

    public array $remaining = [];

    public Collection $usageHistory;

    public function mount(?int $workspaceId = null): void
    {
        $this->workspaceId = $workspaceId ?? auth()->user()?->defaultHostWorkspace()?->id;
        $this->usageHistory = collect();
        $this->loadQuotaData();
    }

    public function loadQuotaData(): void
    {
        if (! $this->workspaceId) {
            return;
        }

        $quotaService = app(McpQuotaService::class);
        $workspace = Workspace::find($this->workspaceId);

        if (! $workspace) {
            return;
        }

        $this->currentUsage = $quotaService->getCurrentUsage($workspace);
        $this->quotaLimits = $quotaService->getQuotaLimits($workspace);
        $this->remaining = $quotaService->getRemainingQuota($workspace);
        $this->usageHistory = $quotaService->getUsageHistory($workspace, 6);
    }

    public function getToolCallsPercentageProperty(): float
    {
        if ($this->quotaLimits['tool_calls_unlimited'] ?? false) {
            return 0;
        }

        $limit = $this->quotaLimits['tool_calls_limit'] ?? 0;
        if ($limit === 0) {
            return 0;
        }

        return min(100, round(($this->currentUsage['tool_calls_count'] ?? 0) / $limit * 100, 1));
    }

    public function getTokensPercentageProperty(): float
    {
        if ($this->quotaLimits['tokens_unlimited'] ?? false) {
            return 0;
        }

        $limit = $this->quotaLimits['tokens_limit'] ?? 0;
        if ($limit === 0) {
            return 0;
        }

        return min(100, round(($this->currentUsage['total_tokens'] ?? 0) / $limit * 100, 1));
    }

    public function getResetDateProperty(): string
    {
        return now()->endOfMonth()->format('j F Y');
    }

    public function render()
    {
        return view('mcp::admin.quota-usage');
    }
}
