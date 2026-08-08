<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Tests\Unit;

use Core\Mcp\Models\McpToolVersion;
use Tests\TestCase;

/**
 * Covers the sortable version key that replaced the MySQL-only
 * SUBSTRING_INDEX ordering in scopeOrderByVersion.
 *
 * The key is only ever compared, never displayed, so what matters is that
 * sorting it as a plain string reproduces semver precedence.
 */
class VersionSortKeyTest extends TestCase
{
    public function test_version_sort_key_good_pads_each_component(): void
    {
        $this->assertSame('00001.00002.00003.~', McpToolVersion::versionSortKey('1.2.3'));
        $this->assertSame('00010.00000.00000.~', McpToolVersion::versionSortKey('10.0.0'));
    }

    public function test_version_sort_key_good_orders_semver_as_plain_strings(): void
    {
        $versions = ['2.0.0', '1.0.0', '10.0.0', '9.9.9', '1.1.0', '0.0.1'];

        $keyed = [];
        foreach ($versions as $version) {
            $keyed[$version] = McpToolVersion::versionSortKey($version);
        }
        asort($keyed);

        // Without the zero padding this is the case that breaks: "10.0.0"
        // sorts before "9.9.9" on a naive string comparison.
        $this->assertSame(
            ['0.0.1', '1.0.0', '1.1.0', '2.0.0', '9.9.9', '10.0.0'],
            array_keys($keyed),
        );
    }

    public function test_version_sort_key_good_ranks_a_release_above_its_prereleases(): void
    {
        $keyed = [
            '1.0.0' => McpToolVersion::versionSortKey('1.0.0'),
            '1.0.0-beta' => McpToolVersion::versionSortKey('1.0.0-beta'),
            '1.0.0-alpha' => McpToolVersion::versionSortKey('1.0.0-alpha'),
        ];
        asort($keyed);

        // semver: a pre-release precedes the release it leads to. "~" is 0x7E,
        // above every alphanumeric, which is what puts the bare release last.
        $this->assertSame(['1.0.0-alpha', '1.0.0-beta', '1.0.0'], array_keys($keyed));
    }

    public function test_version_sort_key_bad_treats_missing_components_as_zero(): void
    {
        $this->assertSame('00001.00000.00000.~', McpToolVersion::versionSortKey('1'));
        $this->assertSame('00001.00002.00000.~', McpToolVersion::versionSortKey('1.2'));
    }

    public function test_version_sort_key_ugly_still_returns_a_key_for_nonsense(): void
    {
        // A malformed version must not throw on write — a comparable key that
        // sorts low is better than a failed save.
        $this->assertSame('00000.00000.00000.~', McpToolVersion::versionSortKey(''));

        // Non-numeric components resolve to zero, and anything after the first
        // "-" is still read as a pre-release suffix, so garbage sorts below
        // every real release rather than blowing up.
        $this->assertSame('00000.00000.00000.a.version', McpToolVersion::versionSortKey('not-a.version'));
        $this->assertSame('00000.00000.00000.~', McpToolVersion::versionSortKey('rubbish'));

        $this->assertLessThan(
            McpToolVersion::versionSortKey('0.0.1'),
            McpToolVersion::versionSortKey('rubbish'),
        );
    }

    public function test_version_sort_key_good_is_written_on_save(): void
    {
        $version = McpToolVersion::create([
            'server_id' => 'test-server',
            'tool_name' => 'test_tool',
            'version' => '3.4.5',
        ]);

        $this->assertSame('00003.00004.00005.~', $version->fresh()->version_sort);

        // Derived, so it follows a corrected version rather than going stale.
        $version->update(['version' => '4.0.0']);

        $this->assertSame('00004.00000.00000.~', $version->fresh()->version_sort);
    }
}
