<?php

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Controllers\McpApiController;
use Core\Mcp\Services\OpenApiGenerator;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Tests\TestCase;

class McpResourceListTest extends TestCase
{
    public function test_resources_endpoint_returns_server_resources(): void
    {
        $controller = new class extends McpApiController {
            protected function loadServerFull(string $id): ?array
            {
                if ($id !== 'demo-server') {
                    return null;
                }

                return [
                    'id' => 'demo-server',
                    'resources' => [
                        [
                            'uri' => 'content://workspace/article',
                            'name' => 'Article',
                            'description' => 'Published article',
                            'mimeType' => 'text/markdown',
                        ],
                        [
                            'uri' => 'plans://all',
                            'name' => 'Plans',
                            'description' => 'Work plan index',
                            'mimeType' => 'text/markdown',
                        ],
                    ],
                ];
            }
        };

        $response = $controller->resources(Request::create('/api/v1/mcp/servers/demo-server/resources', 'GET'), 'demo-server');

        $this->assertInstanceOf(JsonResponse::class, $response);
        $this->assertSame(200, $response->getStatusCode());

        $data = $response->getData(true);
        $this->assertSame('demo-server', $data['server']);
        $this->assertSame(2, $data['count']);
        $this->assertSame('content://workspace/article', $data['resources'][0]['uri']);
        $this->assertSame('plans://all', $data['resources'][1]['uri']);
    }

    public function test_openapi_includes_resource_list_endpoint(): void
    {
        $schema = (new OpenApiGenerator)->generate();

        $this->assertArrayHasKey('/servers/{serverId}/resources', $schema['paths']);
        $this->assertArrayHasKey('ResourceList', $schema['components']['schemas']);
        $this->assertSame(
            '#/components/schemas/ResourceList',
            $schema['paths']['/servers/{serverId}/resources']['get']['responses']['200']['content']['application/json']['schema']['$ref']
        );
    }
}
