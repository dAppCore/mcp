<?php

namespace Core\Mcp\Tools;

use Illuminate\Contracts\JsonSchema\JsonSchema;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Tool;

class ListSites extends Tool
{
    protected string $description = 'List all sites managed by Host Hub';

    public function handle(Request $request): Response
    {
        $sites = [
            ['name' => 'BioHost', 'domain' => 'link.host.uk.com', 'type' => 'WordPress'],
            ['name' => 'SocialHost', 'domain' => 'social.host.uk.com', 'type' => 'Laravel'],
            ['name' => 'AnalyticsHost', 'domain' => 'analytics.host.uk.com', 'type' => 'Node.js'],
            ['name' => 'TrustHost', 'domain' => 'trust.host.uk.com', 'type' => 'WordPress'],
            ['name' => 'NotifyHost', 'domain' => 'notify.host.uk.com', 'type' => 'Go'],
            ['name' => 'MailHost', 'domain' => 'hostmail.cc', 'type' => 'MailCow'],
        ];

        return Response::text(json_encode($sites, JSON_PRETTY_PRINT));
    }

    public function schema(JsonSchema $schema): array
    {
        return [];
    }
}
