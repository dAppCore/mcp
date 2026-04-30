package mcp

import (
	"io"
)

// moved AX-7 triplet TestCoreMedium_Medium_Append_Good
func TestCoreMedium_Medium_Append_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	w, err := m.Append("a.txt")
	AssertNoError(t, err)
	_, err = w.Write([]byte(" world"))
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
	got, err := m.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "hello world", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_Append_Bad
func TestCoreMedium_Medium_Append_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	_, err := m.Append("dir")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_Append_Ugly
func TestCoreMedium_Medium_Append_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.Append("new.txt")
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}

// moved AX-7 triplet TestCoreMedium_Medium_Create_Good
func TestCoreMedium_Medium_Create_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.Create("a.txt")
	AssertNoError(t, err)
	_, err = w.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}

// moved AX-7 triplet TestCoreMedium_Medium_Create_Bad
func TestCoreMedium_Medium_Create_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	_, err := m.Create("dir")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_Create_Ugly
func TestCoreMedium_Medium_Create_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.Create("nested/a.txt")
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}

// moved AX-7 triplet TestCoreMedium_Medium_Delete_Good
func TestCoreMedium_Medium_Delete_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertNoError(t, m.Delete("a.txt"))
	AssertFalse(t, m.Exists("a.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_Delete_Bad
func TestCoreMedium_Medium_Delete_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	err := m.Delete("missing")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_Delete_Ugly
func TestCoreMedium_Medium_Delete_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	err := m.Delete("")
	AssertNoError(t, err)
	AssertFalse(t, m.Exists("missing"))
}

// moved AX-7 triplet TestCoreMedium_Medium_DeleteAll_Good
func TestCoreMedium_Medium_DeleteAll_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("dir/a.txt", "hello"))
	AssertNoError(t, m.DeleteAll("dir"))
	AssertFalse(t, m.Exists("dir/a.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_DeleteAll_Bad
func TestCoreMedium_Medium_DeleteAll_Bad(t *T) {
	m := newCoreMedium("/")
	err := m.DeleteAll("")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_DeleteAll_Ugly
func TestCoreMedium_Medium_DeleteAll_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.DeleteAll("missing"))
	AssertFalse(t, m.Exists("missing"))
}

// moved AX-7 triplet TestCoreMedium_Medium_EnsureDir_Good
func TestCoreMedium_Medium_EnsureDir_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir/sub"))
	AssertTrue(t, m.IsDir("dir/sub"))
}

// moved AX-7 triplet TestCoreMedium_Medium_EnsureDir_Bad
func TestCoreMedium_Medium_EnsureDir_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("file", "x"))
	AssertError(t, m.EnsureDir("file"))
}

// moved AX-7 triplet TestCoreMedium_Medium_EnsureDir_Ugly
func TestCoreMedium_Medium_EnsureDir_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("nested/deep"))
	AssertNoError(t, m.EnsureDir("nested/deep"))
}

// moved AX-7 triplet TestCoreMedium_Medium_Exists_Good
func TestCoreMedium_Medium_Exists_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertTrue(t, m.Exists("a.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_Exists_Bad
func TestCoreMedium_Medium_Exists_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertFalse(t, m.Exists("missing"))
	AssertFalse(t, m.Exists("../missing"))
}

// moved AX-7 triplet TestCoreMedium_Medium_Exists_Ugly
func TestCoreMedium_Medium_Exists_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	AssertTrue(t, m.Exists("dir"))
}

// moved AX-7 triplet TestCoreMedium_Medium_IsDir_Good
func TestCoreMedium_Medium_IsDir_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	AssertTrue(t, m.IsDir("dir"))
}

// moved AX-7 triplet TestCoreMedium_Medium_IsDir_Bad
func TestCoreMedium_Medium_IsDir_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertFalse(t, m.IsDir("missing"))
	AssertFalse(t, m.IsDir(""))
}

// moved AX-7 triplet TestCoreMedium_Medium_IsDir_Ugly
func TestCoreMedium_Medium_IsDir_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertFalse(t, m.IsDir("a.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_IsFile_Good
func TestCoreMedium_Medium_IsFile_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	AssertTrue(t, m.IsFile("a.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_IsFile_Bad
func TestCoreMedium_Medium_IsFile_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertFalse(t, m.IsFile("missing"))
	AssertFalse(t, m.IsFile(""))
}

// moved AX-7 triplet TestCoreMedium_Medium_IsFile_Ugly
func TestCoreMedium_Medium_IsFile_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	AssertFalse(t, m.IsFile("dir"))
}

// moved AX-7 triplet TestCoreMedium_Medium_List_Good
func TestCoreMedium_Medium_List_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("dir/a.txt", "hello"))
	entries, err := m.List("dir")
	AssertNoError(t, err)
	AssertLen(t, entries, 1)
}

// moved AX-7 triplet TestCoreMedium_Medium_List_Bad
func TestCoreMedium_Medium_List_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	entries, err := m.List("missing")
	AssertError(t, err)
	AssertNil(t, entries)
}

// moved AX-7 triplet TestCoreMedium_Medium_List_Ugly
func TestCoreMedium_Medium_List_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("empty"))
	entries, err := m.List("empty")
	AssertNoError(t, err)
	AssertLen(t, entries, 0)
}

// moved AX-7 triplet TestCoreMedium_Medium_Open_Good
func TestCoreMedium_Medium_Open_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	f, err := m.Open("a.txt")
	AssertNoError(t, err)
	AssertNoError(t, f.Close())
}

