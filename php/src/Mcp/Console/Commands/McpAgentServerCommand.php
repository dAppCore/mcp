<?php

declare(strict_types=1);

namespace Core\Mcp\Console\Commands;

use Illuminate\Console\Command;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Str;
use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Models\AgentWorkspaceState;
use Core\Mod\Agentic\Services\PlanTemplateService;
use Core\Mod\Content\Jobs\GenerateContentJob;
use Core\Mod\Content\Models\AIUsage;
use Core\Mod\Content\Models\ContentBrief;
use Core\Mod\Content\Services\AIGatewayService;
use Core\Mcp\Models\McpToolCall;
use Throwable;

/**
 * MCP Agent Server for Host Hub.
 *
 * Provides an MCP (Model Context Protocol) server for AI agent task management.
 * Enables multi-agent handoff, plan tracking, session state, and tool call logging.
 *
 * Run via: php artisan mcp:agent-server
 */
class McpAgentServerCommand extends Command
{
    protected $signature = 'mcp:agent-server';

    protected $description = 'Run the MCP Agent Server for AI task management';

    protected array $tools = [];

    protected array $resources = [];

    protected ?string $currentSessionId = null;

    public function handle(): int
    {
        $this->registerTools();
        $this->registerResources();

        // Run MCP server loop
        while (($line = fgets(STDIN)) !== false) {
            $line = trim($line);
            if (empty($line)) {
                continue;
            }

            try {
                $request = json_decode($line, true, 512, JSON_THROW_ON_ERROR);
                $response = $this->handleRequest($request);

                if ($response !== null) {
                    $this->sendResponse($response);
                }
            } catch (Throwable $e) {
                Log::error('MCP Agent Server error', [
                    'error' => $e->getMessage(),
                    'line' => $line,
                ]);

                $this->sendResponse([
                    'jsonrpc' => '2.0',
                    'id' => null,
                    'error' => [
                        'code' => -32700,
                        'message' => 'Parse error: '.$e->getMessage(),
                    ],
                ]);
            }
        }

        return 0;
    }

