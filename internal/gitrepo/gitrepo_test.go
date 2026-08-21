package gitrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommitStatusStageUnstage(t *testing.T) {
	repo := newTestRepo(t)

	path := filepath.Join(repo.Root, "f.txt")
	if err := os.WriteFile(path, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !files[0].Untracked() {
		t.Fatalf("expected 1 untracked file, got %+v", files)
	}

	if err := repo.StageFile("f.txt"); err != nil {
		t.Fatal(err)
	}
	files, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !files[0].Staged() {
		t.Fatalf("expected 1 staged file, got %+v", files)
	}

	if err := repo.Commit("first commit"); err != nil {
		t.Fatal(err)
	}
	files, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("expected clean status after commit, got %+v", files)
	}
	commits, err := repo.Log(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].Subject != "first commit" {
		t.Fatalf("unexpected log: %+v", commits)
	}

	// модификация отслеживаемого файла: unstaged -> staged -> unstaged
	if err := os.WriteFile(path, []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Staged() {
		t.Fatalf("expected 1 unstaged modification, got %+v", files)
	}

	if err := repo.StageFile("f.txt"); err != nil {
		t.Fatal(err)
	}
	files, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || !files[0].Staged() {
		t.Fatalf("expected staged modification, got %+v", files)
	}

	if err := repo.UnstageFile("f.txt"); err != nil {
		t.Fatal(err)
	}
	files, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Staged() {
		t.Fatalf("expected unstaged modification after UnstageFile, got %+v", files)
	}
}

func TestBranches(t *testing.T) {
	repo := newTestRepo(t)

	branches, err := repo.Branches()
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 0 {
		t.Fatalf("expected no branches before first commit, got %v", branches)
	}

	writeAndCommit(t, repo, "f.txt", "v1\n", "init")
	defaultBranch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	branches, err = repo.Branches()
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 1 || branches[0] != defaultBranch {
		t.Fatalf("expected [%s], got %v", defaultBranch, branches)
	}

	if err := repo.CreateBranch("feature"); err != nil {
		t.Fatal(err)
	}
	current, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if current != "feature" {
		t.Fatalf("expected current branch feature, got %s", current)
	}
	branches, err = repo.Branches()
	if err != nil {
		t.Fatal(err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %v", branches)
	}
}

func TestCheckoutCreateBranch(t *testing.T) {
	repo := newTestRepo(t)
	writeAndCommit(t, repo, "f.txt", "v1\n", "init")
	defaultBranch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.CreateBranch("feature"); err != nil {
		t.Fatal(err)
	}
	current, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if current != "feature" {
		t.Fatalf("expected feature, got %s", current)
	}

	if err := repo.Checkout(defaultBranch); err != nil {
		t.Fatal(err)
	}
	current, err = repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if current != defaultBranch {
		t.Fatalf("expected %s, got %s", defaultBranch, current)
	}

	if err := repo.Checkout("nonexistent"); err == nil {
		t.Fatal("expected error checking out nonexistent branch, got nil")
	}
}
