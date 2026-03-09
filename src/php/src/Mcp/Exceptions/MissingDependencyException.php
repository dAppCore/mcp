<?php

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use Core\Mcp\Dependencies\ToolDependency;
use RuntimeException;

/**
 * Exception thrown when tool dependencies are not met.
 *
 * Provides detailed information about what's missing and how to resolve it.
 */
class MissingDependencyException extends RuntimeException
{
    /**
     * @param  string  $toolName  The tool that has unmet dependencies
     * @param  array<ToolDependency>  $missingDependencies  List of unmet dependencies
     * @param  array<string>  $suggestedOrder  Suggested tools to call first
     */
    public function __construct(
        public readonly string $toolName,
        public readonly array $missingDependencies,
        public readonly array $suggestedOrder = [],
    ) {
        $message = $this->buildMessage();
        parent::__construct($message);
    }

    /**
     * Build a user-friendly error message.
     */
    protected function buildMessage(): string
    {
        $missing = array_map(
            fn (ToolDependency $dep) => "- {$dep->description}",
            $this->missingDependencies
        );

        $message = "Cannot execute '{$this->toolName}': prerequisites not met.\n\n";
        $message .= "Missing:\n".implode("\n", $missing);

        if (! empty($this->suggestedOrder)) {
            $message .= "\n\nSuggested order:\n";
            foreach ($this->suggestedOrder as $i => $tool) {
                $message .= sprintf("  %d. %s\n", $i + 1, $tool);
            }
        }

        return $message;
    }

    /**
     * Get a structured error response for API output.
     */
    public function toApiResponse(): array
    {
        return [
            'error' => 'dependency_not_met',
            'message' => "Cannot execute '{$this->toolName}': prerequisites not met",
            'tool' => $this->toolName,
            'missing_dependencies' => array_map(
                fn (ToolDependency $dep) => $dep->toArray(),
                $this->missingDependencies
            ),
            'suggested_order' => $this->suggestedOrder,
            'help' => $this->getHelpText(),
        ];
    }

    /**
     * Get help text explaining how to resolve the issue.
     */
    protected function getHelpText(): string
    {
        if (empty($this->suggestedOrder)) {
            return 'Ensure all required dependencies are satisfied before calling this tool.';
        }

        return sprintf(
            'Call these tools in order before attempting %s: %s',
            $this->toolName,
            implode(' -> ', $this->suggestedOrder)
        );
    }
}
