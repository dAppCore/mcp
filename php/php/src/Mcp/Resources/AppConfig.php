<?php

namespace Core\Mcp\Resources;

use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Resource;

class AppConfig extends Resource
{
    protected string $description = 'Application configuration for Host Hub';

    public function handle(Request $request): Response
    {
        $config = [
            'name' => config('app.name'),
            'env' => config('app.env'),
            'debug' => config('app.debug'),
            'url' => config('app.url'),
        ];

        return Response::text(json_encode($config, JSON_PRETTY_PRINT));
    }
}
