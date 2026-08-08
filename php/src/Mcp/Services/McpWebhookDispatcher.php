<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Http\Client\Response;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;
use ReflectionMethod;

// Not final: endpointModelClass() / deliveryModelClass() are protected seams the
// suite substitutes to dispatch without the Eloquent models.
// `final` contradicted that and made McpWebhookDispatcherTest a fatal error.
class McpWebhookDispatcher
{
    public function dispatchToolExecuted(
        int $workspaceId,
        string $serverId,
        string $toolName,
        array $arguments,
        bool $success,
        int $durationMs,
        ?string $errorMessage = null
    ): void {
        $endpointClass = $this->endpointModelClass();

        if ($endpointClass === null) {
            return;
        }

        $eventType = 'mcp.tool.executed';
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

        $query = $endpointClass::query();
        $model = $query->getModel();

        if (method_exists($model, 'scopeForWorkspace')) {
            $query->forWorkspace($workspaceId);
        } else {
            $query->where('workspace_id', $workspaceId);
        }

        if (method_exists($model, 'scopeActive')) {
            $query->active();
        } else {
            $query->where('active', true);
        }

        if (method_exists($model, 'scopeForEvent')) {
            $query->forEvent($eventType);
        } else {
            $query->where(function ($inner) use ($eventType): void {
                $inner->whereJsonContains('events', $eventType)
                    ->orWhereJsonContains('events', '*');
            });
        }

        foreach ($query->get() as $endpoint) {
            $this->deliverWebhook($endpoint, $payload);
        }
    }

    protected function deliverWebhook(object $endpoint, array $payload): void
    {
        $timestamp = (string) ($payload['timestamp'] ?? now()->toIso8601String());
        $payloadJson = json_encode($payload, JSON_UNESCAPED_SLASHES);
        $signature = $this->generateSignature($endpoint, $payloadJson, $timestamp);

        try {
            $response = $this->sendWebhook($endpoint, $payloadJson, [
                'Content-Type' => 'application/json',
                'X-Webhook-Signature' => $signature,
                'X-Webhook-Event' => (string) $payload['event'],
                'X-Webhook-Timestamp' => $timestamp,
            ]);

            $this->recordDelivery($endpoint, $payload, $response->status(), $response->body(), $response->successful());

            if ($response->successful() && method_exists($endpoint, 'recordSuccess')) {
                $endpoint->recordSuccess();
            }

            if (! $response->successful() && method_exists($endpoint, 'recordFailure')) {
                $endpoint->recordFailure();
            }
        } catch (\Throwable $throwable) {
            $this->recordDelivery($endpoint, $payload, 0, $throwable->getMessage(), false);

            if (method_exists($endpoint, 'recordFailure')) {
                $endpoint->recordFailure();
            }
        }
    }

    protected function sendWebhook(object $endpoint, string $payloadJson, array $headers): Response
    {
        return Http::timeout(10)
            ->withHeaders($headers)
            ->withBody($payloadJson, 'application/json')
            ->post((string) $endpoint->url);
    }

    protected function recordDelivery(object $endpoint, array $payload, int $responseCode, string $responseBody, bool $successful): void
    {
        $deliveryClass = $this->deliveryModelClass();

        if ($deliveryClass === null) {
            return;
        }

        $deliveryClass::create([
            'webhook_endpoint_id' => $endpoint->id,
            'event_id' => 'evt_'.Str::random(24),
            'event_type' => (string) $payload['event'],
            'payload' => $payload,
            'response_code' => $responseCode,
            'response_body' => mb_substr($responseBody, 0, 1000),
            'status' => $successful ? 'success' : 'failed',
            'attempt' => 1,
            'delivered_at' => $successful ? now() : null,
        ]);
    }

    protected function generateSignature(object $endpoint, string $payloadJson, string $timestamp): string
    {
        if (! method_exists($endpoint, 'generateSignature')) {
            return '';
        }

        $method = new ReflectionMethod($endpoint, 'generateSignature');

        return match (true) {
            $method->getNumberOfRequiredParameters() <= 1 => (string) $endpoint->generateSignature($payloadJson),
            default => (string) $endpoint->generateSignature($payloadJson, strtotime($timestamp) ?: time()),
        };
    }

    protected function endpointModelClass(): ?string
    {
        foreach ([
            'Core\\Api\\Models\\WebhookEndpoint',
            'Core\\Mod\\Api\\Models\\WebhookEndpoint',
        ] as $class) {
            if (class_exists($class)) {
                return $class;
            }
        }

        return null;
    }

    protected function deliveryModelClass(): ?string
    {
        foreach ([
            'Core\\Api\\Models\\WebhookDelivery',
            'Core\\Mod\\Api\\Models\\WebhookDelivery',
        ] as $class) {
            if (class_exists($class)) {
                return $class;
            }
        }

        return null;
    }
}
