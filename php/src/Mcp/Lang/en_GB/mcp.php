<?php

declare(strict_types=1);

/**
 * MCP module translations (en_GB).
 *
 * Key structure: section.subsection.key
 */

return [
    // API Key Manager
    'keys' => [
        'title' => 'API Keys',
        'description' => 'Create API keys to authenticate HTTP requests to MCP servers.',
        'empty' => [
            'title' => 'No API Keys Yet',
            'description' => 'Create an API key to start making authenticated requests to MCP servers over HTTP.',
        ],
        'actions' => [
            'create' => 'Create Key',
            'create_first' => 'Create Your First Key',
            'revoke' => 'Revoke',
        ],
        'table' => [
            'name' => 'Name',
            'key' => 'Key',
            'scopes' => 'Scopes',
            'last_used' => 'Last Used',
            'expires' => 'Expires',
            'actions' => 'Actions',
        ],
        'status' => [
            'expired' => 'Expired',
            'never' => 'Never',
        ],
        'confirm_revoke' => 'Are you sure you want to revoke this API key? This cannot be undone.',

        // Authentication section
        'auth' => [
            'title' => 'Authentication',
            'description' => 'Include your API key in HTTP requests using one of these methods:',
            'header_recommended' => 'Authorization Header (recommended)',
            'header_api_key' => 'X-API-Key Header',
        ],

        // Example section
        'example' => [
            'title' => 'Example Request',
            'description' => 'Call an MCP tool via HTTP POST:',
        ],

        // Create modal
        'create_modal' => [
            'title' => 'Create API Key',
            'name_label' => 'Key Name',
            'name_placeholder' => 'e.g., Production Server, Claude Agent',
            'permissions_label' => 'Permissions',
            'permission_read' => 'Read - Query tools and resources',
            'permission_write' => 'Write - Create and update data',
            'permission_delete' => 'Delete - Remove data',
            'expiry_label' => 'Expiration',
            'expiry_never' => 'Never expires',
            'expiry_30' => '30 days',
            'expiry_90' => '90 days',
            'expiry_1year' => '1 year',
            'cancel' => 'Cancel',
            'create' => 'Create Key',
        ],

        // New key modal
        'new_key_modal' => [
            'title' => 'API Key Created',
            'warning' => 'Copy this key now.',
            'warning_detail' => "You won't be able to see it again.",
            'done' => 'Done',
        ],
    ],

    // Request Log
    'logs' => [
        'title' => 'Request Log',
        'description' => 'View API requests and generate curl commands to replay them.',
        'filters' => [
            'server' => 'Server',
            'status' => 'Status',
            'all_servers' => 'All servers',
            'all' => 'All',
            'success' => 'Success',
            'failed' => 'Failed',
        ],
        'empty' => 'No requests found.',
        'detail' => [
            'title' => 'Request Detail',
            'status' => 'Status',
            'request' => 'Request',
            'response' => 'Response',
            'error' => 'Error',
            'replay_command' => 'Replay Command',
            'copy' => 'Copy',
            'copied' => 'Copied',
            'metadata' => [
                'request_id' => 'Request ID',
                'duration' => 'Duration',
                'ip' => 'IP',
                'time' => 'Time',
            ],
        ],
        'empty_detail' => 'Select a request to view details and generate replay commands.',
        'status_ok' => 'OK',
        'status_error' => 'Error',
    ],

    // Playground
    'playground' => [
        'title' => 'Playground',
        'description' => 'Test MCP tools interactively and execute requests live.',

        // Authentication section
        'auth' => [
            'title' => 'Authentication',
            'api_key_label' => 'API Key',
            'api_key_placeholder' => 'hk_xxxxxxxx_xxxxxxxxxxxx...',
            'api_key_description' => 'Paste your API key to execute requests live',
            'validate' => 'Validate Key',
            'status' => [
                'valid' => 'Valid',
                'invalid' => 'Invalid key',
                'expired' => 'Expired',
                'empty' => 'Enter a key to validate',
            ],
            'key_info' => [
                'name' => 'Name',
                'workspace' => 'Workspace',
                'scopes' => 'Scopes',
                'last_used' => 'Last used',
            ],
            'sign_in_prompt' => 'Sign in',
            'sign_in_description' => 'to create API keys, or paste an existing key above.',
        ],

        // Tool selection section
        'tools' => [
            'title' => 'Select Tool',
            'server_label' => 'Server',
            'server_placeholder' => 'Choose a server...',
            'tool_label' => 'Tool',
            'tool_placeholder' => 'Choose a tool...',
            'arguments' => 'Arguments',
            'no_arguments' => 'This tool has no arguments.',
            'execute' => 'Execute Request',
            'generate' => 'Generate Request',
            'executing' => 'Executing...',
        ],

        // Response section
        'response' => [
            'title' => 'Response',
            'copy' => 'Copy',
            'copied' => 'Copied',
            'empty' => 'Select a server and tool to get started.',
        ],

        // API Reference section
        'reference' => [
            'title' => 'API Reference',
            'endpoint' => 'Endpoint',
            'method' => 'Method',
            'auth' => 'Auth',
            'content_type' => 'Content-Type',
            'manage_keys' => 'Manage API Keys',
        ],
    ],

    // Common
    'common' => [
        'na' => 'N/A',
    ],
];
