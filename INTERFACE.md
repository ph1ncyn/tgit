# tgit — interface

The most convenient, feature-rich git terminal: the mouse as a first-class way of working (not
just the keyboard), a pleasant visual look, a built-in "doctor" for common repository problems
(including macOS junk files), and fast branch switching.

Implementation stack: **Go + Bubble Tea + Lip Gloss + Bubbles**. This gives proper mouse support
in the terminal (clicks, scroll, hover), fast rendering, and a single binary that runs on
Linux/macOS/Windows.

How it differs from lazygit/gitui/tig:
- The mouse isn't an afterthought, it's a full interface: clicks, hovers, context menus, drag to resize.
- A built-in **Doctor** — a scanner for common repository problems with one-click fix buttons.
- A command palette (`Ctrl+K`) — no need to memorize every hotkey.

---

## 0. Language selection

The very first screen tgit shows, before the token/login screen, lets you pick the interface
language:

```
┌────────────────────────────────────────────┐
│ tgit                                        │
│                                              │
│ Select interface language:                  │
│                                              │
│ > English                                   │
│   Русский                                   │
│                                              │
│ ↑/↓ — select  •  enter — confirm            │
└──────────────────────────────────────────────┘
```

`↑`/`↓` moves the selection, `Enter` confirms and switches every other screen (login, main
screen, panels, dialogs, status messages) to the chosen language.

---

## 1. Main screen

