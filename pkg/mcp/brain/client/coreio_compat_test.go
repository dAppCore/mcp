package client

import (
	core "dappco.re/go"
)

// moved AX-7 triplet TestCoreioCompat_CoreFS_Stat_Good
func TestCoreioCompat_CoreFS_Stat_Good(t *T) {
	fs := localFSForTest(t)
	AssertNoError(t, fs.WriteMode("a.txt", "ok", core.FileMode(0o600)))
	info, err := fs.Stat("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "a.txt", info.Name())
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Stat_Bad
func TestCoreioCompat_CoreFS_Stat_Bad(t *T) {
	fs := localFSForTest(t)
	info, err := fs.Stat("missing.txt")
	AssertError(t, err)
	AssertNil(t, info)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Stat_Ugly
func TestCoreioCompat_CoreFS_Stat_Ugly(t *T) {
	fs := localFSForTest(t)
	AssertTrue(t, fs.fs.EnsureDir("dir").OK)
	info, err := fs.Stat("dir")
	AssertNoError(t, err)
	AssertTrue(t, info.IsDir())
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Read_Good
func TestCoreioCompat_CoreFS_Read_Good(t *T) {
	fs := localFSForTest(t)
	AssertNoError(t, fs.WriteMode("a.txt", "ok", core.FileMode(0o600)))
	got, err := fs.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "ok", got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Read_Bad
func TestCoreioCompat_CoreFS_Read_Bad(t *T) {
	fs := localFSForTest(t)
	got, err := fs.Read("missing.txt")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Read_Ugly
func TestCoreioCompat_CoreFS_Read_Ugly(t *T) {
	fs := localFSForTest(t)
	AssertNoError(t, fs.WriteMode("empty.txt", "", core.FileMode(0o600)))
	got, err := fs.Read("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_WriteMode_Good
func TestCoreioCompat_CoreFS_WriteMode_Good(t *T) {
	fs := localFSForTest(t)
	err := fs.WriteMode("a.txt", "ok", core.FileMode(0o600))
	AssertNoError(t, err)
	AssertTrue(t, fs.fs.IsFile("a.txt"))
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_WriteMode_Bad
func TestCoreioCompat_CoreFS_WriteMode_Bad(t *T) {
	fs := &localCoreFS{}
	AssertPanics(t, func() { _ = fs.WriteMode("a.txt", "ok", core.FileMode(0o600)) })
	AssertNil(t, fs.fs)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_WriteMode_Ugly
func TestCoreioCompat_CoreFS_WriteMode_Ugly(t *T) {
	fs := localFSForTest(t)
	AssertTrue(t, fs.fs.EnsureDir("nested").OK)
	err := fs.WriteMode("nested/a.txt", "", core.FileMode(0o600))
	AssertNoError(t, err)
	AssertTrue(t, fs.fs.IsFile("nested/a.txt"))
}
