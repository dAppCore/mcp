<?php

declare(strict_types=1);

namespace Core\Website\Mcp\View\Modal;

use Livewire\Component;
use Core\Mod\Api\Models\ApiKey;
use Mod\Tenant\Models\Workspace;

/**
 * MCP API Key Manager.
 *
 * Allows workspace owners to create and manage API keys
 * for accessing MCP servers via HTTP API.
 */
class ApiKeyManager extends Component
{
    public Workspace $workspace;

    // Create form state
    public bool $showCreateModal = false;

    public string $newKeyName = '';

    public array $newKeyScopes = ['read', 'write'];

    public string $newKeyExpiry = 'never';

    // Show new key (only visible once after creation)
    public ?string $newPlainKey = null;

    public bool $showNewKeyModal = false;

    public function mount(Workspace $workspace): void
    {
        $this->workspace = $workspace;
    }

    public function openCreateModal(): void
    {
        $this->showCreateModal = true;
        $this->newKeyName = '';
        $this->newKeyScopes = ['read', 'write'];
        $this->newKeyExpiry = 'never';
    }

    public function closeCreateModal(): void
    {
        $this->showCreateModal = false;
    }

    public function createKey(): void
    {
        $this->validate([
            'newKeyName' => 'required|string|max:100',
        ]);

        $expiresAt = match ($this->newKeyExpiry) {
            '30days' => now()->addDays(30),
            '90days' => now()->addDays(90),
            '1year' => now()->addYear(),
            default => null,
        };

        $result = ApiKey::generate(
            workspaceId: $this->workspace->id,
            userId: auth()->id(),
            name: $this->newKeyName,
            scopes: $this->newKeyScopes,
            expiresAt: $expiresAt,
        );

        $this->newPlainKey = $result['plain_key'];
        $this->showCreateModal = false;
        $this->showNewKeyModal = true;

        session()->flash('message', 'API key created successfully.');
    }

    public function closeNewKeyModal(): void
    {
        $this->newPlainKey = null;
        $this->showNewKeyModal = false;
    }

    public function revokeKey(int $keyId): void
    {
        $key = $this->workspace->apiKeys()->findOrFail($keyId);
        $key->revoke();

        session()->flash('message', 'API key revoked.');
    }

    public function toggleScope(string $scope): void
    {
        if (in_array($scope, $this->newKeyScopes)) {
            $this->newKeyScopes = array_values(array_diff($this->newKeyScopes, [$scope]));
        } else {
            $this->newKeyScopes[] = $scope;
        }
    }

    public function render()
    {
        return view('mcp::web.api-key-manager', [
            'keys' => $this->workspace->apiKeys()->orderByDesc('created_at')->get(),
        ]);
    }
}