    protected function registerTools(): void
    {
        // Plan management tools
        $this->tools['plan_list'] = [
            'description' => 'List all work plans with their current status and progress',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'status' => [
                        'type' => 'string',
                        'description' => 'Filter by status (draft, active, paused, completed, archived)',
                        'enum' => ['draft', 'active', 'paused', 'completed', 'archived'],
                    ],
                    'include_archived' => [
                        'type' => 'boolean',
                        'description' => 'Include archived plans (default: false)',
                    ],
                ],
            ],
            'handler' => 'toolPlanList',
        ];

        $this->tools['plan_create'] = [
            'description' => 'Create a new work plan with phases and tasks',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'title' => [
                        'type' => 'string',
                        'description' => 'Plan title',
                    ],
                    'slug' => [
                        'type' => 'string',
                        'description' => 'URL-friendly identifier (auto-generated if not provided)',
                    ],
                    'description' => [
                        'type' => 'string',
                        'description' => 'Plan description',
                    ],
                    'context' => [
                        'type' => 'object',
                        'description' => 'Additional context (related files, dependencies, etc.)',
                    ],
                    'phases' => [
                        'type' => 'array',
                        'description' => 'Array of phase definitions with name, description, and tasks',
                        'items' => [
                            'type' => 'object',
                            'properties' => [
                                'name' => ['type' => 'string'],
                                'description' => ['type' => 'string'],
                                'tasks' => [
                                    'type' => 'array',
                                    'items' => ['type' => 'string'],
                                ],
                            ],
                        ],
                    ],
                ],
                'required' => ['title'],
            ],
            'handler' => 'toolPlanCreate',
        ];

        $this->tools['plan_get'] = [
            'description' => 'Get detailed information about a specific plan',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'format' => [
                        'type' => 'string',
                        'description' => 'Output format: json or markdown',
                        'enum' => ['json', 'markdown'],
                    ],
                ],
                'required' => ['slug'],
            ],
            'handler' => 'toolPlanGet',
        ];

        $this->tools['plan_update_status'] = [
            'description' => 'Update the status of a plan',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'status' => [
                        'type' => 'string',
                        'description' => 'New status',
                        'enum' => ['draft', 'active', 'paused', 'completed'],
                    ],
                ],
                'required' => ['slug', 'status'],
            ],
            'handler' => 'toolPlanUpdateStatus',
        ];

        $this->tools['plan_archive'] = [
            'description' => 'Archive a completed or abandoned plan',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'reason' => [
                        'type' => 'string',
                        'description' => 'Reason for archiving',
                    ],
                ],
                'required' => ['slug'],
            ],
            'handler' => 'toolPlanArchive',
        ];

        // Phase tools
        $this->tools['phase_get'] = [
            'description' => 'Get details of a specific phase within a plan',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'phase' => [
                        'type' => 'string',
                        'description' => 'Phase identifier (number or name)',
                    ],
                ],
                'required' => ['plan_slug', 'phase'],
            ],
            'handler' => 'toolPhaseGet',
        ];

        $this->tools['phase_update_status'] = [
            'description' => 'Update the status of a phase',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'phase' => [
                        'type' => 'string',
                        'description' => 'Phase identifier (number or name)',
                    ],
                    'status' => [
                        'type' => 'string',
                        'description' => 'New status',
                        'enum' => ['pending', 'in_progress', 'completed', 'blocked', 'skipped'],
                    ],
                    'notes' => [
                        'type' => 'string',
                        'description' => 'Optional notes about the status change',
                    ],
                ],
                'required' => ['plan_slug', 'phase', 'status'],
            ],
            'handler' => 'toolPhaseUpdateStatus',
        ];

        $this->tools['phase_add_checkpoint'] = [
            'description' => 'Add a checkpoint note to a phase',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'phase' => [
                        'type' => 'string',
                        'description' => 'Phase identifier (number or name)',
                    ],
                    'note' => [
                        'type' => 'string',
                        'description' => 'Checkpoint note',
                    ],
                    'context' => [
                        'type' => 'object',
                        'description' => 'Additional context data',
                    ],
                ],
                'required' => ['plan_slug', 'phase', 'note'],
            ],
            'handler' => 'toolPhaseAddCheckpoint',
        ];

        // Task tools
        $this->tools['task_toggle'] = [
            'description' => 'Toggle a task completion status',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'phase' => [
                        'type' => 'string',
                        'description' => 'Phase identifier (number or name)',
                    ],
                    'task_index' => [
                        'type' => 'integer',
                        'description' => 'Task index (0-based)',
                    ],
                ],
                'required' => ['plan_slug', 'phase', 'task_index'],
            ],
            'handler' => 'toolTaskToggle',
        ];

        $this->tools['task_update'] = [
            'description' => 'Update task details (status, notes)',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'phase' => [
                        'type' => 'string',
                        'description' => 'Phase identifier (number or name)',
                    ],
                    'task_index' => [
                        'type' => 'integer',
                        'description' => 'Task index (0-based)',
                    ],
                    'status' => [
                        'type' => 'string',
                        'description' => 'New status',
                        'enum' => ['pending', 'in_progress', 'completed', 'blocked', 'skipped'],
                    ],
                    'notes' => [
                        'type' => 'string',
                        'description' => 'Task notes',
                    ],
                ],
                'required' => ['plan_slug', 'phase', 'task_index'],
            ],
            'handler' => 'toolTaskUpdate',
        ];

        // Session tools (for multi-agent handoff)
        $this->tools['session_start'] = [
            'description' => 'Start a new agent session for a plan',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'agent_type' => [
                        'type' => 'string',
                        'description' => 'Type of agent (e.g., opus, sonnet, haiku)',
                    ],
                    'context' => [
                        'type' => 'object',
                        'description' => 'Initial session context',
                    ],
                ],
                'required' => ['agent_type'],
            ],
            'handler' => 'toolSessionStart',
        ];

        $this->tools['session_log'] = [
            'description' => 'Log an entry in the current session',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'message' => [
                        'type' => 'string',
                        'description' => 'Log message',
                    ],
                    'type' => [
                        'type' => 'string',
                        'description' => 'Log type',
                        'enum' => ['info', 'progress', 'decision', 'error', 'checkpoint'],
                    ],
                    'data' => [
                        'type' => 'object',
                        'description' => 'Additional data to log',
                    ],
                ],
                'required' => ['message'],
            ],
            'handler' => 'toolSessionLog',
        ];

        $this->tools['session_artifact'] = [
            'description' => 'Record an artifact created/modified during the session',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'path' => [
                        'type' => 'string',
                        'description' => 'File or resource path',
                    ],
                    'action' => [
                        'type' => 'string',
                        'description' => 'Action performed',
                        'enum' => ['created', 'modified', 'deleted', 'reviewed'],
                    ],
                    'description' => [
                        'type' => 'string',
                        'description' => 'Description of changes',
                    ],
                ],
                'required' => ['path', 'action'],
            ],
            'handler' => 'toolSessionArtifact',
        ];

        $this->tools['session_handoff'] = [
            'description' => 'Prepare session for handoff to another agent',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'summary' => [
                        'type' => 'string',
                        'description' => 'Summary of work done',
                    ],
                    'next_steps' => [
                        'type' => 'array',
                        'description' => 'Recommended next steps',
                        'items' => ['type' => 'string'],
                    ],
                    'blockers' => [
                        'type' => 'array',
                        'description' => 'Any blockers encountered',
                        'items' => ['type' => 'string'],
                    ],
                    'context_for_next' => [
                        'type' => 'object',
                        'description' => 'Context to pass to next agent',
                    ],
                ],
                'required' => ['summary'],
            ],
            'handler' => 'toolSessionHandoff',
        ];

        $this->tools['session_end'] = [
            'description' => 'End the current session',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'status' => [
                        'type' => 'string',
                        'description' => 'Final session status',
                        'enum' => ['completed', 'handed_off', 'paused', 'failed'],
                    ],
                    'summary' => [
                        'type' => 'string',
                        'description' => 'Final summary',
                    ],
                ],
                'required' => ['status'],
            ],
            'handler' => 'toolSessionEnd',
        ];

        // State tools (persistent workspace state)
        $this->tools['state_get'] = [
            'description' => 'Get a workspace state value',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'key' => [
                        'type' => 'string',
                        'description' => 'State key',
                    ],
                ],
                'required' => ['plan_slug', 'key'],
            ],
            'handler' => 'toolStateGet',
        ];

        $this->tools['state_set'] = [
            'description' => 'Set a workspace state value',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'key' => [
                        'type' => 'string',
                        'description' => 'State key',
                    ],
                    'value' => [
                        'type' => ['string', 'number', 'boolean', 'object', 'array'],
                        'description' => 'State value',
                    ],
                    'category' => [
                        'type' => 'string',
                        'description' => 'State category for organisation',
                    ],
                ],
                'required' => ['plan_slug', 'key', 'value'],
            ],
            'handler' => 'toolStateSet',
        ];

        $this->tools['state_list'] = [
            'description' => 'List all state values for a plan',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug identifier',
                    ],
                    'category' => [
                        'type' => 'string',
                        'description' => 'Filter by category',
                    ],
                ],
                'required' => ['plan_slug'],
            ],
            'handler' => 'toolStateList',
        ];

        // Template tools
        $this->tools['template_list'] = [
            'description' => 'List available plan templates',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'category' => [
                        'type' => 'string',
                        'description' => 'Filter by category',
                    ],
                ],
            ],
            'handler' => 'toolTemplateList',
        ];

        $this->tools['template_preview'] = [
            'description' => 'Preview a template with variables',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'template' => [
                        'type' => 'string',
                        'description' => 'Template name/slug',
                    ],
                    'variables' => [
                        'type' => 'object',
                        'description' => 'Variable values for the template',
                    ],
                ],
                'required' => ['template'],
            ],
            'handler' => 'toolTemplatePreview',
        ];

        $this->tools['template_create_plan'] = [
            'description' => 'Create a new plan from a template',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'template' => [
                        'type' => 'string',
                        'description' => 'Template name/slug',
                    ],
                    'variables' => [
                        'type' => 'object',
                        'description' => 'Variable values for the template',
                    ],
                    'slug' => [
                        'type' => 'string',
                        'description' => 'Custom slug for the plan',
                    ],
                ],
                'required' => ['template', 'variables'],
            ],
            'handler' => 'toolTemplateCreatePlan',
        ];

        // Content generation tools
        $this->tools['content_status'] = [
            'description' => 'Get content generation pipeline status (AI provider availability, brief counts)',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [],
            ],
            'handler' => 'toolContentStatus',
        ];

        $this->tools['content_brief_create'] = [
            'description' => 'Create a content brief for AI generation',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'title' => [
                        'type' => 'string',
                        'description' => 'Content title',
                    ],
                    'content_type' => [
                        'type' => 'string',
                        'description' => 'Type of content',
                        'enum' => ['help_article', 'blog_post', 'landing_page', 'social_post'],
                    ],
                    'service' => [
                        'type' => 'string',
                        'description' => 'Service context (e.g., BioHost, QRHost)',
                    ],
                    'keywords' => [
                        'type' => 'array',
                        'description' => 'SEO keywords to include',
                        'items' => ['type' => 'string'],
                    ],
                    'target_word_count' => [
                        'type' => 'integer',
                        'description' => 'Target word count (default: 800)',
                    ],
                    'description' => [
                        'type' => 'string',
                        'description' => 'Brief description of what to write about',
                    ],
                    'difficulty' => [
                        'type' => 'string',
                        'description' => 'Target audience level',
                        'enum' => ['beginner', 'intermediate', 'advanced'],
                    ],
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Link to an existing plan',
                    ],
                ],
                'required' => ['title', 'content_type'],
            ],
            'handler' => 'toolContentBriefCreate',
        ];

        $this->tools['content_brief_list'] = [
            'description' => 'List content briefs with optional status filter',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'status' => [
                        'type' => 'string',
                        'description' => 'Filter by status',
                        'enum' => ['pending', 'queued', 'generating', 'review', 'published', 'failed'],
                    ],
                    'limit' => [
                        'type' => 'integer',
                        'description' => 'Maximum results (default: 20)',
                    ],
                ],
            ],
            'handler' => 'toolContentBriefList',
        ];

        $this->tools['content_brief_get'] = [
            'description' => 'Get details of a specific content brief including generated content',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'id' => [
                        'type' => 'integer',
                        'description' => 'Brief ID',
                    ],
                ],
                'required' => ['id'],
            ],
            'handler' => 'toolContentBriefGet',
        ];

        $this->tools['content_generate'] = [
            'description' => 'Generate content for a brief using AI pipeline (Gemini draft → Claude refine)',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'brief_id' => [
                        'type' => 'integer',
                        'description' => 'Brief ID to generate content for',
                    ],
                    'mode' => [
                        'type' => 'string',
                        'description' => 'Generation mode',
                        'enum' => ['draft', 'refine', 'full'],
                    ],
                    'sync' => [
                        'type' => 'boolean',
                        'description' => 'Run synchronously (wait for result) vs queue for async processing',
                    ],
                ],
                'required' => ['brief_id'],
            ],
            'handler' => 'toolContentGenerate',
        ];

        $this->tools['content_batch_generate'] = [
            'description' => 'Queue multiple briefs for batch content generation',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'limit' => [
                        'type' => 'integer',
                        'description' => 'Maximum briefs to process (default: 5)',
                    ],
                    'mode' => [
                        'type' => 'string',
                        'description' => 'Generation mode',
                        'enum' => ['draft', 'refine', 'full'],
                    ],
                ],
            ],
            'handler' => 'toolContentBatchGenerate',
        ];

        $this->tools['content_from_plan'] = [
            'description' => 'Create content briefs from plan tasks and queue for generation',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'plan_slug' => [
                        'type' => 'string',
                        'description' => 'Plan slug to generate content from',
                    ],
                    'content_type' => [
                        'type' => 'string',
                        'description' => 'Type of content to generate',
                        'enum' => ['help_article', 'blog_post', 'landing_page', 'social_post'],
                    ],
                    'service' => [
                        'type' => 'string',
                        'description' => 'Service context',
                    ],
                    'limit' => [
                        'type' => 'integer',
                        'description' => 'Maximum briefs to create (default: 5)',
                    ],
                    'target_word_count' => [
                        'type' => 'integer',
                        'description' => 'Target word count per article',
                    ],
                ],
                'required' => ['plan_slug'],
            ],
            'handler' => 'toolContentFromPlan',
        ];

        $this->tools['content_usage_stats'] = [
            'description' => 'Get AI usage statistics (tokens, costs) for content generation',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'period' => [
                        'type' => 'string',
                        'description' => 'Time period for stats',
                        'enum' => ['day', 'week', 'month', 'year'],
                    ],
                ],
            ],
            'handler' => 'toolContentUsageStats',
        ];
    }

    protected function registerResources(): void
    {
        $this->resources['plans://all'] = [
            'name' => 'All Plans Overview',
            'description' => 'Overview of all work plans and their status',
            'mimeType' => 'text/markdown',
            'handler' => 'resourceAllPlans',
        ];

        // Dynamic plan resources are handled in getResourcesList
    }

    protected function handleRequest(array $request): ?array
    {
        $method = $request['method'] ?? '';
        $id = $request['id'] ?? null;
        $params = $request['params'] ?? [];

        return match ($method) {
            'initialize' => $this->handleInitialize($id, $params),
            'tools/list' => $this->handleToolsList($id),
            'tools/call' => $this->handleToolsCall($id, $params),
            'resources/list' => $this->handleResourcesList($id),
            'resources/read' => $this->handleResourcesRead($id, $params),
            'notifications/initialized' => null,
            default => $this->errorResponse($id, -32601, "Method not found: {$method}"),
        };
    }

    protected function handleInitialize(mixed $id, array $params): array
    {
        return [
            'jsonrpc' => '2.0',
            'id' => $id,
            'result' => [
                'protocolVersion' => '2024-11-05',
                'capabilities' => [
                    'tools' => ['listChanged' => true],
                    'resources' => ['subscribe' => false, 'listChanged' => true],
                ],
                'serverInfo' => [
                    'name' => 'hosthub-agent',
                    'version' => '1.0.0',
                ],
            ],
        ];
    }

    protected function handleToolsList(mixed $id): array
    {
        $tools = [];

        foreach ($this->tools as $name => $tool) {
            $tools[] = [
                'name' => $name,
                'description' => $tool['description'],
                'inputSchema' => $tool['inputSchema'],
            ];
        }

        return [
            'jsonrpc' => '2.0',
            'id' => $id,
            'result' => ['tools' => $tools],
        ];
    }

    protected function handleToolsCall(mixed $id, array $params): array
    {
        $toolName = $params['name'] ?? '';
        $args = $params['arguments'] ?? [];
        $startTime = microtime(true);

        if (! isset($this->tools[$toolName])) {
            return $this->errorResponse($id, -32602, "Unknown tool: {$toolName}");
        }

        try {
            $handler = $this->tools[$toolName]['handler'];
            $result = $this->$handler($args);

            // Log tool call
            $this->logToolCall($toolName, $args, $result, $startTime, true);

            return [
                'jsonrpc' => '2.0',
                'id' => $id,
                'result' => [
                    'content' => [
                        [
                            'type' => 'text',
                            'text' => json_encode($result, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES),
                        ],
                    ],
                ],
            ];
        } catch (Throwable $e) {
            $this->logToolCall($toolName, $args, ['error' => $e->getMessage()], $startTime, false);

            Log::error('MCP tool error', [
                'tool' => $toolName,
                'error' => $e->getMessage(),
                'trace' => $e->getTraceAsString(),
            ]);

            return $this->errorResponse($id, -32603, $e->getMessage());
        }
    }

    protected function handleResourcesList(mixed $id): array
    {
        $resources = [];

        // Static resources
        foreach ($this->resources as $uri => $resource) {
            $resources[] = [
                'uri' => $uri,
                'name' => $resource['name'],
                'description' => $resource['description'],
                'mimeType' => $resource['mimeType'],
            ];
        }

        // Dynamic plan resources
        $plans = AgentPlan::notArchived()->get();
        foreach ($plans as $plan) {
            $resources[] = [
                'uri' => "plans://{$plan->slug}",
                'name' => $plan->title,
                'description' => "Work plan: {$plan->title}",
                'mimeType' => 'text/markdown',
            ];
        }

        return [
            'jsonrpc' => '2.0',
            'id' => $id,
            'result' => ['resources' => $resources],
        ];
    }

    protected function handleResourcesRead(mixed $id, array $params): array
    {
        $uri = $params['uri'] ?? '';

        // Handle static resources
        if (isset($this->resources[$uri])) {
            $handler = $this->resources[$uri]['handler'];
            $content = $this->$handler();

            return [
                'jsonrpc' => '2.0',
                'id' => $id,
                'result' => [
                    'contents' => [
                        [
                            'uri' => $uri,
                            'mimeType' => $this->resources[$uri]['mimeType'],
                            'text' => $content,
                        ],
                    ],
                ],
            ];
        }

        // Handle dynamic plan resources
        if (str_starts_with($uri, 'plans://')) {
            $path = substr($uri, 9); // Remove 'plans://'
            $parts = explode('/', $path);
            $slug = $parts[0];

            // plans://{slug}/phases/{order}
            if (count($parts) === 3 && $parts[1] === 'phases') {
                $content = $this->resourcePhaseChecklist($slug, (int) $parts[2]);
            }
            // plans://{slug}/state/{key}
            elseif (count($parts) === 3 && $parts[1] === 'state') {
                $content = $this->resourceStateValue($slug, $parts[2]);
            }
            // plans://{slug}
            else {
                $content = $this->resourcePlanDocument($slug);
            }

            return [
                'jsonrpc' => '2.0',
                'id' => $id,
                'result' => [
                    'contents' => [
                        [
                            'uri' => $uri,
                            'mimeType' => 'text/markdown',
                            'text' => $content,
                        ],
                    ],
                ],
            ];
        }

        // Handle session resources
        if (str_starts_with($uri, 'sessions://')) {
            $path = substr($uri, 11);
            $parts = explode('/', $path);

            if (count($parts) === 2 && $parts[1] === 'context') {
                $content = $this->resourceSessionContext($parts[0]);

                return [
                    'jsonrpc' => '2.0',
                    'id' => $id,
                    'result' => [
                        'contents' => [
                            [
                                'uri' => $uri,
                                'mimeType' => 'text/markdown',
                                'text' => $content,
                            ],
                        ],
                    ],
                ];
            }
        }

        return $this->errorResponse($id, -32602, "Resource not found: {$uri}");
    }

    protected function sendResponse(array $response): void
    {
        echo json_encode($response, JSON_UNESCAPED_SLASHES)."\n";
        flush();
    }

    protected function logToolCall(string $tool, array $args, array $result, float $startTime, bool $success): void
    {
        $duration = (int) ((microtime(true) - $startTime) * 1000);

        // Use the log() method which updates daily stats automatically
        McpToolCall::log(
            serverId: 'hosthub-agent',
            toolName: $tool,
            params: $args,
            success: $success,
            durationMs: $duration,
            errorMessage: $success ? null : ($result['error'] ?? null),
            errorCode: $success ? null : ($result['code'] ?? null),
            resultSummary: $success ? $result : null,
            sessionId: $this->currentSessionId,
        );
    }

    // ===== TOOL IMPLEMENTATIONS =====

    protected function toolPlanList(array $args): array
    {
        $query = AgentPlan::with('agentPhases')
            ->orderBy('updated_at', 'desc');

        if (! ($args['include_archived'] ?? false)) {
            $query->notArchived();
        }

        if (! empty($args['status'])) {
            $query->where('status', $args['status']);
        }

        $plans = $query->get();

        return [
            'plans' => $plans->map(fn ($plan) => [
                'slug' => $plan->slug,
                'title' => $plan->title,
                'status' => $plan->status,
                'progress' => $plan->getProgress(),
                'updated_at' => $plan->updated_at->toIso8601String(),
            ])->all(),
            'total' => $plans->count(),
        ];
    }

    protected function toolPlanCreate(array $args): array
    {
        $slug = $args['slug'] ?? Str::slug($args['title']).'-'.Str::random(6);

        if (AgentPlan::where('slug', $slug)->exists()) {
            return ['error' => "Plan with slug '{$slug}' already exists"];
        }

        $plan = AgentPlan::create([
            'slug' => $slug,
            'title' => $args['title'],
            'description' => $args['description'] ?? null,
            'status' => 'draft',
            'context' => $args['context'] ?? [],
        ]);

        // Create phases if provided
        if (! empty($args['phases'])) {
            foreach ($args['phases'] as $order => $phaseData) {
                $tasks = collect($phaseData['tasks'] ?? [])->map(fn ($task) => [
                    'name' => $task,
                    'status' => 'pending',
                ])->all();

                AgentPhase::create([
                    'agent_plan_id' => $plan->id,
                    'name' => $phaseData['name'],
                    'description' => $phaseData['description'] ?? null,
                    'order' => $order + 1,
                    'status' => 'pending',
                    'tasks' => $tasks,
                ]);
            }
        }

        $plan->load('agentPhases');

        return [
            'success' => true,
            'plan' => [
                'slug' => $plan->slug,
                'title' => $plan->title,
                'status' => $plan->status,
                'phases' => $plan->agentPhases->count(),
            ],
        ];
    }

    protected function toolPlanGet(array $args): array
    {
        $plan = AgentPlan::with('agentPhases')
            ->where('slug', $args['slug'])
            ->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['slug']}"];
        }

        $format = $args['format'] ?? 'json';

        if ($format === 'markdown') {
            return ['markdown' => $plan->toMarkdown()];
        }

        return [
            'plan' => [
                'slug' => $plan->slug,
                'title' => $plan->title,
                'description' => $plan->description,
                'status' => $plan->status,
                'context' => $plan->context,
                'progress' => $plan->getProgress(),
                'phases' => $plan->agentPhases->map(fn ($phase) => [
                    'order' => $phase->order,
                    'name' => $phase->name,
                    'description' => $phase->description,
                    'status' => $phase->status,
                    'tasks' => $phase->tasks,
                    'checkpoints' => $phase->checkpoints,
                ])->all(),
                'created_at' => $plan->created_at->toIso8601String(),
                'updated_at' => $plan->updated_at->toIso8601String(),
            ],
        ];
    }

    protected function toolPlanUpdateStatus(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['slug']}"];
        }

        $plan->update(['status' => $args['status']]);

        return [
            'success' => true,
            'plan' => [
                'slug' => $plan->slug,
                'status' => $plan->fresh()->status,
            ],
        ];
    }

    protected function toolPlanArchive(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['slug']}"];
        }

        $plan->archive($args['reason'] ?? null);

        return [
            'success' => true,
            'plan' => [
                'slug' => $plan->slug,
                'status' => 'archived',
                'archived_at' => $plan->archived_at->toIso8601String(),
            ],
        ];
    }

    protected function toolPhaseGet(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $phase = $this->findPhase($plan, $args['phase']);

        if (! $phase) {
            return ['error' => "Phase not found: {$args['phase']}"];
        }

        return [
            'phase' => [
                'order' => $phase->order,
                'name' => $phase->name,
                'description' => $phase->description,
                'status' => $phase->status,
                'tasks' => $phase->tasks,
                'checkpoints' => $phase->checkpoints,
                'dependencies' => $phase->dependencies,
            ],
        ];
    }

    protected function toolPhaseUpdateStatus(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $phase = $this->findPhase($plan, $args['phase']);

        if (! $phase) {
            return ['error' => "Phase not found: {$args['phase']}"];
        }

        $updateData = ['status' => $args['status']];

        if (! empty($args['notes'])) {
            $phase->addCheckpoint($args['notes'], ['status_change' => $args['status']]);
        }

        $phase->update($updateData);

        return [
            'success' => true,
            'phase' => [
                'order' => $phase->order,
                'name' => $phase->name,
                'status' => $phase->fresh()->status,
            ],
        ];
    }

    protected function toolPhaseAddCheckpoint(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $phase = $this->findPhase($plan, $args['phase']);

        if (! $phase) {
            return ['error' => "Phase not found: {$args['phase']}"];
        }

        $phase->addCheckpoint($args['note'], $args['context'] ?? []);

        return [
            'success' => true,
            'checkpoints' => $phase->fresh()->checkpoints,
        ];
    }

    protected function toolTaskToggle(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $phase = $this->findPhase($plan, $args['phase']);

        if (! $phase) {
            return ['error' => "Phase not found: {$args['phase']}"];
        }

        $tasks = $phase->tasks ?? [];
        $index = $args['task_index'];

        if (! isset($tasks[$index])) {
            return ['error' => "Task not found at index: {$index}"];
        }

        $currentStatus = is_string($tasks[$index])
            ? 'pending'
            : ($tasks[$index]['status'] ?? 'pending');

        $newStatus = $currentStatus === 'completed' ? 'pending' : 'completed';

        if (is_string($tasks[$index])) {
            $tasks[$index] = [
                'name' => $tasks[$index],
                'status' => $newStatus,
            ];
        } else {
            $tasks[$index]['status'] = $newStatus;
        }

        $phase->update(['tasks' => $tasks]);

        return [
            'success' => true,
            'task' => $tasks[$index],
            'plan_progress' => $plan->fresh()->getProgress(),
        ];
    }

    protected function toolTaskUpdate(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $phase = $this->findPhase($plan, $args['phase']);

        if (! $phase) {
            return ['error' => "Phase not found: {$args['phase']}"];
        }

        $tasks = $phase->tasks ?? [];
        $index = $args['task_index'];

        if (! isset($tasks[$index])) {
            return ['error' => "Task not found at index: {$index}"];
        }

        if (is_string($tasks[$index])) {
            $tasks[$index] = ['name' => $tasks[$index], 'status' => 'pending'];
        }

        if (isset($args['status'])) {
            $tasks[$index]['status'] = $args['status'];
        }

        if (isset($args['notes'])) {
            $tasks[$index]['notes'] = $args['notes'];
        }

        $phase->update(['tasks' => $tasks]);

        return [
            'success' => true,
            'task' => $tasks[$index],
        ];
    }

    protected function toolSessionStart(array $args): array
    {
        $plan = null;
        if (! empty($args['plan_slug'])) {
            $plan = AgentPlan::where('slug', $args['plan_slug'])->first();
        }

        $sessionId = 'ses_'.Str::random(12);
        $this->currentSessionId = $sessionId;

        $session = AgentSession::create([
            'session_id' => $sessionId,
            'agent_plan_id' => $plan?->id,
            'agent_type' => $args['agent_type'],
            'status' => 'active',
            'started_at' => now(),
            'context_summary' => $args['context'] ?? [],
        ]);

        return [
            'success' => true,
            'session' => [
                'session_id' => $session->session_id,
                'agent_type' => $session->agent_type,
                'plan' => $plan?->slug,
                'status' => $session->status,
            ],
        ];
    }

    protected function toolSessionLog(array $args): array
    {
        if (! $this->currentSessionId) {
            return ['error' => 'No active session. Call session_start first.'];
        }

        $session = AgentSession::where('session_id', $this->currentSessionId)->first();

        if (! $session) {
            return ['error' => 'Session not found'];
        }

        $session->addWorkLogEntry(
            $args['message'],
            $args['type'] ?? 'info',
            $args['data'] ?? []
        );

        return ['success' => true, 'logged' => $args['message']];
    }

    protected function toolSessionArtifact(array $args): array
    {
        if (! $this->currentSessionId) {
            return ['error' => 'No active session. Call session_start first.'];
        }

        $session = AgentSession::where('session_id', $this->currentSessionId)->first();

        if (! $session) {
            return ['error' => 'Session not found'];
        }

        $session->addArtifact(
            $args['path'],
            $args['action'],
            $args['description'] ?? null
        );

        return ['success' => true, 'artifact' => $args['path']];
    }

    protected function toolSessionHandoff(array $args): array
    {
        if (! $this->currentSessionId) {
            return ['error' => 'No active session. Call session_start first.'];
        }

        $session = AgentSession::where('session_id', $this->currentSessionId)->first();

        if (! $session) {
            return ['error' => 'Session not found'];
        }

        $session->prepareHandoff(
            $args['summary'],
            $args['next_steps'] ?? [],
            $args['blockers'] ?? [],
            $args['context_for_next'] ?? []
        );

        return [
            'success' => true,
            'handoff_context' => $session->getHandoffContext(),
        ];
    }

    protected function toolSessionEnd(array $args): array
    {
        if (! $this->currentSessionId) {
            return ['error' => 'No active session'];
        }

        $session = AgentSession::where('session_id', $this->currentSessionId)->first();

        if (! $session) {
            return ['error' => 'Session not found'];
        }

        $session->end($args['status'], $args['summary'] ?? null);
        $this->currentSessionId = null;

        return [
            'success' => true,
            'session' => [
                'session_id' => $session->session_id,
                'status' => $session->status,
                'duration' => $session->getDurationFormatted(),
            ],
        ];
    }

    protected function toolStateGet(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $state = $plan->states()->where('key', $args['key'])->first();

        if (! $state) {
            return ['error' => "State not found: {$args['key']}"];
        }

        return [
            'key' => $state->key,
            'value' => $state->value,
            'category' => $state->category,
            'updated_at' => $state->updated_at->toIso8601String(),
        ];
    }

    protected function toolStateSet(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $state = AgentWorkspaceState::updateOrCreate(
            [
                'agent_plan_id' => $plan->id,
                'key' => $args['key'],
            ],
            [
                'value' => $args['value'],
                'category' => $args['category'] ?? 'general',
            ]
        );

        return [
            'success' => true,
            'state' => [
                'key' => $state->key,
                'value' => $state->value,
                'category' => $state->category,
            ],
        ];
    }

    protected function toolStateList(array $args): array
    {
        $plan = AgentPlan::where('slug', $args['plan_slug'])->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $query = $plan->states();

        if (! empty($args['category'])) {
            $query->where('category', $args['category']);
        }

        $states = $query->get();

        return [
            'states' => $states->map(fn ($state) => [
                'key' => $state->key,
                'value' => $state->value,
                'category' => $state->category,
            ])->all(),
            'total' => $states->count(),
        ];
    }

    protected function toolTemplateList(array $args): array
    {
        $templateService = app(PlanTemplateService::class);
        $templates = $templateService->listTemplates();

        if (! empty($args['category'])) {
            $templates = array_filter($templates, fn ($t) => ($t['category'] ?? '') === $args['category']);
        }

        return [
            'templates' => array_values($templates),
            'total' => count($templates),
        ];
    }

    protected function toolTemplatePreview(array $args): array
    {
        $templateService = app(PlanTemplateService::class);
        $templateSlug = $args['template'];
        $variables = $args['variables'] ?? [];

        $preview = $templateService->previewTemplate($templateSlug, $variables);

        if (! $preview) {
            return ['error' => "Template not found: {$templateSlug}"];
        }

        return [
            'template' => $templateSlug,
            'preview' => $preview,
        ];
    }

    protected function toolTemplateCreatePlan(array $args): array
    {
        $templateService = app(PlanTemplateService::class);
        $templateSlug = $args['template'];
        $variables = $args['variables'] ?? [];

        $options = [];

        if (! empty($args['slug'])) {
            $options['slug'] = $args['slug'];
        }

        $plan = $templateService->createPlan($templateSlug, $variables, $options);

        if (! $plan) {
            return ['error' => 'Failed to create plan from template'];
        }

        return [
            'success' => true,
            'plan' => [
                'slug' => $plan->slug,
                'title' => $plan->title,
                'status' => $plan->status,
                'phases' => $plan->agentPhases->count(),
                'total_tasks' => $plan->getProgress()['total'],
            ],
            'commands' => [
                'view' => "php artisan plan:show {$plan->slug}",
                'activate' => "php artisan plan:status {$plan->slug} --set=active",
            ],
        ];
    }

    // ===== CONTENT GENERATION TOOL IMPLEMENTATIONS =====

    protected function toolContentStatus(array $args): array
    {
        $gateway = app(AIGatewayService::class);

        return [
            'providers' => [
                'gemini' => $gateway->isGeminiAvailable(),
                'claude' => $gateway->isClaudeAvailable(),
            ],
            'pipeline_available' => $gateway->isAvailable(),
            'briefs' => [
                'pending' => ContentBrief::pending()->count(),
                'queued' => ContentBrief::where('status', ContentBrief::STATUS_QUEUED)->count(),
                'generating' => ContentBrief::where('status', ContentBrief::STATUS_GENERATING)->count(),
                'review' => ContentBrief::needsReview()->count(),
                'published' => ContentBrief::where('status', ContentBrief::STATUS_PUBLISHED)->count(),
                'failed' => ContentBrief::where('status', ContentBrief::STATUS_FAILED)->count(),
            ],
        ];
    }

    protected function toolContentBriefCreate(array $args): array
    {
        $plan = null;
        if (! empty($args['plan_slug'])) {
            $plan = AgentPlan::where('slug', $args['plan_slug'])->first();
        }

        $brief = ContentBrief::create([
            'title' => $args['title'],
            'slug' => Str::slug($args['title']).'-'.Str::random(6),
            'content_type' => $args['content_type'],
            'service' => $args['service'] ?? null,
            'description' => $args['description'] ?? null,
            'keywords' => $args['keywords'] ?? null,
            'target_word_count' => $args['target_word_count'] ?? 800,
            'difficulty' => $args['difficulty'] ?? null,
            'status' => ContentBrief::STATUS_PENDING,
            'metadata' => $plan ? [
                'plan_id' => $plan->id,
                'plan_slug' => $plan->slug,
            ] : null,
        ]);

        return [
            'success' => true,
            'brief' => [
                'id' => $brief->id,
                'title' => $brief->title,
                'slug' => $brief->slug,
                'status' => $brief->status,
                'content_type' => $brief->content_type,
            ],
        ];
    }

    protected function toolContentBriefList(array $args): array
    {
        $query = ContentBrief::query()->orderBy('created_at', 'desc');

        if (! empty($args['status'])) {
            $query->where('status', $args['status']);
        }

        $limit = $args['limit'] ?? 20;
        $briefs = $query->limit($limit)->get();

        return [
            'briefs' => $briefs->map(fn ($brief) => [
                'id' => $brief->id,
                'title' => $brief->title,
                'status' => $brief->status,
                'content_type' => $brief->content_type,
                'service' => $brief->service,
                'created_at' => $brief->created_at->toIso8601String(),
            ])->all(),
            'total' => $briefs->count(),
        ];
    }

    protected function toolContentBriefGet(array $args): array
    {
        $brief = ContentBrief::find($args['id']);

        if (! $brief) {
            return ['error' => "Brief not found: {$args['id']}"];
        }

        return [
            'brief' => [
                'id' => $brief->id,
                'title' => $brief->title,
                'slug' => $brief->slug,
                'status' => $brief->status,
                'content_type' => $brief->content_type,
                'service' => $brief->service,
                'description' => $brief->description,
                'keywords' => $brief->keywords,
                'target_word_count' => $brief->target_word_count,
                'difficulty' => $brief->difficulty,
                'draft_output' => $brief->draft_output,
                'refined_output' => $brief->refined_output,
                'final_content' => $brief->final_content,
                'best_content' => $brief->best_content,
                'error_message' => $brief->error_message,
                'generation_log' => $brief->generation_log,
                'total_cost' => $brief->total_cost,
                'created_at' => $brief->created_at->toIso8601String(),
                'generated_at' => $brief->generated_at?->toIso8601String(),
                'refined_at' => $brief->refined_at?->toIso8601String(),
            ],
        ];
    }

    protected function toolContentGenerate(array $args): array
    {
        $brief = ContentBrief::find($args['brief_id']);

        if (! $brief) {
            return ['error' => "Brief not found: {$args['brief_id']}"];
        }

        $gateway = app(AIGatewayService::class);

        if (! $gateway->isAvailable()) {
            return ['error' => 'AI providers not configured. Set GOOGLE_AI_API_KEY and ANTHROPIC_API_KEY.'];
        }

        $mode = $args['mode'] ?? 'full';
        $sync = $args['sync'] ?? false;

        if ($sync) {
            try {
                if ($mode === 'full') {
                    $result = $gateway->generateAndRefine($brief);

                    return [
                        'success' => true,
                        'brief_id' => $brief->id,
                        'status' => $brief->fresh()->status,
                        'draft' => [
                            'model' => $result['draft']->model,
                            'tokens' => $result['draft']->totalTokens(),
                            'cost' => $result['draft']->estimateCost(),
                        ],
                        'refined' => [
                            'model' => $result['refined']->model,
                            'tokens' => $result['refined']->totalTokens(),
                            'cost' => $result['refined']->estimateCost(),
                        ],
                    ];
                } elseif ($mode === 'draft') {
                    $response = $gateway->generateDraft($brief);
                    $brief->markDraftComplete($response->content);

                    return [
                        'success' => true,
                        'brief_id' => $brief->id,
                        'status' => $brief->fresh()->status,
                        'draft' => [
                            'model' => $response->model,
                            'tokens' => $response->totalTokens(),
                            'cost' => $response->estimateCost(),
                        ],
                    ];
                } elseif ($mode === 'refine') {
                    if (! $brief->isGenerated()) {
                        return ['error' => 'No draft to refine. Generate draft first.'];
                    }
                    $response = $gateway->refineDraft($brief, $brief->draft_output);
                    $brief->markRefined($response->content);

                    return [
                        'success' => true,
                        'brief_id' => $brief->id,
                        'status' => $brief->fresh()->status,
                        'refined' => [
                            'model' => $response->model,
                            'tokens' => $response->totalTokens(),
                            'cost' => $response->estimateCost(),
                        ],
                    ];
                }
            } catch (\Exception $e) {
                $brief->markFailed($e->getMessage());

                return ['error' => $e->getMessage()];
            }
        }

        // Async - queue for processing
        $brief->markQueued();
        GenerateContentJob::dispatch($brief, $mode);

        return [
            'success' => true,
            'queued' => true,
            'brief_id' => $brief->id,
            'mode' => $mode,
            'message' => 'Brief queued for generation',
        ];
    }

    protected function toolContentBatchGenerate(array $args): array
    {
        $limit = $args['limit'] ?? 5;
        $mode = $args['mode'] ?? 'full';

        $briefs = ContentBrief::readyToProcess()->limit($limit)->get();

        if ($briefs->isEmpty()) {
            return ['message' => 'No briefs ready for processing', 'queued' => 0];
        }

        foreach ($briefs as $brief) {
            GenerateContentJob::dispatch($brief, $mode);
        }

        return [
            'success' => true,
            'queued' => $briefs->count(),
            'mode' => $mode,
            'brief_ids' => $briefs->pluck('id')->all(),
        ];
    }

    protected function toolContentFromPlan(array $args): array
    {
        $plan = AgentPlan::with('agentPhases')
            ->where('slug', $args['plan_slug'])
            ->first();

        if (! $plan) {
            return ['error' => "Plan not found: {$args['plan_slug']}"];
        }

        $limit = $args['limit'] ?? 5;
        $contentType = $args['content_type'] ?? 'help_article';
        $service = $args['service'] ?? ($plan->metadata['service'] ?? null);
        $wordCount = $args['target_word_count'] ?? 800;

        $phases = $plan->agentPhases()
            ->whereIn('status', ['pending', 'in_progress'])
            ->get();

        if ($phases->isEmpty()) {
            return ['message' => 'No pending phases in plan', 'created' => 0];
        }

        $briefsCreated = [];

        foreach ($phases as $phase) {
            $tasks = $phase->getTasks();

            foreach ($tasks as $index => $task) {
                if (count($briefsCreated) >= $limit) {
                    break 2;
                }

                $taskName = is_string($task) ? $task : ($task['name'] ?? '');
                $taskStatus = is_array($task) ? ($task['status'] ?? 'pending') : 'pending';

                if ($taskStatus === 'completed') {
                    continue;
                }

                $brief = ContentBrief::create([
                    'title' => $taskName,
                    'slug' => Str::slug($taskName).'-'.time(),
                    'content_type' => $contentType,
                    'service' => $service,
                    'target_word_count' => $wordCount,
                    'status' => ContentBrief::STATUS_QUEUED,
                    'metadata' => [
                        'plan_id' => $plan->id,
                        'plan_slug' => $plan->slug,
                        'phase_id' => $phase->id,
                        'phase_order' => $phase->order,
                        'task_index' => $index,
                    ],
                ]);

                GenerateContentJob::dispatch($brief, 'full');
                $briefsCreated[] = [
                    'id' => $brief->id,
                    'title' => $brief->title,
                ];
            }
        }

        return [
            'success' => true,
            'plan' => $plan->slug,
            'created' => count($briefsCreated),
            'briefs' => $briefsCreated,
        ];
    }

    protected function toolContentUsageStats(array $args): array
    {
        $period = $args['period'] ?? 'month';
        $stats = AIUsage::statsForWorkspace(null, $period);

        return [
            'period' => $period,
            'total_requests' => $stats['total_requests'],
            'total_input_tokens' => $stats['total_input_tokens'],
            'total_output_tokens' => $stats['total_output_tokens'],
            'total_cost' => number_format($stats['total_cost'], 4),
            'by_provider' => $stats['by_provider'],
            'by_purpose' => $stats['by_purpose'],
        ];
    }

    // ===== RESOURCE IMPLEMENTATIONS =====

    protected function resourceAllPlans(): string
    {
        $plans = AgentPlan::with('agentPhases')->notArchived()->orderBy('updated_at', 'desc')->get();

        $md = "# Work Plans\n\n";
        $md .= '**Total:** '.$plans->count()." plan(s)\n\n";

        foreach ($plans->groupBy('status') as $status => $group) {
            $md .= '## '.ucfirst($status).' ('.$group->count().")\n\n";

            foreach ($group as $plan) {
                $progress = $plan->getProgress();
                $md .= "- **[{$plan->slug}]** {$plan->title} - {$progress['percentage']}%\n";
            }
            $md .= "\n";
        }

        return $md;
    }

    protected function resourcePlanDocument(string $slug): string
    {
        $plan = AgentPlan::with('agentPhases')->where('slug', $slug)->first();

        if (! $plan) {
            return "Plan not found: {$slug}";
        }

        return $plan->toMarkdown();
    }

    protected function resourcePhaseChecklist(string $slug, int $phaseOrder): string
    {
        $plan = AgentPlan::where('slug', $slug)->first();

        if (! $plan) {
            return "Plan not found: {$slug}";
        }

        $phase = $plan->agentPhases()->where('order', $phaseOrder)->first();

        if (! $phase) {
            return "Phase not found: {$phaseOrder}";
        }

        $md = "# Phase {$phase->order}: {$phase->name}\n\n";
        $md .= "**Status:** {$phase->getStatusIcon()} {$phase->status}\n\n";

        if ($phase->description) {
            $md .= "{$phase->description}\n\n";
        }

        $md .= "## Tasks\n\n";

        foreach ($phase->tasks ?? [] as $task) {
            $status = is_string($task) ? 'pending' : ($task['status'] ?? 'pending');
            $name = is_string($task) ? $task : ($task['name'] ?? 'Unknown');
            $icon = $status === 'completed' ? '✅' : '⬜';
            $md .= "- {$icon} {$name}\n";
        }

        return $md;
    }

    protected function resourceStateValue(string $slug, string $key): string
    {
        $plan = AgentPlan::where('slug', $slug)->first();

        if (! $plan) {
            return "Plan not found: {$slug}";
        }

        $state = $plan->states()->where('key', $key)->first();

        if (! $state) {
            return "State key not found: {$key}";
        }

        return $state->getFormattedValue();
    }

    protected function resourceSessionContext(string $sessionId): string
    {
        $session = AgentSession::where('session_id', $sessionId)->first();

        if (! $session) {
            return "Session not found: {$sessionId}";
        }

        $context = $session->getHandoffContext();

        $md = "# Session: {$session->session_id}\n\n";
        $md .= "**Agent:** {$session->agent_type}\n";
        $md .= "**Status:** {$session->status}\n";
        $md .= "**Duration:** {$session->getDurationFormatted()}\n\n";

        if ($session->plan) {
            $md .= "## Plan\n\n";
            $md .= "**{$session->plan->title}** ({$session->plan->slug})\n\n";
        }

        if (! empty($context['context_summary'])) {
            $md .= "## Context Summary\n\n";
            $md .= json_encode($context['context_summary'], JSON_PRETTY_PRINT)."\n\n";
        }

        if (! empty($context['handoff_notes'])) {
            $md .= "## Handoff Notes\n\n";
            $md .= json_encode($context['handoff_notes'], JSON_PRETTY_PRINT)."\n\n";
        }

        if (! empty($context['artifacts'])) {
            $md .= "## Artifacts\n\n";
            foreach ($context['artifacts'] as $artifact) {
                $md .= "- {$artifact['action']}: {$artifact['path']}\n";
            }
            $md .= "\n";
        }

        return $md;
    }

    // ===== HELPERS =====

    protected function findPhase(AgentPlan $plan, string|int $identifier): ?AgentPhase
    {
        if (is_numeric($identifier)) {
            return $plan->agentPhases()->where('order', (int) $identifier)->first();
        }

        return $plan->agentPhases()
            ->where(function ($query) use ($identifier) {
                $query->where('name', $identifier)
                    ->orWhere('order', $identifier);
            })
            ->first();
    }

    protected function errorResponse(mixed $id, int $code, string $message): array
    {
        return [
            'jsonrpc' => '2.0',
            'id' => $id,
            'error' => [
                'code' => $code,
                'message' => $message,
            ],
        ];
    }
}
