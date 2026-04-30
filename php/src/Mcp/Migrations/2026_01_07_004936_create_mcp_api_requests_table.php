<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_api_requests')) {
            Schema::create('mcp_api_requests', function (Blueprint $table) {
                $table->id();
                $table->string('request_id', 32)->unique();
                $table->foreignId('workspace_id')->nullable()->constrained('workspaces')->nullOnDelete();
                $table->foreignId('api_key_id')->nullable()->constrained('api_keys')->nullOnDelete();
                $table->string('method', 10);
                $table->string('path', 255);
                $table->json('headers')->nullable();
                $table->json('request_body')->nullable();
                $table->unsignedSmallInteger('response_status');
                $table->json('response_body')->nullable();
                $table->unsignedInteger('duration_ms')->default(0);
                $table->string('server_id', 64)->nullable();
                $table->string('tool_name', 128)->nullable();
                $table->text('error_message')->nullable();
                $table->string('ip_address', 45)->nullable();
                $table->timestamps();

                $table->index(['workspace_id', 'created_at']);
                $table->index(['server_id', 'tool_name']);
                $table->index('created_at');
                $table->index('response_status');
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('mcp_api_requests');
    }
};
