package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"tgit/internal/doctor"
	"tgit/internal/gitrepo"
)

// actionResultMsg — результат любой асинхронной git-операции, запущенной с
// главного экрана (push/pull/fetch/checkout/commit/stash).
type actionResultMsg struct {
	action string
	arg    string
	output string
	err    error
}

func runAction(action, arg string, fn func() (string, error)) tea.Cmd {
	return func() tea.Msg {
		out, err := fn()
		return actionResultMsg{action: action, arg: arg, output: out, err: err}
	}
}

func pushCmd(repo *gitrepo.Repo, token string) tea.Cmd {
	return runAction("push", "", func() (string, error) { return repo.Push(token) })
}

func pullCmd(repo *gitrepo.Repo, token string) tea.Cmd {
	return runAction("pull", "", func() (string, error) { return repo.Pull(token) })
}

func fetchCmd(repo *gitrepo.Repo, token string) tea.Cmd {
	return runAction("fetch", "", func() (string, error) { return repo.Fetch(token) })
}

func checkoutCmd(repo *gitrepo.Repo, branch string) tea.Cmd {
	return runAction("checkout", branch, func() (string, error) { return "", repo.Checkout(branch) })
}

func createBranchCmd(repo *gitrepo.Repo, name string) tea.Cmd {
	return runAction("create-branch", name, func() (string, error) { return "", repo.CreateBranch(name) })
}

func commitCmd(repo *gitrepo.Repo, message string) tea.Cmd {
	return runAction("commit", "", func() (string, error) { return "", repo.Commit(message) })
}

func stashPushCmd(repo *gitrepo.Repo) tea.Cmd {
	return runAction("stash-push", "", func() (string, error) { return repo.StashPush() })
}

func stashPopCmd(repo *gitrepo.Repo, ref string) tea.Cmd {
	return runAction("stash-pop", ref, func() (string, error) { return repo.StashPop(ref) })
}

func stashApplyCmd(repo *gitrepo.Repo, ref string) tea.Cmd {
	return runAction("stash-apply", ref, func() (string, error) { return repo.StashApply(ref) })
}

func stashDropCmd(repo *gitrepo.Repo, ref string) tea.Cmd {
	return runAction("stash-drop", ref, func() (string, error) { return repo.StashDrop(ref) })
}

// --- stash list ---

type stashListedMsg struct {
	stashes []gitrepo.Stash
	err     error
}

func stashListCmd(repo *gitrepo.Repo) tea.Cmd {
	return func() tea.Msg {
		stashes, err := repo.StashList()
		return stashListedMsg{stashes: stashes, err: err}
	}
}

type stashStatMsg struct {
	ref     string
	content string
	err     error
}

func stashStatCmd(repo *gitrepo.Repo, ref string) tea.Cmd {
	return func() tea.Msg {
		out, err := repo.StashStat(ref)
		return stashStatMsg{ref: ref, content: out, err: err}
	}
}

// --- diff ---

type diffLoadedMsg struct {
	content string
	err     error
}

func diffFileCmd(repo *gitrepo.Repo, path string, staged bool) tea.Cmd {
	return func() tea.Msg {
		d, err := repo.DiffFile(path, staged)
		return diffLoadedMsg{content: d, err: err}
	}
}

func diffCommitCmd(repo *gitrepo.Repo, hash string) tea.Cmd {
	return func() tea.Msg {
		d, err := repo.DiffCommit(hash)
		return diffLoadedMsg{content: d, err: err}
	}
}

// --- doctor ---

type doctorScannedMsg struct {
	issues []doctor.Issue
	err    error
}

func doctorScanCmd(repo *gitrepo.Repo) tea.Cmd {
	return func() tea.Msg {
		issues, err := doctor.Scan(repo)
		return doctorScannedMsg{issues: issues, err: err}
	}
}

type doctorFixedMsg struct {
	title string
	err   error
}

func doctorFixCmd(repo *gitrepo.Repo, issue doctor.Issue) tea.Cmd {
	return func() tea.Msg {
		err := issue.Fix(repo)
		return doctorFixedMsg{title: issue.Title, err: err}
	}
}
