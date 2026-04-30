<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_audit_logs')) {
            Schema::create('mcp_audit_logs', function (Blueprint $table) {
                $table->id();

                // Tool execution details
                $table->string('server_id')->index();
                $table->string('tool_name')->index();
                $table->unsignedBigInteger('workspace_id')->nullable()->index();
                $table->string('session_id')->nullable()->index();

                // Input/output (stored as JSON, may be redacted)
                $table->json('input_params')->nullable();
                $table->json('output_summary')->nullable();
                $table->boolean('success')->default(true);
                $table->unsignedInteger('duration_ms')->nullable();
                $table->string('error_code')->nullable();
                $table->text('error_message')->nullable();

                // Actor information
                $table->string('actor_type')->nullable(); // user, api_key, system
                $table->unsignedBigInteger('actor_id')->nullable();
                $table->string('actor_ip', 45)->nullable(); // IPv4 or IPv6

                // Sensitive tool flagging
                $table->boolean('is_sensitive')->default(false)->index();
                $table->string('sensitivity_reason')->nullable();

                // Hash chain for tamper detection
                $table->string('previous_hash', 64)->nullable(); // SHA-256 of previous entry
                $table->string('entry_hash', 64)->index(); // SHA-256 of this entry

                // Agent context
                $table->string('agent_type')->nullable();
                $table->string('plan_slug')->nullable();

                // Timestamps (immutable - no updated_at updates after creation)
                $table->timestamp('created_at')->useCurrent();
                $table->timestamp('updated_at')->nullable();

                // Foreign key constraint
                $table->foreign('workspace_id')
                    ->references('id')
                    ->on('workspaces')
                    ->nullOnDelete();

                // Composite indexes for common queries
                $table->index(['workspace_id', 'created_at']);
                $table->index(['tool_name', 'created_at']);
                $table->index(['is_sensitive', 'created_at']);
                $table->index(['actor_type', 'actor_id']);
            });
        }

        // Table for tracking sensitive tool definitions
        if (! Schema::hasTable('mcp_sensitive_tools')) {
            Schema::create('mcp_sensitive_tools', function (Blueprint $table) {
                $table->id();
                $table->string('tool_name')->unique();
                $table->string('reason');
                $table->json('redact_fields')->nullable(); // Fields to redact in audit logs
                $table->boolean('require_explicit_consent')->default(false);
                $table->timestamps();
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('mcp_sensitive_tools');
        Schema::dropIfExists('mcp_audit_logs');
    }
};
