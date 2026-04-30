<?php

declare(strict_types=1);

namespace Core\Mcp\DTO;

/**
 * Tool Statistics Data Transfer Object.
 *
 * Represents aggregated statistics for a single MCP tool.
 */
readonly class ToolStats
{
    public function __construct(
        public string $toolName,
        public int $totalCalls,
        public int $errorCount,
        public float $errorRate,
        public float $avgDurationMs,
        public int $minDurationMs,
        public int $maxDurationMs,
    ) {}

    /**
     * Create from an array of data.
     */
    public static function fromArray(array $data): self
    {
        return new self(
            toolName: $data['tool_name'] ?? $data['toolName'] ?? '',
            totalCalls: (int) ($data['total_calls'] ?? $data['totalCalls'] ?? 0),
            errorCount: (int) ($data['error_count'] ?? $data['errorCount'] ?? 0),
            errorRate: (float) ($data['error_rate'] ?? $data['errorRate'] ?? 0.0),
            avgDurationMs: (float) ($data['avg_duration_ms'] ?? $data['avgDurationMs'] ?? 0.0),
            minDurationMs: (int) ($data['min_duration_ms'] ?? $data['minDurationMs'] ?? 0),
            maxDurationMs: (int) ($data['max_duration_ms'] ?? $data['maxDurationMs'] ?? 0),
        );
    }

    /**
     * Convert to array.
     */
    public function toArray(): array
    {
        return [
            'tool_name' => $this->toolName,
            'total_calls' => $this->totalCalls,
            'error_count' => $this->errorCount,
            'error_rate' => $this->errorRate,
            'avg_duration_ms' => $this->avgDurationMs,
            'min_duration_ms' => $this->minDurationMs,
            'max_duration_ms' => $this->maxDurationMs,
        ];
    }

    /**
     * Get success rate as percentage.
     */
    public function getSuccessRate(): float
    {
        return 100.0 - $this->errorRate;
    }

    /**
     * Get average duration formatted for display.
     */
    public function getAvgDurationForHumans(): string
    {
        if ($this->avgDurationMs === 0.0) {
            return '-';
        }

        if ($this->avgDurationMs < 1000) {
            return round($this->avgDurationMs).'ms';
        }

        return round($this->avgDurationMs / 1000, 2).'s';
    }

    /**
     * Check if the tool has a high error rate (above threshold).
     */
    public function hasHighErrorRate(float $threshold = 10.0): bool
    {
        return $this->errorRate > $threshold;
    }

    /**
     * Check if the tool has slow response times (above threshold in ms).
     */
    public function isSlowResponding(int $thresholdMs = 5000): bool
    {
        return $this->avgDurationMs > $thresholdMs;
    }
}
