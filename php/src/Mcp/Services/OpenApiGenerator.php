<?php

declare(strict_types=1);

namespace Core\Mcp\Services;

use Symfony\Component\Yaml\Yaml;

/**
 * Generates OpenAPI 3.0 spec from MCP YAML definitions.
 */
class OpenApiGenerator
{
    protected array $registry;

    protected array $servers = [];

    public function generate(): array
    {
        $this->loadRegistry();
        $this->loadServers();

        return [
            'openapi' => '3.0.3',
            'info' => $this->buildInfo(),
            'servers' => $this->buildServers(),
            'tags' => $this->buildTags(),
            'paths' => $this->buildPaths(),
            'components' => $this->buildComponents(),
        ];
    }

    public function toJson(): string
    {
        return json_encode($this->generate(), JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES);
    }

    public function toYaml(): string
    {
        return Yaml::dump($this->generate(), 10, 2);
    }

    protected function loadRegistry(): void
    {
        $path = resource_path('mcp/registry.yaml');
        $this->registry = file_exists($path) ? Yaml::parseFile($path) : ['servers' => []];
    }

    protected function loadServers(): void
    {
        foreach ($this->registry['servers'] ?? [] as $ref) {
            $path = resource_path("mcp/servers/{$ref['id']}.yaml");
            if (file_exists($path)) {
                $this->servers[$ref['id']] = Yaml::parseFile($path);
            }
        }
    }

    protected function buildInfo(): array
    {
        return [
            'title' => 'Host UK MCP API',
            'description' => 'HTTP API for interacting with Host UK MCP servers. Execute tools, read resources, and discover available capabilities.',
            'version' => '1.0.0',
            'contact' => [
                'name' => 'Host UK Support',
                'url' => 'https://host.uk.com/contact',
                'email' => 'support@host.uk.com',
            ],
            'license' => [
                'name' => 'Proprietary',
                'url' => 'https://host.uk.com/terms',
            ],
        ];
    }

    protected function buildServers(): array
    {
        return [
            [
                'url' => 'https://mcp.host.uk.com/api/v1/mcp',
                'description' => 'Production',
            ],
            [
                'url' => 'https://mcp.test/api/v1/mcp',
                'description' => 'Local development',
            ],
        ];
    }

    protected function buildTags(): array
    {
        $tags = [
            [
                'name' => 'Discovery',
                'description' => 'Server and tool discovery endpoints',
            ],
            [
                'name' => 'Execution',
                'description' => 'Tool execution endpoints',
            ],
        ];

        foreach ($this->servers as $id => $server) {
            $tags[] = [
                'name' => $server['name'] ?? $id,
                'description' => $server['tagline'] ?? $server['description'] ?? '',
            ];
        }

        return $tags;
    }

    protected function buildPaths(): array
    {
        $paths = [];

        // Discovery endpoints
        $paths['/servers'] = [
            'get' => [
                'tags' => ['Discovery'],
                'summary' => 'List all MCP servers',
                'operationId' => 'listServers',
                'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                'responses' => [
                    '200' => [
                        'description' => 'List of available servers',
                        'content' => [
                            'application/json' => [
                                'schema' => [
                                    '$ref' => '#/components/schemas/ServerList',
                                ],
                            ],
                        ],
                    ],
                ],
            ],
        ];

