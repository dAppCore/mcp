<?php

declare(strict_types=1);

namespace Core\Mcp\Tools\Commerce;

use Core\Mod\Commerce\Models\Subscription;
use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;
use Mod\Mcp\Tools\Concerns\RequiresWorkspaceContext;

/**
 * Get billing status for the authenticated workspace.
 *
 * SECURITY: This tool uses authenticated workspace context, not user-supplied
 * workspace_id parameters, to prevent cross-tenant data access.
 */
class GetBillingStatus extends Tool
{
    use RequiresWorkspaceContext;

    protected string $description = 'Get billing status for your workspace including subscription, current plan, and billing period';

    public function handle(Request $request): Response
    {
        // Get workspace from authenticated context (not from request parameters)
        $workspace = $this->getWorkspace();
        $workspaceId = $workspace->id;

        // Get active subscription
        $subscription = Subscription::with('workspacePackage.package')
            ->where('workspace_id', $workspaceId)
            ->whereIn('status', ['active', 'trialing', 'past_due'])
            ->first();

        // Get workspace packages
        $packages = $workspace->workspacePackages()
            ->with('package')
            ->whereIn('status', ['active', 'trial'])
            ->get();

        $status = [
            'workspace' => [
                'id' => $workspace->id,
                'name' => $workspace->name,
            ],
            'subscription' => $subscription ? [
                'id' => $subscription->id,
                'status' => $subscription->status,
                'gateway' => $subscription->gateway,
                'billing_cycle' => $subscription->billing_cycle,
                'current_period_start' => $subscription->current_period_start?->toIso8601String(),
                'current_period_end' => $subscription->current_period_end?->toIso8601String(),
                'days_until_renewal' => $subscription->daysUntilRenewal(),
                'cancel_at_period_end' => $subscription->cancel_at_period_end,
                'on_trial' => $subscription->onTrial(),
                'trial_ends_at' => $subscription->trial_ends_at?->toIso8601String(),
            ] : null,
            'packages' => $packages->map(fn ($wp) => [
                'code' => $wp->package?->code,
                'name' => $wp->package?->name,
                'status' => $wp->status,
                'expires_at' => $wp->expires_at?->toIso8601String(),
            ])->values()->all(),
        ];

        return Response::text(json_encode($status, JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        // No parameters needed - workspace comes from authentication context
        return [];
    }
}