```
┌ tgit ── ~/projects/tgit ── ⎇ main ↑2 ↓0 ── ✓ clean ─────────────────────────┐
│                                                                              │
│ ┌─ Branches ───────────┐ ┌─ Files ───────────────┐ ┌─ Diff: cli/app.go ────┐ │
│ │ ⎇ main          [•]  │ │ Staged (2)            │ │  12  func Run() {     │ │
│ │   feature/doctor     │ │  [x] M cli/app.go     │ │  13 -   old()         │ │
│ │   fix/apple-double   │ │  [x] A doctor/rules.go│ │  13 +   New()          │ │
│ │ ── remotes ──        │ │ Unstaged (1)          │ │  14      }             │ │
│ │   origin/main        │ │  [ ] M README.md      │ │                        │ │
│ │ ── tags ──            │ │ Untracked (1)         │ │                        │ │
│ │   v0.1.0             │ │  [ ] ? notes.txt       │ │                        │ │
│ └───────────────────────┘ └────────────────────────┘ └────────────────────────┘ │
│ ┌─ Stash ──────────────┐ ┌─ Log ──────────────────────────────────────────────┐ │
│ │  (empty)              │ │ ● a1b2c3 fix: doctor rule for ._files (HEAD)      │ │
│ │                        │ │ ● 9f8e7d feat: branch switcher                    │ │
│ │                        │ │ │ ● 4c5d6e wip                                    │ │
│ │                        │ │ ●/  1122aa init                                  │ │
│ └───────────────────────┘ └────────────────────────────────────────────────────┘ │
│                                                                              │
│ [Commit] [Push] [Pull] [Fetch] [Branch] [Stash] [Merge] [Rebase] [Doctor⚠2] │
│  c        p      P      f       b        s       m       r       d          │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Panels:**
- **Branches** — local, remote, tags; the current one is marked `[•]`.
- **Files** — Staged / Unstaged / Untracked, checkboxes on the left.
- **Diff** — the content of the selected file/commit on the right.
- **Stash** — the list of saved stashes.
- **Log** — the commit graph.
- **Top bar** — repository, current branch, ahead/behind, cleanliness status.
- **Bottom toolbar** — clickable action buttons with the hotkey hint under each. The `Doctor`
  button shows a badge with the number of issues found.

The active panel is highlighted with an accent border; `Tab`/click switches focus.

---

## 2. Buttons and mouse

- **Toolbar** — a row of bordered buttons; hover highlights them, click runs the action. Each
  button has its hotkey letter shown underneath.
- **Clicking a file** — toggles stage/unstage (click the checkbox to the left of the name).
- **Clicking a branch** — offers checkout; if there are uncommitted changes, asks for
  confirmation (stash / commit / cancel).
- **Clicking a commit** in the graph — opens its diff/details in the right panel.
- **Scrolling** — independent per panel, wherever the cursor is hovering.
- **Dragging** a border between panels — resize (not in the first version, marked as a future
  extension).
- **Right click** — a context menu whose contents depend on what was clicked:
  - on a file: Stage / Unstage / Discard / Diff
  - on a branch: Checkout / Rename / Delete / Merge into current
  - on a commit: Checkout / Cherry-pick / Revert / Copy hash
- Every mouse action has a keyboard equivalent — the mouse speeds things up, it's never the only way.

---

## 3. Git Doctor — fixing common problems

The `[Doctor]` button opens a panel listing the problems found, their status, and a quick-fix
button next to each:

```
┌─ Doctor ───────────────────────────────────────────────────────────────┐
│ ⚠ 3 issues found                                                        │
│                                                                          │
│ • macOS files (._*, .DS_Store) cluttering the repo         [Fix]      │
│     ._app.go, .DS_Store in cli/                                        │
│     → add to .gitignore / global gitignore / git rm --cached           │
│                                                                          │
│ • node_modules/ is tracked but looks like a build dir      [Fix]      │
│     → add a pattern to .gitignore                                      │
│                                                                          │
│ • Mixed line endings (CRLF/LF) in 4 files                  [Fix]      │
│     → offer to add .gitattributes                                      │
└──────────────────────────────────────────────────────────────────────────┘
```

Rules in the first version:
- **AppleDouble files `._*` and `.DS_Store`** (including already-committed ones) → "Add to
  `.gitignore`" / "Add to global `~/.gitignore_global`" / "Remove from the repository
  (`git rm --cached`)".
- **Untracked but looking like build/tooling output** (node_modules, .env, dist/) → offer to
  add a pattern to `.gitignore`.
- **Mixed line endings (CRLF/LF)** → offer to add `.gitattributes`.
- **Unfinished merge/rebase/conflict** → quick jump into the conflict-resolution view.
- **Detached HEAD** → offer to create a branch or return to the previous one.
- **Large files** in history/staged → offer to move them to Git LFS.

The rule list is extensible — new checks can be added without changing the rest of the interface.

---

## 4. Switching branches

- The "Branches" panel on the left: local / remote / tags, the current one highlighted.
- Quick switcher — `Ctrl+B` or click the `[Branch]` button: a modal with fuzzy-search by name,
  arrow-key or mouse navigation, `Enter`/click — checkout.

```
┌─ Switch branch ────────────────────────────┐
│ > feat                                     │
│   feature/doctor            ↑2 ↓0          │
│   feature/branch-switcher   ↑0 ↓1          │
│ ── recent ──                               │
│   main                                     │
│   fix/apple-double                         │
└─────────────────────────────────────────────┘
```

- At the top — "recent branches" for jumping back and forth instantly.
- Next to each branch — ahead/behind relative to upstream.
- If there are uncommitted changes in the working directory — a stash/commit/cancel prompt
  appears before checkout.

---

## 5. Visual style

- Rounded panel borders (Lip Gloss borders); the active panel has an accent-colored border.
- Color semantics:
  - green — staged / clean
  - red — unstaged / conflict
  - yellow — modified
  - blue/cyan — branch / remote
  - gray — untracked / ignored
- Dark and light themes; icons via Nerd Font with a plain-ASCII fallback if the font isn't available.
- A separate status line at the bottom of the window (not the toolbar): repository/branch/mode
  (`NORMAL` / `COMMIT MSG` / `CONFLICT`).

---

## 6. Command palette

`Ctrl+K` or clicking the icon in the top bar — a quick-search window for actions by name (like
VS Code): "checkout", "stash pop", "force push", etc. Useful until every hotkey is second nature.

---

## 7. Future work (roadmap, no implementation details)

- Visual interactive rebase (drag-and-drop commits).
- A built-in merge-conflict resolver (3-way view).
- Per-file blame view.
- Search across the repository and across commits.
- GitHub/GitLab integration: PR list, CI status right inside tgit.
- Undo for the last git action.
- Configurable keybindings and theme via a config file.

Stash GUI:
┌─ Select files for stash ──────────────────┐  Date is the last-modified time
│ [x] File1.txt                      1 week │
│ [x]  main.py                       2 days │
│ []  .env                           3 week │
│                                           │
└───────────────────────────────────────────┘
