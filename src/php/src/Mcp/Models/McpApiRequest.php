<?php

declare(strict_types=1);

namespace Core\Mcp\Models;

use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Str;

/**
 * MCP API Request - logs full request/response for debugging and replay.
 *
 * @property int $id
 * @property string $request_id
 * @property int|null $workspace_id
 * @property int|null $api_key_id
 * @property string $method
 * @property string $path
 * @property array $headers
 * @property array $request_body
 * @property int $response_status
 * @property array|null $response_body
 * @property int $duration_ms
 * @property string|null $server_id
 * @property string|null $tool_name
 * @property string|null $error_message
 * @property string|null $ip_address
 * @property \Carbon\Carbon|null $created_at
 * @property \Carbon\Carbon|null $updated_at
 */
class McpApiRequest extends Model
{
    protected $fillable = [
        'request_id',
        'workspace_id',
        'api_key_id',
        'method',
        'path',
        'headers',
        'request_body',
        'response_status',
        'response_body',
        'duration_ms',
        'server_id',
        'tool_name',
        'error_message',
        'ip_address',
    ];

    protected $casts = [
        'headers' => 'array',
        'request_body' => 'array',
        'response_body' => 'array',
        'duration_ms' => 'integer',
        'response_status' => 'integer',
    ];

    /**
     * Log an API request.
     */
    public static function log(
        string $method,
        string $path,
        array $requestBody,
        int $responseStatus,
        ?array $responseBody = null,
        int $durationMs = 0,
        ?int $workspaceId = null,
        ?int $apiKeyId = null,
        ?string $serverId = null,
        ?string $toolName = null,
        ?string $errorMessage = null,
        ?string $ipAddress = null,
        array $headers = []
    ): self {
        // Sanitise headers - remove sensitive info
        $sanitisedHeaders = collect($headers)
            ->except(['authorization', 'x-api-key', 'cookie'])
            ->toArray();

        return static::create([
            'request_id' => 'req_'.Str::random(20),
            'workspace_id' => $workspaceId,
            'api_key_id' => $apiKeyId,
            'method' => $method,
            'path' => $path,
            'headers' => $sanitisedHeaders,
            'request_body' => $requestBody,
            'response_status' => $responseStatus,
            'response_body' => $responseBody,
            'duration_ms' => $durationMs,
            'server_id' => $serverId,
            'tool_name' => $toolName,
            'error_message' => $errorMessage,
            'ip_address' => $ipAddress,
        ]);
    }

    /**
     * Generate curl command to replay this request.
     */
    public function toCurl(string $apiKey = 'YOUR_API_KEY'): string
    {
        $url = config('app.url').'/api/v1/mcp'.$this->path;

        $curl = "curl -X {$this->method} \"{$url}\"";
        $curl .= " \\\n  -H \"Authorization: Bearer {$apiKey}\"";
        $curl .= " \\\n  -H \"Content-Type: application/json\"";

        if (! empty($this->request_body)) {
            $curl .= " \\\n  -d '".json_encode($this->request_body)."'";
        }

        return $curl;
    }

    /**
     * Get duration formatted for humans.
     */
    public function getDurationForHumansAttribute(): string
    {
        if ($this->duration_ms < 1000) {
            return $this->duration_ms.'ms';
        }

        return round($this->duration_ms / 1000, 2).'s';
    }

    /**
     * Check if request was successful.
     */
    public function isSuccessful(): bool
    {
        return $this->response_status >= 200 && $this->response_status < 300;
    }

    // Relationships
    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    // Scopes
    public function scopeForWorkspace(Builder $query, int $workspaceId): Builder
    {
        return $query->where('workspace_id', $workspaceId);
    }

    public function scopeForServer(Builder $query, string $serverId): Builder
    {
        return $query->where('server_id', $serverId);
    }

    public function scopeForTool(Builder $query, string $toolName): Builder
    {
        return $query->where('tool_name', $toolName);
    }

    public function scopeFailed(Builder $query): Builder
    {
        return $query->where('response_status', '>=', 400);
    }

    public function scopeSuccessful(Builder $query): Builder
    {
        return $query->whereBetween('response_status', [200, 299]);
    }

    public function scopeRecent(Builder $query, int $hours = 24): Builder
    {
        return $query->where('created_at', '>=', now()->subHours($hours));
    }
}
