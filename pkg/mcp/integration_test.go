package mcp

import (
	"context"

	core "dappco.re/go"
)

func TestIntegration_FileTools(t *core.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	core.AssertNoError(t, err)

	ctx := context.Background()

	// 1. Test file_write
	writeInput := WriteFileInput{
		Path:    "test.txt",
		Content: "hello world",
	}
	_, writeOutput, err := s.writeFile(ctx, nil, writeInput)
	core.AssertNoError(t, err)
	core.AssertTrue(t, writeOutput.Success)
	core.AssertEqual(t, "test.txt", writeOutput.Path)

	// Verify on disk
	readResult := core.ReadFile(core.PathJoin(tmpDir, "test.txt"))
	core.AssertTrue(t, readResult.OK)
	content := readResult.Value.([]byte)
	core.AssertEqual(t, "hello world", string(content))

	// 2. Test file_read
	readInput := ReadFileInput{
		Path: "test.txt",
	}
	_, readOutput, err := s.readFile(ctx, nil, readInput)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "hello world", readOutput.Content)
	core.AssertEqual(t, "plaintext", readOutput.Language)

	// 3. Test file_edit (replace_all=false)
	editInput := EditDiffInput{
		Path:      "test.txt",
		OldString: "world",
		NewString: "mcp",
	}
	_, editOutput, err := s.editDiff(ctx, nil, editInput)
	core.AssertNoError(t, err)
	core.AssertTrue(t, editOutput.Success)
	core.AssertEqual(t, 1, editOutput.Replacements)

	// Verify change
	_, readOutput, _ = s.readFile(ctx, nil, readInput)
	core.AssertEqual(t, "hello mcp", readOutput.Content)

	// 4. Test file_edit (replace_all=true)
	_ = s.medium.Write("multi.txt", "abc abc abc")
	editInputMulti := EditDiffInput{
		Path:       "multi.txt",
		OldString:  "abc",
		NewString:  "xyz",
		ReplaceAll: true,
	}
	_, editOutput, err = s.editDiff(ctx, nil, editInputMulti)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 3, editOutput.Replacements)

	readResult = core.ReadFile(core.PathJoin(tmpDir, "multi.txt"))
	core.AssertTrue(t, readResult.OK)
	content = readResult.Value.([]byte)
	core.AssertEqual(t, "xyz xyz xyz", string(content))

	// 5. Test dir_list
	_ = s.medium.EnsureDir("subdir")
	_ = s.medium.Write("subdir/file1.txt", "content1")

	listInput := ListDirectoryInput{
		Path: "subdir",
	}
	_, listOutput, err := s.listDirectory(ctx, nil, listInput)
	core.AssertNoError(t, err)
	core.AssertLen(t, listOutput.Entries, 1)
	core.AssertEqual(t, "file1.txt", listOutput.Entries[0].Name)
	core.AssertFalse(t, listOutput.Entries[0].IsDir)
}

func TestIntegration_ErrorPaths(t *core.T) {
	tmpDir := t.TempDir()
	s, err := New(Options{WorkspaceRoot: tmpDir})
	core.AssertNoError(t, err)

	ctx := context.Background()

	// Read nonexistent file
	_, _, err = s.readFile(ctx, nil, ReadFileInput{Path: "nonexistent.txt"})
	core.AssertError(t, err)

	// Edit nonexistent file
	_, _, err = s.editDiff(ctx, nil, EditDiffInput{
		Path:      "nonexistent.txt",
		OldString: "foo",
		NewString: "bar",
	})
	core.AssertError(t, err)

	// Edit with empty old_string
	_, _, err = s.editDiff(ctx, nil, EditDiffInput{
		Path:      "test.txt",
		OldString: "",
		NewString: "bar",
	})
	core.AssertError(t, err)

	// Edit with old_string not found
	_ = s.medium.Write("test.txt", "hello")
	_, _, err = s.editDiff(ctx, nil, EditDiffInput{
		Path:      "test.txt",
		OldString: "missing",
		NewString: "bar",
	})
	core.AssertError(t, err)
}
