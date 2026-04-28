// SPDX-License-Identifier: EUPL-1.2

package mcp

import "testing"

func ax7NewServiceForTest(t testing.TB, opts Options) *Service {
	t.Helper()
	svc, err := New(opts)
	if err != nil {
		t.Fatalf("New(%+v) failed: %v", opts, err)
	}
	return svc
}
