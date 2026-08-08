<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Resources;

use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Resource;

final class AppConfig extends Resource
{
    protected string $description = 'Application configuration for Host Hub';

    public function handle(Request $request): Response
    {
        return Response::text((string) json_encode([
            'name' => config('app.name'),
            'env' => config('app.env'),
            'debug' => config('app.debug'),
            'url' => config('app.url'),
        ], JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES));
    }
}
