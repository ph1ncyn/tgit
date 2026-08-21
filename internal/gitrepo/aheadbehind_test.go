package gitrepo

import "testing"

func TestAheadBehind_WithUpstream(t *testing.T) {
	local, remoteDir := newRepoWithRemote(t)
	writeAndCommit(t, local, "f.txt", "v1\n", "init")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "HEAD")

	other := cloneRemote(t, remoteDir)
	writeAndCommit(t, other, "f.txt", "v2\n", "remote change 1")
	writeAndCommit(t, other, "f.txt", "v3\n", "remote change 2")
	runGitT(t, other.Root, "push", "-q")

	runGitT(t, local.Root, "fetch", "-q")
	writeAndCommit(t, local, "g.txt", "local\n", "local change")

	ahead, behind, err := local.AheadBehind()
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 1 || behind != 2 {
		t.Fatalf("expected ahead=1 behind=2, got ahead=%d behind=%d", ahead, behind)
	}
}

func TestAheadBehind_NoUpstreamFallsBackToRemotes(t *testing.T) {
	local, remoteDir := newRepoWithRemote(t)
	writeAndCommit(t, local, "f.txt", "v1\n", "init")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "HEAD")

	// новая ветка без upstream — ни разу не запушена
	runGitT(t, local.Root, "checkout", "-q", "-b", "feature")
	writeAndCommit(t, local, "f.txt", "v2\n", "feature work")

	_ = remoteDir
	ahead, behind, err := local.AheadBehind()
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 1 || behind != 0 {
		t.Fatalf("expected ahead=1 behind=0 (no-upstream fallback), got ahead=%d behind=%d", ahead, behind)
	}
}

func TestAheadBehind_NoRemotesAndNoCommits(t *testing.T) {
	repo := newTestRepo(t)

	// без remote и без коммитов вообще (unborn HEAD) — rev-list HEAD падает
	// целиком, это должно тихо превратиться в 0/0, а не в ошибку.
	ahead, behind, err := repo.AheadBehind()
	if err != nil {
		t.Fatalf("expected no error on empty repo, got %v", err)
	}
	if ahead != 0 || behind != 0 {
		t.Fatalf("expected ahead=0 behind=0 on empty repo, got ahead=%d behind=%d", ahead, behind)
	}
}

func TestAheadBehind_CommitsButNoRemotes(t *testing.T) {
	repo := newTestRepo(t)
	writeAndCommit(t, repo, "f.txt", "v1\n", "init")
	writeAndCommit(t, repo, "f.txt", "v2\n", "second")

	// есть коммиты, но remote вообще нет — --not --remotes ничего не
	// исключает, поэтому ahead = все коммиты HEAD, а не 0.
	ahead, behind, err := repo.AheadBehind()
	if err != nil {
		t.Fatal(err)
	}
	if ahead != 2 || behind != 0 {
		t.Fatalf("expected ahead=2 behind=0 (no remotes at all), got ahead=%d behind=%d", ahead, behind)
	}
}

func TestDeleteGoneBranches_RemovesGoneBranch(t *testing.T) {
	local, remoteDir := newRepoWithRemote(t)
	writeAndCommit(t, local, "f.txt", "v1\n", "init")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "HEAD")
	mainBranch, err := local.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	runGitT(t, local.Root, "checkout", "-q", "-b", "feature-a")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "feature-a")
	runGitT(t, local.Root, "checkout", "-q", mainBranch)

	// удаляем ветку прямо на remote, как будто это сделали через GitHub
	runGitT(t, remoteDir, "branch", "-D", "feature-a")
	runGitT(t, local.Root, "fetch", "-q", "--prune")

	deleted, err := local.deleteGoneBranches()
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 1 || deleted[0] != "feature-a" {
		t.Fatalf("expected [feature-a] deleted, got %v", deleted)
	}
	branches, err := local.Branches()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range branches {
		if b == "feature-a" {
			t.Fatalf("feature-a should have been removed, branches=%v", branches)
		}
	}
}

func TestDeleteGoneBranches_NeverRemovesCurrentBranch(t *testing.T) {
	local, remoteDir := newRepoWithRemote(t)
	writeAndCommit(t, local, "f.txt", "v1\n", "init")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "HEAD")

	runGitT(t, local.Root, "checkout", "-q", "-b", "feature-a")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "feature-a")
	// остаёмся на feature-a — именно её удаляем на remote

	runGitT(t, remoteDir, "branch", "-D", "feature-a")
	runGitT(t, local.Root, "fetch", "-q", "--prune")

	deleted, err := local.deleteGoneBranches()
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 0 {
		t.Fatalf("expected current branch never deleted, got %v", deleted)
	}
	current, err := local.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if current != "feature-a" {
		t.Fatalf("expected still on feature-a, got %s", current)
	}
}

func TestDeleteGoneBranches_PreservesUnmergedLocalCommits(t *testing.T) {
	local, remoteDir := newRepoWithRemote(t)
	writeAndCommit(t, local, "f.txt", "v1\n", "init")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "HEAD")
	mainBranch, err := local.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	runGitT(t, local.Root, "checkout", "-q", "-b", "feature-a")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "feature-a")
	// локальный коммит поверх запушенного — не отправлен на remote
	writeAndCommit(t, local, "f.txt", "v2\n", "unmerged local work")
	runGitT(t, local.Root, "checkout", "-q", mainBranch)

	runGitT(t, remoteDir, "branch", "-D", "feature-a")
	runGitT(t, local.Root, "fetch", "-q", "--prune")

	deleted, err := local.deleteGoneBranches()
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 0 {
		t.Fatalf("expected branch with unmerged commits preserved, got deleted=%v", deleted)
	}
	branches, err := local.Branches()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, b := range branches {
		if b == "feature-a" {
			found = true
		}
	}
	if !found {
		t.Fatalf("feature-a should still exist, branches=%v", branches)
	}
}

func TestBranchTracking(t *testing.T) {
	local, remoteDir := newRepoWithRemote(t)
	writeAndCommit(t, local, "f.txt", "v1\n", "init")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "HEAD")
	mainBranch, err := local.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}

	// ahead с upstream
	runGitT(t, local.Root, "checkout", "-q", "-b", "tracked-ahead")
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "tracked-ahead")
	writeAndCommit(t, local, "f.txt", "v2\n", "local change")

	// без upstream вовсе
	runGitT(t, local.Root, "checkout", "-q", "-b", "no-upstream", mainBranch)

	// gone
	runGitT(t, local.Root, "checkout", "-q", "-b", "to-delete", mainBranch)
	runGitT(t, local.Root, "push", "-q", "-u", "origin", "to-delete")
	runGitT(t, remoteDir, "branch", "-D", "to-delete")
	runGitT(t, local.Root, "fetch", "-q", "--prune")

	runGitT(t, local.Root, "checkout", "-q", mainBranch)

	tr, err := local.BranchTracking()
	if err != nil {
		t.Fatal(err)
	}
	if got := tr[mainBranch]; got.NoUpstream || got.Ahead != 0 || got.Behind != 0 || got.Gone {
		t.Errorf("%s expected clean tracked branch, got %+v", mainBranch, got)
	}
	if got := tr["tracked-ahead"]; got.NoUpstream || got.Ahead != 1 || got.Behind != 0 {
		t.Errorf("tracked-ahead expected ahead=1, got %+v", got)
	}
	if got := tr["no-upstream"]; !got.NoUpstream {
		t.Errorf("no-upstream expected NoUpstream=true, got %+v", got)
	}
	if got := tr["to-delete"]; !got.Gone {
		t.Errorf("to-delete expected Gone=true, got %+v", got)
	}
}