        $paths['/servers/{serverId}'] = [
            'get' => [
                'tags' => ['Discovery'],
                'summary' => 'Get server details',
                'operationId' => 'getServer',
                'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                'parameters' => [
                    [
                        'name' => 'serverId',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                        'description' => 'Server identifier',
                    ],
                ],
                'responses' => [
                    '200' => [
                        'description' => 'Server details with tools and resources',
                        'content' => [
                            'application/json' => [
                                'schema' => [
                                    '$ref' => '#/components/schemas/Server',
                                ],
                            ],
                        ],
                    ],
                    '404' => ['description' => 'Server not found'],
                ],
            ],
        ];

        $paths['/servers/{serverId}/tools'] = [
            'get' => [
                'tags' => ['Discovery'],
                'summary' => 'List tools for a server',
                'operationId' => 'listServerTools',
                'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                'parameters' => [
                    [
                        'name' => 'serverId',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                    ],
                ],
                'responses' => [
                    '200' => [
                        'description' => 'List of tools',
                        'content' => [
                            'application/json' => [
                                'schema' => [
                                    '$ref' => '#/components/schemas/ToolList',
                                ],
                            ],
                        ],
                    ],
                ],
            ],
        ];

        $paths['/servers/{serverId}/resources'] = [
            'get' => [
                'tags' => ['Discovery'],
                'summary' => 'List resources for a server',
                'operationId' => 'listServerResources',
                'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                'parameters' => [
                    [
                        'name' => 'serverId',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                    ],
                ],
                'responses' => [
                    '200' => [
                        'description' => 'List of resources',
                        'content' => [
                            'application/json' => [
                                'schema' => [
                                    '$ref' => '#/components/schemas/ResourceList',
                                ],
                            ],
                        ],
                    ],
                ],
            ],
        ];

        // Execution endpoint
        $paths['/tools/call'] = [
            'post' => [
                'tags' => ['Execution'],
                'summary' => 'Execute an MCP tool',
                'operationId' => 'callTool',
                'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                'requestBody' => [
                    'required' => true,
                    'content' => [
                        'application/json' => [
                            'schema' => [
                                '$ref' => '#/components/schemas/ToolCallRequest',
                            ],
                        ],
                    ],
                ],
                'responses' => [
                    '200' => [
                        'description' => 'Tool executed successfully',
                        'content' => [
                            'application/json' => [
                                'schema' => [
                                    '$ref' => '#/components/schemas/ToolCallResponse',
                                ],
                            ],
                        ],
                    ],
                    '400' => ['description' => 'Invalid request'],
                    '401' => ['description' => 'Unauthorized'],
                    '404' => ['description' => 'Server or tool not found'],
                    '500' => ['description' => 'Tool execution error'],
                ],
            ],
        ];

        // Resource endpoint
        $paths['/resources/{uri}'] = [
            'get' => [
                'tags' => ['Execution'],
                'summary' => 'Read a resource',
                'operationId' => 'readResource',
                'security' => [['bearerAuth' => []], ['apiKeyAuth' => []]],
                'parameters' => [
                    [
                        'name' => 'uri',
                        'in' => 'path',
                        'required' => true,
                        'schema' => ['type' => 'string'],
                        'description' => 'Resource URI (server://path)',
                    ],
                ],
                'responses' => [
                    '200' => [
                        'description' => 'Resource content',
                        'content' => [
                            'application/json' => [
                                'schema' => [
                                    '$ref' => '#/components/schemas/ResourceResponse',
                                ],
                            ],
                        ],
                    ],
                ],
            ],
        ];

        return $paths;
    }

    protected function buildComponents(): array
    {
        return [
            'securitySchemes' => [
                'bearerAuth' => [
                    'type' => 'http',
                    'scheme' => 'bearer',
                    'description' => 'API key in Bearer format: hk_xxx_yyy',
                ],
                'apiKeyAuth' => [
                    'type' => 'apiKey',
                    'in' => 'header',
                    'name' => 'X-API-Key',
                    'description' => 'API key header',
                ],
            ],
            'schemas' => $this->buildSchemas(),
        ];
    }

    protected function buildSchemas(): array
    {
        $schemas = [
            'ServerList' => [
                'type' => 'object',
                'properties' => [
                    'servers' => [
                        'type' => 'array',
                        'items' => ['$ref' => '#/components/schemas/ServerSummary'],
                    ],
                    'count' => ['type' => 'integer'],
                ],
            ],
            'ServerSummary' => [
                'type' => 'object',
                'properties' => [
                    'id' => ['type' => 'string'],
                    'name' => ['type' => 'string'],
                    'tagline' => ['type' => 'string'],
                    'status' => ['type' => 'string', 'enum' => ['available', 'beta', 'deprecated']],
                    'tool_count' => ['type' => 'integer'],
                    'resource_count' => ['type' => 'integer'],
                ],
            ],
            'Server' => [
                'type' => 'object',
                'properties' => [
                    'id' => ['type' => 'string'],
                    'name' => ['type' => 'string'],
                    'tagline' => ['type' => 'string'],
                    'description' => ['type' => 'string'],
                    'tools' => [
                        'type' => 'array',
                        'items' => ['$ref' => '#/components/schemas/Tool'],
                    ],
                    'resources' => [
                        'type' => 'array',
                        'items' => ['$ref' => '#/components/schemas/Resource'],
                    ],
                ],
            ],
            'Tool' => [
                'type' => 'object',
                'properties' => [
                    'name' => ['type' => 'string'],
                    'description' => ['type' => 'string'],
                    'inputSchema' => [
                        'type' => 'object',
                        'additionalProperties' => true,
                    ],
                ],
            ],
            'Resource' => [
                'type' => 'object',
                'properties' => [
                    'uri' => ['type' => 'string'],
                    'name' => ['type' => 'string'],
                    'description' => ['type' => 'string'],
                    'mimeType' => ['type' => 'string'],
                ],
            ],
            'ToolList' => [
                'type' => 'object',
                'properties' => [
                    'server' => ['type' => 'string'],
                    'tools' => [
                        'type' => 'array',
                        'items' => ['$ref' => '#/components/schemas/Tool'],
                    ],
                    'count' => ['type' => 'integer'],
                ],
            ],
            'ToolCallRequest' => [
                'type' => 'object',
                'required' => ['server', 'tool'],
                'properties' => [
                    'server' => [
                        'type' => 'string',
                        'description' => 'Server ID',
                    ],
                    'tool' => [
                        'type' => 'string',
                        'description' => 'Tool name',
                    ],
                    'arguments' => [
                        'type' => 'object',
                        'description' => 'Tool arguments',
                        'additionalProperties' => true,
                    ],
                ],
            ],
            'ToolCallResponse' => [
                'type' => 'object',
                'properties' => [
                    'success' => ['type' => 'boolean'],
                    'server' => ['type' => 'string'],
                    'tool' => ['type' => 'string'],
                    'result' => [
                        'type' => 'object',
                        'additionalProperties' => true,
                    ],
                    'duration_ms' => ['type' => 'integer'],
                    'error' => ['type' => 'string'],
                ],
            ],
            'ResourceResponse' => [
                'type' => 'object',
                'properties' => [
                    'uri' => ['type' => 'string'],
                    'content' => [
                        'type' => 'object',
                        'additionalProperties' => true,
                    ],
                ],
            ],
            'ResourceList' => [
                'type' => 'object',
                'properties' => [
                    'server' => ['type' => 'string'],
                    'resources' => [
                        'type' => 'array',
                        'items' => ['$ref' => '#/components/schemas/Resource'],
                    ],
                    'count' => ['type' => 'integer'],
                ],
            ],
        ];

        return $schemas;
    }
}
