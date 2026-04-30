<?php

declare(strict_types=1);

namespace Core\Mcp\Dependencies;

/**
 * Represents a single tool dependency.
 *
 * Defines what must be satisfied before a tool can execute.
 */
class ToolDependency
{
    /**
     * Create a new tool dependency.
     *
     * @param  DependencyType  $type  The type of dependency
     * @param  string  $key  The identifier (tool name, state key, context key, etc.)
     * @param  string|null  $description  Human-readable description for error messages
     * @param  bool  $optional  If true, this is a soft dependency (warning, not error)
     * @param  array  $metadata  Additional metadata for custom validation
     */
    public function __construct(
        public readonly DependencyType $type,
        public readonly string $key,
        public readonly ?string $description = null,
        public readonly bool $optional = false,
        public readonly array $metadata = [],
    ) {}

    /**
     * Create a tool_called dependency.
     */
    public static function toolCalled(string $toolName, ?string $description = null): self
    {
        return new self(
            type: DependencyType::TOOL_CALLED,
            key: $toolName,
            description: $description ?? "Tool '{$toolName}' must be called first",
        );
    }

    /**
     * Create a session_state dependency.
     */
    public static function sessionState(string $stateKey, ?string $description = null): self
    {
        return new self(
            type: DependencyType::SESSION_STATE,
            key: $stateKey,
            description: $description ?? "Session state '{$stateKey}' is required",
        );
    }

    /**
     * Create a context_exists dependency.
     */
    public static function contextExists(string $contextKey, ?string $description = null): self
    {
        return new self(
            type: DependencyType::CONTEXT_EXISTS,
            key: $contextKey,
            description: $description ?? "Context '{$contextKey}' is required",
        );
    }

    /**
     * Create an entity_exists dependency.
     */
    public static function entityExists(string $entityType, ?string $description = null, array $metadata = []): self
    {
        return new self(
            type: DependencyType::ENTITY_EXISTS,
            key: $entityType,
            description: $description ?? "Entity '{$entityType}' must exist",
            metadata: $metadata,
        );
    }

    /**
     * Create a custom dependency with callback metadata.
     */
    public static function custom(string $name, ?string $description = null, array $metadata = []): self
    {
        return new self(
            type: DependencyType::CUSTOM,
            key: $name,
            description: $description,
            metadata: $metadata,
        );
    }

    /**
     * Mark this dependency as optional (soft dependency).
     */
    public function asOptional(): self
    {
        return new self(
            type: $this->type,
            key: $this->key,
            description: $this->description,
            optional: true,
            metadata: $this->metadata,
        );
    }

    /**
     * Convert to array representation.
     */
    public function toArray(): array
    {
        return [
            'type' => $this->type->value,
            'key' => $this->key,
            'description' => $this->description,
            'optional' => $this->optional,
            'metadata' => $this->metadata,
        ];
    }

    /**
     * Create from array representation.
     */
    public static function fromArray(array $data): self
    {
        return new self(
            type: DependencyType::from($data['type']),
            key: $data['key'],
            description: $data['description'] ?? null,
            optional: $data['optional'] ?? false,
            metadata: $data['metadata'] ?? [],
        );
    }
}
