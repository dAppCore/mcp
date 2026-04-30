package agentic

import (
	. "dappco.re/go"
)

// moved AX-7 triplet TestCoreioCompat_CoreFS_Read_Good
func TestCoreioCompat_CoreFS_Read_Good(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	got, err := fs.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "ok", got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Read_Bad
func TestCoreioCompat_CoreFS_Read_Bad(t *T) {
	fs := agenticFSForTest(t)
	got, err := fs.Read("missing")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Read_Ugly
func TestCoreioCompat_CoreFS_Read_Ugly(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("empty.txt", "").OK)
	got, err := fs.Read("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_EnsureDir_Good
func TestCoreioCompat_CoreFS_EnsureDir_Good(t *T) {
	fs := agenticFSForTest(t)
	AssertNoError(t, fs.EnsureDir("a/b"))
	AssertTrue(t, fs.IsFile("a/b") == false)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_EnsureDir_Bad
func TestCoreioCompat_CoreFS_EnsureDir_Bad(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("file", "ok").OK)
	err := fs.EnsureDir("file")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_EnsureDir_Ugly
func TestCoreioCompat_CoreFS_EnsureDir_Ugly(t *T) {
	fs := agenticFSForTest(t)
	AssertNoError(t, fs.EnsureDir("a/b"))
	AssertNoError(t, fs.EnsureDir("a/b"))
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_List_Good
func TestCoreioCompat_CoreFS_List_Good(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	got, err := fs.List(".")
	AssertNoError(t, err)
	AssertTrue(t, len(got) > 0)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_List_Bad
func TestCoreioCompat_CoreFS_List_Bad(t *T) {
	fs := agenticFSForTest(t)
	got, err := fs.List("missing")
	AssertError(t, err)
	AssertNil(t, got)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_List_Ugly
func TestCoreioCompat_CoreFS_List_Ugly(t *T) {
	fs := agenticFSForTest(t)
	AssertNoError(t, fs.EnsureDir("empty"))
	got, err := fs.List("empty")
	AssertNoError(t, err)
	AssertLen(t, got, 0)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_IsFile_Good
func TestCoreioCompat_CoreFS_IsFile_Good(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	AssertTrue(t, fs.IsFile("a.txt"))
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_IsFile_Bad
func TestCoreioCompat_CoreFS_IsFile_Bad(t *T) {
	fs := agenticFSForTest(t)
	AssertFalse(t, fs.IsFile("missing"))
	AssertFalse(t, fs.IsFile(""))
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_IsFile_Ugly
func TestCoreioCompat_CoreFS_IsFile_Ugly(t *T) {
	fs := agenticFSForTest(t)
	AssertNoError(t, fs.EnsureDir("dir"))
	AssertFalse(t, fs.IsFile("dir"))
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Delete_Good
func TestCoreioCompat_CoreFS_Delete_Good(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("a.txt", "ok").OK)
	AssertNoError(t, fs.Delete("a.txt"))
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Delete_Bad
func TestCoreioCompat_CoreFS_Delete_Bad(t *T) {
	fs := agenticFSForTest(t)
	err := fs.Delete("missing")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreioCompat_CoreFS_Delete_Ugly
func TestCoreioCompat_CoreFS_Delete_Ugly(t *T) {
	fs := agenticFSForTest(t)
	AssertTrue(t, fs.fs.Write("nested/a.txt", "").OK)
	AssertNoError(t, fs.Delete("nested/a.txt"))
}
