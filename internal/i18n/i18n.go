// Package i18n holds tgit's UI strings for every supported interface
// language and tracks which one is currently active. The language is chosen
// once, on the language-select screen shown before the login screen (see
// internal/ui.langModel), and every other screen reads strings through the T
// package variable so a single Set call re-languages the whole UI.
package i18n

// Lang is a supported interface language.
type Lang int

const (
	English Lang = iota
	Russian
)

// T holds the active language's strings. Screens read fields directly off
// this pointer at render time (e.g. i18n.T.LoginTitle), so calling Set
// re-languages every screen that hasn't rendered yet without needing to
// rebuild any model.
var T = &en

// Set makes lang the active interface language.
func Set(lang Lang) {
	switch lang {
	case Russian:
		T = &ru
	default:
		T = &en
	}
}

// Messages is the full set of user-facing strings tgit shows, in one
// language. Fields ending in Fmt are fmt format strings (Sprintf/Errorf).
type Messages struct {
	// Login screen
	LoginTitle            string
	LoginNeedTokenPrefix  string
	LoginNeedTokenSuffix  string
	LoginCreateTokenLabel string
	LoginCtrlOHint        string
	LoginTokenFieldLabel  string
	LoginChecking         string
	LoginEmptyTokenErr    string
	LoginHelp             string
	SavedTokenInvalid     string

	// Main screen: errors / status prefixes
	ErrNotGitRepoLocal       string
	DiffLoadFailedPrefix     string
	DiffEmptyPlaceholder     string
	DoctorPrefix             string
	DoctorFixFailedPrefix    string
	DoctorFixedPrefix        string
	StashPrefix              string
	StashContentFailedPrefix string

	// Action labels / results
	ActionCheckoutLabel     string
	ActionCreateBranchLabel string
	ActionCommitLabel       string
	ActionDoneGeneric       string
	PushDone                string
	PushResultPrefix        string
	PullDone                string
	PullResultPrefix        string
	FetchDone               string
	CheckoutDoneFmt         string
	BranchCreatedFmt        string
	CommitDoneMsg           string
	StashPushDoneMsg        string
	StashPopDoneMsg         string
	StashApplyDoneMsg       string
	StashDropDoneMsg        string

	// Busy labels / inline errors
	HashCopiedMsg           string
	NoRepoErr               string
	CheckingRepoBusy        string
	LoadingStashBusy        string
	PoppingStashBusy        string
	PushingBusy             string
	PullingBusy             string
	FetchingBusy            string
	SwitchingBranchBusy     string
	CreatingBranchBusy      string
	NothingToCommitErr      string
	EnterCommitMessageErr   string
	CommittingBusy          string
	StageChangeFailedPrefix string
	FixingBusy              string
	StashingBusy            string
	ApplyingStashBusy       string
	DroppingStashBusy       string

	// Panels / normal view
	GitHubNotConnected    string
	GitHubConnectedPrefix string
	PanelBranchesTitle    string
	NoDataMsg             string
	PanelFilesTitleFmt    string
	CleanMsg              string
	PanelLogTitle         string
	NoCommitsMsg          string
	PanelDiffTitle        string
	SelectFileOrCommitMsg string
	StatusHelpMsg         string

	// Commit modal
	NewCommitTitle  string
	StagedFilesFmt  string
	MessageLabel    string
	CommitModalHelp string

	// Branch switch modal
	SwitchBranchTitle  string
	FilterLabel        string
	NoBranchesMsg      string
	NoMatchesCreateFmt string
	BranchModalHelp    string

	// Doctor modal
	NoIssuesFoundMsg string
	CloseHelp        string
	IssuesFoundFmt   string
	FixConfirmMsg    string
	YesNoHelp        string
	DoctorListHelp   string

	// Stash modal
	StashTitle             string
	StashEmptyMsg          string
	StashEmptyHelp         string
	StashChangedFilesLabel string
	StashDropConfirmMsg    string
	StashListHelp          string

	// Input placeholders
	CommitInputPlaceholder  string
	BranchFilterPlaceholder string

	// No-repo screen
	NoRepoNotFoundFmt    string
	CloneHereLabel       string
	CloningBusy          string
	EnterRepoURLErr      string
	CloneModalHelp       string
	NoRecentProjectsMsg  string
	RecentProjectsLabel  string
	OpeningProjectBusy   string
	CloneActionHelp      string
	SelectOpenHelpPrefix string

	// doctor package issue titles
	MacJunkTitleFmt  string
	JunkDirsTitleFmt string

	// ghauth package errors
	NetworkErrFmt       string
	TokenRejectedMsg    string
	UnexpectedStatusFmt string
	ParseFailedFmt      string

	// gitrepo package errors
	NotGitRepoFmt        string
	CloneFailedFmt       string
	UnexpectedRevListMsg string
	BinaryPreviewMsg     string
	TruncatedSuffixMsg   string
}
