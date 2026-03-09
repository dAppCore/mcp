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

uses(TestCase::class)->in('Feature', 'Unit', '../src/Mcp/Tests/Unit');

/*
|--------------------------------------------------------------------------
| Database Refresh
|--------------------------------------------------------------------------
|
| Apply RefreshDatabase to Feature tests that need a clean database state.
| Unit tests typically don't require database access.
|
*/

uses(RefreshDatabase::class)->in('Feature', '../src/Mcp/Tests/Unit');
