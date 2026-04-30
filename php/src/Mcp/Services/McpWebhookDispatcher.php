<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Core\Mod\Api\Models\WebhookDelivery;
use Core\Mod\Api\Models\WebhookEndpoint;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

/**
 * Dispatches webhooks for MCP tool execution events.
 */
class McpWebhookDispatcher
{
    /**
     * Dispatch tool.executed event to all subscribed endpoints.
     */
    public function dispatchToolExecuted(
        int $workspaceId,
        string $serverId,
        string $toolName,
        array $arguments,
        bool $success,
        int $durationMs,
        ?string $errorMessage = null
    ): void {
        $eventType = 'mcp.tool.executed';

        $endpoints = WebhookEndpoint::query()
            ->forWorkspace($workspaceId)
            ->active()
            ->forEvent($eventType)
            ->get();

        if ($endpoints->isEmpty()) {
            return;
        }

        $payload = [
            'event' => $eventType,
            'timestamp' => now()->toIso8601String(),
            'data' => [
                'server_id' => $serverId,
                'tool_name' => $toolName,
                'arguments' => $arguments,
                'success' => $success,
                'duration_ms' => $durationMs,
                'error' => $errorMessage,
            ],
        ];

        foreach ($endpoints as $endpoint) {
            $this->deliverWebhook($endpoint, $payload);
        }
    }

    /**
     * Deliver a webhook to an endpoint.
     */
    protected function deliverWebhook(WebhookEndpoint $endpoint, array $payload): void
    {
        $payloadJson = json_encode($payload);
        $signature = $endpoint->generateSignature($payloadJson);

        $startTime = microtime(true);

        try {
            $response = Http::timeout(10)
                ->withHeaders([
                    'Content-Type' => 'application/json',
                    'X-Webhook-Signature' => $signature,
                    'X-Webhook-Event' => $payload['event'],
                    'X-Webhook-Timestamp' => $payload['timestamp'],
                ])
                ->withBody($payloadJson, 'application/json')
                ->post($endpoint->url);

            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            // Record delivery
            WebhookDelivery::create([
                'webhook_endpoint_id' => $endpoint->id,
                'event_id' => 'evt_'.uniqid(),
                'event_type' => $payload['event'],
                'payload' => $payload,
                'response_code' => $response->status(),
                'response_body' => substr($response->body(), 0, 1000),
                'status' => $response->successful() ? 'success' : 'failed',
                'attempt' => 1,
                'delivered_at' => $response->successful() ? now() : null,
            ]);

            if ($response->successful()) {
                $endpoint->recordSuccess();
            } else {
                $endpoint->recordFailure();
                Log::warning('MCP Webhook delivery failed', [
                    'endpoint_id' => $endpoint->id,
                    'url' => $endpoint->url,
                    'status' => $response->status(),
                ]);
            }
        } catch (\Throwable $e) {
            $durationMs = (int) ((microtime(true) - $startTime) * 1000);

            WebhookDelivery::create([
                'webhook_endpoint_id' => $endpoint->id,
                'event_id' => 'evt_'.uniqid(),
                'event_type' => $payload['event'],
                'payload' => $payload,
                'response_code' => 0,
                'response_body' => $e->getMessage(),
                'status' => 'failed',
                'attempt' => 1,
            ]);

            $endpoint->recordFailure();

            Log::error('MCP Webhook delivery error', [
                'endpoint_id' => $endpoint->id,
                'url' => $endpoint->url,
                'error' => $e->getMessage(),
            ]);
        }
    }
}
