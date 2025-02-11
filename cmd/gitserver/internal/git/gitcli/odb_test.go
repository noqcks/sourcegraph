package gitcli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/internal/gitserver/gitdomain"
	"github.com/sourcegraph/sourcegraph/lib/errors"
)

func TestGitCLIBackend_ReadFile(t *testing.T) {
	ctx := context.Background()

	// Prepare repo state:
	backend := BackendWithRepoCommands(t,
		// simple file
		"echo abcd > file1",
		"git add file1",
		"git commit -m commit --author='Foo Author <foo@sourcegraph.com>'",

		// test we handle file names with .. (git show by default interprets
		// this). Ensure past the .. exists as a branch. Then if we use git
		// show it would return a diff instead of file contents.
		"mkdir subdir",
		"echo old > subdir/name",
		"echo old > subdir/name..dev",
		"git add subdir",
		"git commit -m commit --author='Foo Author <foo@sourcegraph.com>'",
		"echo dotdot > subdir/name..dev",
		"git add subdir",
		"git commit -m commit --author='Foo Author <foo@sourcegraph.com>'",
		"git branch dev",
	)

	commitID, err := backend.RevParseHead(ctx)
	require.NoError(t, err)

	t.Run("read simple file", func(t *testing.T) {
		r, err := backend.ReadFile(ctx, commitID, "file1")
		require.NoError(t, err)
		t.Cleanup(func() { r.Close() })
		contents, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, "abcd\n", string(contents))
	})

	t.Run("non existent file", func(t *testing.T) {
		_, err := backend.ReadFile(ctx, commitID, "filexyz")
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("non existent commit", func(t *testing.T) {
		_, err := backend.ReadFile(ctx, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "file1")
		require.Error(t, err)
		require.True(t, errors.HasType(err, &gitdomain.RevisionNotFoundError{}))
	})

	t.Run("special file paths", func(t *testing.T) {
		// File with .. in path name:
		{
			r, err := backend.ReadFile(ctx, commitID, "subdir/name..dev")
			require.NoError(t, err)
			t.Cleanup(func() { r.Close() })
			contents, err := io.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, "dotdot\n", string(contents))
		}
		// File with .. in path name that doesn't exist:
		{
			_, err := backend.ReadFile(ctx, commitID, "subdir/404..dev")
			require.Error(t, err)
			require.True(t, os.IsNotExist(err))
		}
		// This test case ensures we do not return a log with diff for the
		// specially crafted "git show HASH:..branch". IE a way to bypass
		// sub-repo permissions.
		{
			_, err := backend.ReadFile(ctx, commitID, "..dev")
			require.Error(t, err)
			require.True(t, os.IsNotExist(err))
		}

		// 3 dots ... as a prefix when using git show will return an error like
		// error: object b5462a7c880ce339ba3f93ac343706c0fa35babc is a tree, not a commit
		// fatal: Invalid symmetric difference expression 269e2b9bda9a95ad4181a7a6eb2058645d9bad82:...dev
		{
			_, err := backend.ReadFile(ctx, commitID, "...dev")
			require.Error(t, err)
			require.True(t, os.IsNotExist(err))
		}
	})

	t.Run("submodule", func(t *testing.T) {
		submodDir := RepoWithCommands(t,
			// simple file
			"echo abcd > file1",
			"git add file1",
			"git commit -m commit --author='Foo Author <foo@sourcegraph.com>'",
		)

		// Prepare repo state:
		backend := BackendWithRepoCommands(t,
			// simple file
			"echo abcd > file1",
			"git add file1",
			"git commit -m commit --author='Foo Author <foo@sourcegraph.com>'",

			// Add submodule
			"git -c protocol.file.allow=always submodule add "+filepath.ToSlash(string(submodDir))+" submod",
			"git commit -m 'add submodule' --author='Foo Author <foo@sourcegraph.com>'",
		)

		commitID, err := backend.RevParseHead(ctx)
		require.NoError(t, err)

		r, err := backend.ReadFile(ctx, commitID, "submod")
		require.NoError(t, err)
		t.Cleanup(func() { r.Close() })
		contents, err := io.ReadAll(r)
		require.NoError(t, err)
		// A submodule should read like an empty file for now.
		require.Equal(t, "", string(contents))
	})
}

func TestGitCLIBackend_ReadFile_GoroutineLeak(t *testing.T) {
	ctx := context.Background()

	// Prepare repo state:
	backend := BackendWithRepoCommands(t,
		// simple file
		"echo abcd > file1",
		"git add file1",
		"git commit -m commit --author='Foo Author <foo@sourcegraph.com>'",
	)

	commitID, err := backend.RevParseHead(ctx)
	require.NoError(t, err)

	routinesBefore := runtime.NumGoroutine()

	r, err := backend.ReadFile(ctx, commitID, "file1")
	require.NoError(t, err)

	// Read just a few bytes, but not enough to complete.
	buf := make([]byte, 2)
	n, err := r.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// Don't complete reading all the output, instead, bail and close the reader.
	require.NoError(t, r.Close())

	// Expect no leaked routines.
	routinesAfter := runtime.NumGoroutine()
	require.Equal(t, routinesBefore, routinesAfter)
}
