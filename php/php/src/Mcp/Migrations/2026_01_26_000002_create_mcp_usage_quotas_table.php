<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_usage_quotas')) {
            Schema::create('mcp_usage_quotas', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->constrained('workspaces')->cascadeOnDelete();
                $table->string('month', 7); // YYYY-MM format
                $table->unsignedBigInteger('tool_calls_count')->default(0);
                $table->unsignedBigInteger('input_tokens')->default(0);
                $table->unsignedBigInteger('output_tokens')->default(0);
                $table->timestamps();

                $table->unique(['workspace_id', 'month']);
                $table->index('month');
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('mcp_usage_quotas');
    }
};
