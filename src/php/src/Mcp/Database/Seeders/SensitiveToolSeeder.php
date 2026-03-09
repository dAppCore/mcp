<?php

declare(strict_types=1);

namespace Core\Mcp\Database\Seeders;

use Core\Mcp\Models\McpSensitiveTool;
use Illuminate\Database\Seeder;

/**
 * Seeds default sensitive tool definitions.
 *
 * These tools require stricter auditing due to their potential
 * impact on security, privacy, or critical operations.
 */
class SensitiveToolSeeder extends Seeder
{
    /**
     * Run the database seeds.
     */
    public function run(): void
    {
        $sensitiveTools = [
            // Database operations
            [
                'tool_name' => 'query_database',
                'reason' => 'Direct database access - may expose sensitive data',
                'redact_fields' => ['password', 'email', 'phone', 'address', 'ssn'],
                'require_explicit_consent' => false,
            ],

            // User management
            [
                'tool_name' => 'create_user',
                'reason' => 'User account creation - security sensitive',
                'redact_fields' => ['password', 'secret'],
                'require_explicit_consent' => true,
            ],
            [
                'tool_name' => 'update_user',
                'reason' => 'User account modification - security sensitive',
                'redact_fields' => ['password', 'secret', 'email'],
                'require_explicit_consent' => true,
            ],
            [
                'tool_name' => 'delete_user',
                'reason' => 'User account deletion - irreversible operation',
                'redact_fields' => [],
                'require_explicit_consent' => true,
            ],

            // API key management
            [
                'tool_name' => 'create_api_key',
                'reason' => 'API key creation - security credential',
                'redact_fields' => ['key', 'secret', 'token'],
                'require_explicit_consent' => true,
            ],
            [
                'tool_name' => 'revoke_api_key',
                'reason' => 'API key revocation - access control',
                'redact_fields' => [],
                'require_explicit_consent' => true,
            ],

            // Billing and financial
            [
                'tool_name' => 'upgrade_plan',
                'reason' => 'Plan upgrade - financial impact',
                'redact_fields' => ['card_number', 'cvv', 'payment_method'],
                'require_explicit_consent' => true,
            ],
            [
                'tool_name' => 'create_coupon',
                'reason' => 'Coupon creation - financial impact',
                'redact_fields' => [],
                'require_explicit_consent' => false,
            ],
            [
                'tool_name' => 'process_refund',
                'reason' => 'Refund processing - financial transaction',
                'redact_fields' => ['card_number', 'bank_account'],
                'require_explicit_consent' => true,
            ],

            // Content operations
            [
                'tool_name' => 'delete_content',
                'reason' => 'Content deletion - irreversible data loss',
                'redact_fields' => [],
                'require_explicit_consent' => true,
            ],
            [
                'tool_name' => 'publish_content',
                'reason' => 'Public content publishing - visibility impact',
                'redact_fields' => [],
                'require_explicit_consent' => false,
            ],

            // System configuration
            [
                'tool_name' => 'update_config',
                'reason' => 'System configuration change - affects application behaviour',
                'redact_fields' => ['api_key', 'secret', 'password'],
                'require_explicit_consent' => true,
            ],

            // Webhook management
            [
                'tool_name' => 'create_webhook',
                'reason' => 'External webhook creation - data exfiltration risk',
                'redact_fields' => ['secret', 'token'],
                'require_explicit_consent' => true,
            ],
        ];

        foreach ($sensitiveTools as $tool) {
            McpSensitiveTool::updateOrCreate(
                ['tool_name' => $tool['tool_name']],
                [
                    'reason' => $tool['reason'],
                    'redact_fields' => $tool['redact_fields'],
                    'require_explicit_consent' => $tool['require_explicit_consent'],
                ]
            );
        }

        $this->command->info('Registered '.count($sensitiveTools).' sensitive tool definitions.');
    }
}
