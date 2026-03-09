<?php

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Services\ToolVersionService;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Carbon;
use Tests\TestCase;

class ToolVersionServiceTest extends TestCase
{
    use RefreshDatabase;

    protected ToolVersionService $service;

    protected function setUp(): void
    {
        parent::setUp();
        $this->service = new ToolVersionService;
    }

    public function test_can_register_new_version(): void
    {
        $version = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
            inputSchema: ['type' => 'object', 'properties' => ['query' => ['type' => 'string']]],
            description: 'A test tool',
            options: ['mark_latest' => true]
        );

        $this->assertSame('test-server', $version->server_id);
        $this->assertSame('test_tool', $version->tool_name);
        $this->assertSame('1.0.0', $version->version);
        $this->assertTrue($version->is_latest);
    }

    public function test_first_version_is_automatically_latest(): void
    {
        $version = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
        );

        $this->assertTrue($version->is_latest);
    }

    public function test_can_get_tool_at_specific_version(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $this->service->registerVersion('test-server', 'test_tool', '2.0.0');

        $v1 = $this->service->getToolAtVersion('test-server', 'test_tool', '1.0.0');
        $v2 = $this->service->getToolAtVersion('test-server', 'test_tool', '2.0.0');

        $this->assertSame('1.0.0', $v1->version);
        $this->assertSame('2.0.0', $v2->version);
    }

    public function test_get_latest_version(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $v2 = $this->service->registerVersion('test-server', 'test_tool', '2.0.0', options: ['mark_latest' => true]);

        $latest = $this->service->getLatestVersion('test-server', 'test_tool');

        $this->assertSame('2.0.0', $latest->version);
        $this->assertTrue($latest->is_latest);
    }

    public function test_resolve_version_returns_latest_when_no_version_specified(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $this->service->registerVersion('test-server', 'test_tool', '2.0.0', options: ['mark_latest' => true]);

        $result = $this->service->resolveVersion('test-server', 'test_tool', null);

        $this->assertNotNull($result['version']);
        $this->assertSame('2.0.0', $result['version']->version);
        $this->assertNull($result['warning']);
        $this->assertNull($result['error']);
    }

    public function test_resolve_version_returns_specific_version(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $this->service->registerVersion('test-server', 'test_tool', '2.0.0', options: ['mark_latest' => true]);

        $result = $this->service->resolveVersion('test-server', 'test_tool', '1.0.0');

        $this->assertNotNull($result['version']);
        $this->assertSame('1.0.0', $result['version']->version);
    }

    public function test_resolve_version_returns_error_for_nonexistent_version(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');

        $result = $this->service->resolveVersion('test-server', 'test_tool', '9.9.9');

        $this->assertNull($result['version']);
        $this->assertNotNull($result['error']);
        $this->assertSame('VERSION_NOT_FOUND', $result['error']['code']);
    }

    public function test_resolve_deprecated_version_returns_warning(): void
    {
        $version = $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $version->deprecate(Carbon::now()->addDays(30));

        $this->service->registerVersion('test-server', 'test_tool', '2.0.0', options: ['mark_latest' => true]);

        $result = $this->service->resolveVersion('test-server', 'test_tool', '1.0.0');

        $this->assertNotNull($result['version']);
        $this->assertNotNull($result['warning']);
        $this->assertSame('TOOL_VERSION_DEPRECATED', $result['warning']['code']);
    }

    public function test_resolve_sunset_version_returns_error(): void
    {
        $version = $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $version->deprecated_at = Carbon::now()->subDays(60);
        $version->sunset_at = Carbon::now()->subDays(30);
        $version->save();

        $this->service->registerVersion('test-server', 'test_tool', '2.0.0', options: ['mark_latest' => true]);

        $result = $this->service->resolveVersion('test-server', 'test_tool', '1.0.0');

        $this->assertNull($result['version']);
        $this->assertNotNull($result['error']);
        $this->assertSame('TOOL_VERSION_SUNSET', $result['error']['code']);
    }

    public function test_is_deprecated(): void
    {
        $version = $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $version->deprecate();

        $this->assertTrue($this->service->isDeprecated('test-server', 'test_tool', '1.0.0'));
    }

    public function test_is_sunset(): void
    {
        $version = $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $version->deprecated_at = Carbon::now()->subDays(60);
        $version->sunset_at = Carbon::now()->subDays(30);
        $version->save();

        $this->assertTrue($this->service->isSunset('test-server', 'test_tool', '1.0.0'));
    }

    public function test_compare_versions(): void
    {
        $this->assertSame(-1, $this->service->compareVersions('1.0.0', '2.0.0'));
        $this->assertSame(0, $this->service->compareVersions('1.0.0', '1.0.0'));
        $this->assertSame(1, $this->service->compareVersions('2.0.0', '1.0.0'));
        $this->assertSame(-1, $this->service->compareVersions('1.0.0', '1.0.1'));
        $this->assertSame(-1, $this->service->compareVersions('1.0.0', '1.1.0'));
    }

    public function test_get_version_history(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $this->service->registerVersion('test-server', 'test_tool', '1.1.0');
        $this->service->registerVersion('test-server', 'test_tool', '2.0.0');

        $history = $this->service->getVersionHistory('test-server', 'test_tool');

        $this->assertCount(3, $history);
        // Should be ordered by version desc
        $this->assertSame('2.0.0', $history[0]->version);
        $this->assertSame('1.1.0', $history[1]->version);
        $this->assertSame('1.0.0', $history[2]->version);
    }

    public function test_migrate_tool_call(): void
    {
        $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
            inputSchema: [
                'type' => 'object',
                'properties' => ['query' => ['type' => 'string']],
                'required' => ['query'],
            ]
        );

        $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '2.0.0',
            inputSchema: [
                'type' => 'object',
                'properties' => [
                    'query' => ['type' => 'string'],
                    'limit' => ['type' => 'integer', 'default' => 10],
                ],
                'required' => ['query', 'limit'],
            ]
        );

        $result = $this->service->migrateToolCall(
            serverId: 'test-server',
            toolName: 'test_tool',
            fromVersion: '1.0.0',
            toVersion: '2.0.0',
            arguments: ['query' => 'SELECT * FROM users']
        );

        $this->assertTrue($result['success']);
        $this->assertSame('SELECT * FROM users', $result['arguments']['query']);
        $this->assertSame(10, $result['arguments']['limit']); // Default applied
    }

    public function test_deprecate_version(): void
    {
        $this->service->registerVersion('test-server', 'test_tool', '1.0.0');

        $sunsetDate = Carbon::now()->addDays(30);
        $deprecatedVersion = $this->service->deprecateVersion(
            'test-server',
            'test_tool',
            '1.0.0',
            $sunsetDate
        );

        $this->assertNotNull($deprecatedVersion->deprecated_at);
        $this->assertSame($sunsetDate->toDateString(), $deprecatedVersion->sunset_at->toDateString());
    }

    public function test_get_tools_with_versions(): void
    {
        $this->service->registerVersion('test-server', 'tool_a', '1.0.0');
        $this->service->registerVersion('test-server', 'tool_a', '2.0.0', options: ['mark_latest' => true]);
        $this->service->registerVersion('test-server', 'tool_b', '1.0.0');

        $tools = $this->service->getToolsWithVersions('test-server');

        $this->assertCount(2, $tools);
        $this->assertArrayHasKey('tool_a', $tools);
        $this->assertArrayHasKey('tool_b', $tools);
        $this->assertSame(2, $tools['tool_a']['version_count']);
        $this->assertSame(1, $tools['tool_b']['version_count']);
    }

    public function test_get_servers_with_versions(): void
    {
        $this->service->registerVersion('server-a', 'tool', '1.0.0');
        $this->service->registerVersion('server-b', 'tool', '1.0.0');

        $servers = $this->service->getServersWithVersions();

        $this->assertCount(2, $servers);
        $this->assertContains('server-a', $servers);
        $this->assertContains('server-b', $servers);
    }

    public function test_sync_from_server_config(): void
    {
        $config = [
            'id' => 'test-server',
            'tools' => [
                [
                    'name' => 'tool_a',
                    'description' => 'Tool A',
                    'inputSchema' => ['type' => 'object'],
                ],
                [
                    'name' => 'tool_b',
                    'purpose' => 'Tool B purpose',
                ],
            ],
        ];

        $registered = $this->service->syncFromServerConfig($config, '1.0.0');

        $this->assertSame(2, $registered);

        $toolA = $this->service->getToolAtVersion('test-server', 'tool_a', '1.0.0');
        $toolB = $this->service->getToolAtVersion('test-server', 'tool_b', '1.0.0');

        $this->assertNotNull($toolA);
        $this->assertNotNull($toolB);
        $this->assertSame('Tool A', $toolA->description);
        $this->assertSame('Tool B purpose', $toolB->description);
    }

    public function test_get_stats(): void
    {
        $this->service->registerVersion('server-a', 'tool_a', '1.0.0');
        $this->service->registerVersion('server-a', 'tool_a', '2.0.0');
        $this->service->registerVersion('server-b', 'tool_b', '1.0.0');

        $stats = $this->service->getStats();

        $this->assertSame(3, $stats['total_versions']);
        $this->assertSame(2, $stats['total_tools']);
        $this->assertSame(2, $stats['servers']);
    }

    public function test_invalid_semver_throws_exception(): void
    {
        $this->expectException(\InvalidArgumentException::class);
        $this->expectExceptionMessage('Invalid semver version');

        $this->service->registerVersion('test-server', 'test_tool', 'invalid');
    }

    public function test_valid_semver_formats(): void
    {
        // Basic versions
        $v1 = $this->service->registerVersion('test-server', 'tool', '1.0.0');
        $this->assertSame('1.0.0', $v1->version);

        // Prerelease
        $v2 = $this->service->registerVersion('test-server', 'tool', '2.0.0-beta');
        $this->assertSame('2.0.0-beta', $v2->version);

        // Prerelease with dots
        $v3 = $this->service->registerVersion('test-server', 'tool', '2.0.0-alpha.1');
        $this->assertSame('2.0.0-alpha.1', $v3->version);

        // Build metadata
        $v4 = $this->service->registerVersion('test-server', 'tool', '2.0.0+build.123');
        $this->assertSame('2.0.0+build.123', $v4->version);
    }

    public function test_updating_existing_version(): void
    {
        $original = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
            description: 'Original description'
        );

        $updated = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
            description: 'Updated description'
        );

        $this->assertSame($original->id, $updated->id);
        $this->assertSame('Updated description', $updated->description);
    }

    public function test_model_compare_schema_with(): void
    {
        $v1 = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
            inputSchema: [
                'type' => 'object',
                'properties' => [
                    'query' => ['type' => 'string'],
                    'format' => ['type' => 'string'],
                ],
            ]
        );

        $v2 = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '2.0.0',
            inputSchema: [
                'type' => 'object',
                'properties' => [
                    'query' => ['type' => 'string', 'maxLength' => 1000], // Changed
                    'limit' => ['type' => 'integer'], // Added
                ],
            ]
        );

        $diff = $v1->compareSchemaWith($v2);

        $this->assertContains('limit', $diff['added']);
        $this->assertContains('format', $diff['removed']);
        $this->assertArrayHasKey('query', $diff['changed']);
    }

    public function test_model_mark_as_latest(): void
    {
        $v1 = $this->service->registerVersion('test-server', 'test_tool', '1.0.0');
        $v2 = $this->service->registerVersion('test-server', 'test_tool', '2.0.0');

        $v2->markAsLatest();

        $this->assertFalse($v1->fresh()->is_latest);
        $this->assertTrue($v2->fresh()->is_latest);
    }

    public function test_model_status_attribute(): void
    {
        $version = $this->service->registerVersion('test-server', 'test_tool', '1.0.0');

        $this->assertSame('latest', $version->status);

        $version->is_latest = false;
        $version->save();
        $this->assertSame('active', $version->fresh()->status);

        $version->deprecated_at = Carbon::now()->subDay();
        $version->save();
        $this->assertSame('deprecated', $version->fresh()->status);

        $version->sunset_at = Carbon::now()->subDay();
        $version->save();
        $this->assertSame('sunset', $version->fresh()->status);
    }

    public function test_model_to_api_array(): void
    {
        $version = $this->service->registerVersion(
            serverId: 'test-server',
            toolName: 'test_tool',
            version: '1.0.0',
            inputSchema: ['type' => 'object'],
            description: 'Test tool',
            options: ['changelog' => 'Initial release']
        );

        $array = $version->toApiArray();

        $this->assertSame('test-server', $array['server_id']);
        $this->assertSame('test_tool', $array['tool_name']);
        $this->assertSame('1.0.0', $array['version']);
        $this->assertTrue($array['is_latest']);
        $this->assertSame('latest', $array['status']);
        $this->assertSame('Test tool', $array['description']);
        $this->assertSame('Initial release', $array['changelog']);
    }
}
