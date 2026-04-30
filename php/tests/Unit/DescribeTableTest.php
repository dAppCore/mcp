<?php

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Tools\DescribeTable;
use Illuminate\Support\Facades\Config;
use Illuminate\Support\Facades\DB;
use Laravel\Mcp\Request;
use Mockery;
use Tests\TestCase;

class DescribeTableTest extends TestCase
{
    protected function tearDown(): void
    {
        Mockery::close();
        parent::tearDown();
    }

    public function test_handle_returns_columns_and_indexes_for_a_table(): void
    {
        DB::shouldReceive('select')
            ->once()
            ->with('SHOW FULL COLUMNS FROM `users`')
            ->andReturn([
                (object) [
                    'Field' => 'id',
                    'Type' => 'bigint unsigned',
                    'Null' => 'NO',
                    'Key' => 'PRI',
                    'Default' => null,
                    'Extra' => 'auto_increment',
                    'Privileges' => 'select,insert,update,references',
                    'Comment' => 'Primary key',
                ],
                (object) [
                    'Field' => 'email',
                    'Type' => 'varchar(255)',
                    'Null' => 'NO',
                    'Key' => 'UNI',
                    'Default' => null,
                    'Extra' => '',
                    'Privileges' => 'select,insert,update,references',
                    'Comment' => '',
                ],
            ]);

        DB::shouldReceive('select')
            ->once()
            ->with('SHOW INDEX FROM `users`')
            ->andReturn([
                (object) [
                    'Key_name' => 'PRIMARY',
                    'Non_unique' => 0,
                    'Index_type' => 'BTREE',
                    'Column_name' => 'id',
                    'Seq_in_index' => 1,
                    'Collation' => 'A',
                    'Cardinality' => 1,
                    'Sub_part' => null,
                    'Null' => '',
                    'Comment' => '',
                ],
                (object) [
                    'Key_name' => 'users_email_unique',
                    'Non_unique' => 0,
                    'Index_type' => 'BTREE',
                    'Column_name' => 'email',
                    'Seq_in_index' => 1,
                    'Collation' => 'A',
                    'Cardinality' => 1,
                    'Sub_part' => null,
                    'Null' => '',
                    'Comment' => '',
                ],
            ]);

        $tool = new DescribeTable();
        $response = $tool->handle(new Request(['table' => 'users']));
        $data = json_decode($response->getContent(), true, flags: JSON_THROW_ON_ERROR);

        $this->assertSame('users', $data['table']);
        $this->assertCount(2, $data['columns']);
        $this->assertSame('id', $data['columns'][0]['field']);
        $this->assertSame('bigint unsigned', $data['columns'][0]['type']);
        $this->assertSame('PRIMARY', $data['indexes'][0]['name']);
        $this->assertSame(['id'], array_column($data['indexes'][0]['columns'], 'name'));
    }

    public function test_handle_rejects_invalid_table_names(): void
    {
        $tool = new DescribeTable();
        $response = $tool->handle(new Request(['table' => 'users; DROP TABLE users']));
        $data = json_decode($response->getContent(), true, flags: JSON_THROW_ON_ERROR);

        $this->assertSame('VALIDATION_ERROR', $data['code']);
        $this->assertStringContainsString('Invalid table name', $data['error']);
    }

    public function test_handle_blocks_system_tables(): void
    {
        Config::set('mcp.database.blocked_tables', []);

        $tool = new DescribeTable();
        $response = $tool->handle(new Request(['table' => 'information_schema']));
        $data = json_decode($response->getContent(), true, flags: JSON_THROW_ON_ERROR);

        $this->assertSame('VALIDATION_ERROR', $data['code']);
        $this->assertStringContainsString('not permitted', $data['error']);
    }
}
