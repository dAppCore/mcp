<?php

declare(strict_types=1);

/*
|--------------------------------------------------------------------------
| Pest Configuration
|--------------------------------------------------------------------------
|
| Configure Pest testing framework for the core-mcp package.
| This file binds test traits to test cases and provides helper functions.
|
*/

use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;

/*
|--------------------------------------------------------------------------
| Test Case
|--------------------------------------------------------------------------
|
| The closure passed to the "uses()" method binds an abstract test case
| to all Feature and Unit tests. The TestCase class provides a bridge
| between Laravel's testing utilities and Pest's expressive syntax.
|
*/

// __DIR__, not bare names: Pest resolves a bare 'Feature' against its default
// test path (./tests), but this suite lives at php/tests. Unanchored, nothing
// matched, so no TestCase was bound, no Testbench app booted, and every test
// would have died on a null application had the suite ever been run.
uses(TestCase::class)->in(
    __DIR__.'/Feature',
    __DIR__.'/Unit',
    __DIR__.'/../src/Mcp/Tests/Unit',
);

/*
|--------------------------------------------------------------------------
| Database Refresh
|--------------------------------------------------------------------------
|
| Apply RefreshDatabase to Feature tests that need a clean database state.
| Unit tests typically don't require database access.
|
*/

uses(RefreshDatabase::class)->in(
    __DIR__.'/Feature',
    __DIR__.'/../src/Mcp/Tests/Unit',
);
