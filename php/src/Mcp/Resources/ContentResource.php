<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Resources;

use Core\Mod\Content\Models\ContentItem;
use Core\Tenant\Models\Workspace;
use Laravel\Mcp\Request;
use Laravel\Mcp\Response;
use Laravel\Mcp\Server\Resource;
use Symfony\Component\Yaml\Yaml;

// Not final: resolveWorkspace / resolveContentItem / listResources are
// deliberate protected seams — the whole point is that a caller (and the
// test suite) substitutes them to run this resource without a database.
// `final` contradicted that and made ContentResourceTest a fatal error.
class ContentResource extends Resource
{
    protected string $description = 'Content items from the CMS - returns markdown for AI context';

    public function handle(Request $request): Response
    {
        $uri = (string) $request->get('uri', '');
        $parts = $this->parseUri($uri);

        if ($parts === null) {
            return Response::text('Invalid URI format. Expected: content://{workspace}/{slug}');
        }

        [$workspaceIdentifier, $contentIdentifier] = $parts;
        $workspace = $this->resolveWorkspace($workspaceIdentifier);

        if ($workspace === null) {
            return Response::text(sprintf('Workspace not found: %s', $workspaceIdentifier));
        }

        $item = $this->resolveContentItem($workspace, $contentIdentifier);

        if ($item === null) {
            return Response::text(sprintf('Content not found: %s', $contentIdentifier));
        }

        return Response::text($this->contentToMarkdown($item, $workspace));
    }

    public static function list(): array
    {
        return (new self)->listResources();
    }

    protected function listResources(): array
    {
        if (! class_exists(Workspace::class) || ! class_exists(ContentItem::class)) {
            return [];
        }

        $resources = [];

        foreach (Workspace::query()->get(['id', 'slug']) as $workspace) {
            foreach ($this->publishedItemsForWorkspace($workspace)->take(50) as $item) {
                $resources[] = [
                    'uri' => sprintf('content://%s/%s', $workspace->slug, $item->slug),
                    'name' => (string) $item->title,
                    'description' => sprintf('%s: %s', ucfirst((string) ($item->type ?? 'content')), (string) $item->title),
                    'mimeType' => 'text/markdown',
                ];
            }
        }

        return $resources;
    }

    protected function parseUri(string $uri): ?array
    {
        if (! str_starts_with($uri, 'content://')) {
            return null;
        }

        $parts = explode('/', substr($uri, 10), 2);

        return count($parts) === 2 ? $parts : null;
    }

    // ?object, matching resolveContentItem: Workspace lives in the tenant
    // package, which is optional here (see the class_exists guard), so the
    // seam cannot narrow to a type that may not be loaded.
    protected function resolveWorkspace(string $identifier): ?object
    {
        if (! class_exists(Workspace::class)) {
            return null;
        }

        return Workspace::query()
            ->where('slug', $identifier)
            ->orWhere('id', $identifier)
            ->first();
    }

    protected function resolveContentItem(object $workspace, string $identifier): ?object
    {
        if (! class_exists(ContentItem::class)) {
            return null;
        }

        $query = ContentItem::query();

        if (method_exists($query->getModel(), 'scopeForWorkspace')) {
            $query->forWorkspace($workspace->id);
        } else {
            $query->where('workspace_id', $workspace->id);
        }

        if (method_exists($query->getModel(), 'scopeNative')) {
            $query->native();
        }

        $item = (clone $query)->where('slug', $identifier)->first();

        if ($item === null && is_numeric($identifier)) {
            $item = (clone $query)->find((int) $identifier);
        }

        if ($item !== null && method_exists($item, 'loadMissing')) {
            $item->loadMissing(['author', 'categories', 'tags', 'taxonomies']);
        }

        return $item;
    }

    protected function publishedItemsForWorkspace(object $workspace)
    {
        $query = ContentItem::query();

        if (method_exists($query->getModel(), 'scopeForWorkspace')) {
            $query->forWorkspace($workspace->id);
        } else {
            $query->where('workspace_id', $workspace->id);
        }

        if (method_exists($query->getModel(), 'scopeNative')) {
            $query->native();
        }

        if (method_exists($query->getModel(), 'scopePublished')) {
            $query->published();
        } else {
            $query->where('status', 'publish');
        }

        return $query
            ->orderByDesc('updated_at')
            ->limit(50)
            ->get(['id', 'slug', 'title', 'type']);
    }

    protected function contentToMarkdown(object $item, object $workspace): string
    {
        $frontMatter = [
            'title' => (string) ($item->title ?? ''),
            'slug' => (string) ($item->slug ?? ''),
            'workspace' => (string) ($workspace->slug ?? $workspace->id ?? ''),
            'type' => (string) ($item->type ?? ''),
            'status' => (string) ($item->status ?? ''),
            'author' => data_get($item, 'author.name'),
            'categories' => $this->taxonomyNames($item, 'categories', 'category'),
            'tags' => $this->taxonomyNames($item, 'tags', 'tag'),
            'publish_at' => $item->publish_at?->toIso8601String(),
            'created_at' => $item->created_at?->toIso8601String(),
            'updated_at' => $item->updated_at?->toIso8601String(),
            'seo_title' => data_get($item, 'seo_meta.title') ?? data_get($item, 'seo_title'),
            'seo_description' => data_get($item, 'seo_meta.description') ?? data_get($item, 'seo_description'),
        ];

        $frontMatter = array_filter($frontMatter, static fn (mixed $value): bool => $value !== null && $value !== []);

        $markdown = "---\n".Yaml::dump($frontMatter, 3, 2)."---\n\n";
        $excerpt = trim((string) ($item->excerpt ?? ''));

        if ($excerpt !== '') {
            $markdown .= collect(preg_split('/\R/', $excerpt) ?: [])
                ->map(static fn (string $line): string => '> '.$line)
                ->implode("\n")."\n\n";
        }

        $markdown .= $this->contentBody($item);

        return $markdown;
    }

    protected function taxonomyNames(object $item, string $relation, string $fallbackType): array
    {
        $names = collect(data_get($item, $relation, []))
            ->map(static fn (mixed $taxonomy): ?string => is_object($taxonomy) ? ($taxonomy->name ?? null) : null)
            ->filter()
            ->values()
            ->all();

        if ($names !== []) {
            return $names;
        }

        return collect(data_get($item, 'taxonomies', []))
            ->filter(static fn (mixed $taxonomy): bool => is_object($taxonomy) && (($taxonomy->type ?? null) === $fallbackType || ($taxonomy->taxonomy ?? null) === $fallbackType))
            ->map(static fn (object $taxonomy): ?string => $taxonomy->name ?? null)
            ->filter()
            ->values()
            ->all();
    }

    protected function contentBody(object $item): string
    {
        $markdown = trim((string) ($item->content_markdown ?? ''));

        if ($markdown !== '') {
            return $markdown;
        }

        $cleanHtml = trim((string) ($item->content_html_clean ?? ''));

        if ($cleanHtml !== '') {
            return trim(strip_tags($cleanHtml));
        }

        return trim(strip_tags((string) ($item->content_html_original ?? '')));
    }
}