// moved AX-7 triplet TestCoreMedium_Medium_Open_Bad
func TestCoreMedium_Medium_Open_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	f, err := m.Open("missing.txt")
	AssertError(t, err)
	AssertNil(t, f)
}

// moved AX-7 triplet TestCoreMedium_Medium_Open_Ugly
func TestCoreMedium_Medium_Open_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("empty.txt", ""))
	f, err := m.Open("empty.txt")
	AssertNoError(t, err)
	AssertNoError(t, f.Close())
}

// moved AX-7 triplet TestCoreMedium_Medium_Read_Good
func TestCoreMedium_Medium_Read_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	got, err := m.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "hello", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_Read_Bad
func TestCoreMedium_Medium_Read_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	got, err := m.Read("missing.txt")
	AssertError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_Read_Ugly
func TestCoreMedium_Medium_Read_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("nested/empty.txt", ""))
	got, err := m.Read("nested/empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_ReadStream_Good
func TestCoreMedium_Medium_ReadStream_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	r, err := m.ReadStream("a.txt")
	AssertNoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	AssertNoError(t, err)
	AssertEqual(t, "hello", string(data))
}

// moved AX-7 triplet TestCoreMedium_Medium_ReadStream_Bad
func TestCoreMedium_Medium_ReadStream_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	r, err := m.ReadStream("missing.txt")
	AssertError(t, err)
	AssertNil(t, r)
}

// moved AX-7 triplet TestCoreMedium_Medium_ReadStream_Ugly
func TestCoreMedium_Medium_ReadStream_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("empty.txt", ""))
	r, err := m.ReadStream("empty.txt")
	AssertNoError(t, err)
	defer r.Close()
	data, err := io.ReadAll(r)
	AssertNoError(t, err)
	AssertEqual(t, "", string(data))
}

// moved AX-7 triplet TestCoreMedium_Medium_Rename_Good
func TestCoreMedium_Medium_Rename_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("old.txt", "hello"))
	AssertNoError(t, m.Rename("old.txt", "new.txt"))
	AssertTrue(t, m.Exists("new.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_Rename_Bad
func TestCoreMedium_Medium_Rename_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	err := m.Rename("missing", "new")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_Rename_Ugly
func TestCoreMedium_Medium_Rename_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("old.txt", ""))
	AssertNoError(t, m.EnsureDir("nested"))
	AssertNoError(t, m.Rename("old.txt", "nested/new.txt"))
	AssertTrue(t, m.Exists("nested/new.txt"))
}

// moved AX-7 triplet TestCoreMedium_Medium_Stat_Good
func TestCoreMedium_Medium_Stat_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	info, err := m.Stat("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "a.txt", info.Name())
}

// moved AX-7 triplet TestCoreMedium_Medium_Stat_Bad
func TestCoreMedium_Medium_Stat_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	info, err := m.Stat("missing.txt")
	AssertError(t, err)
	AssertNil(t, info)
}

// moved AX-7 triplet TestCoreMedium_Medium_Stat_Ugly
func TestCoreMedium_Medium_Stat_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("empty.txt", ""))
	info, err := m.Stat("empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, int64(0), info.Size())
}

// moved AX-7 triplet TestCoreMedium_Medium_Write_Good
func TestCoreMedium_Medium_Write_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("a.txt", "hello"))
	got, err := m.Read("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "hello", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_Write_Bad
func TestCoreMedium_Medium_Write_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	err := m.Write("dir", "hello")
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_Write_Ugly
func TestCoreMedium_Medium_Write_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.Write("nested/empty.txt", ""))
	got, err := m.Read("nested/empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_WriteMode_Good
func TestCoreMedium_Medium_WriteMode_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.WriteMode("a.txt", "hello", 0o600))
	info, err := m.Stat("a.txt")
	AssertNoError(t, err)
	AssertEqual(t, "a.txt", info.Name())
}

// moved AX-7 triplet TestCoreMedium_Medium_WriteMode_Bad
func TestCoreMedium_Medium_WriteMode_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	err := m.WriteMode("dir", "hello", 0o600)
	AssertError(t, err)
}

// moved AX-7 triplet TestCoreMedium_Medium_WriteMode_Ugly
func TestCoreMedium_Medium_WriteMode_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.WriteMode("nested/empty.txt", "", 0o644))
	got, err := m.Read("nested/empty.txt")
	AssertNoError(t, err)
	AssertEqual(t, "", got)
}

// moved AX-7 triplet TestCoreMedium_Medium_WriteStream_Good
func TestCoreMedium_Medium_WriteStream_Good(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.WriteStream("a.txt")
	AssertNoError(t, err)
	_, err = w.Write([]byte("hello"))
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}

// moved AX-7 triplet TestCoreMedium_Medium_WriteStream_Bad
func TestCoreMedium_Medium_WriteStream_Bad(t *T) {
	m := newCoreMedium(t.TempDir())
	AssertNoError(t, m.EnsureDir("dir"))
	w, err := m.WriteStream("dir")
	AssertError(t, err)
	AssertNil(t, w)
}

// moved AX-7 triplet TestCoreMedium_Medium_WriteStream_Ugly
func TestCoreMedium_Medium_WriteStream_Ugly(t *T) {
	m := newCoreMedium(t.TempDir())
	w, err := m.WriteStream("nested/a.txt")
	AssertNoError(t, err)
	AssertNoError(t, w.Close())
}
