# tgit — keybindings

Full list of keybindings by screen. A short version is in [README.md](README.md#main-screen);
this one is exhaustive, including the login screen and every modal. The mouse is documented
separately in [README.md#mouse](README.md#mouse).

Unless stated otherwise, **`Ctrl+C`** closes tgit anywhere.

---

## 0. Language selection screen

Shown before anything else, on every launch, before the login screen.

| Key         | Action                                                            |
|-------------|--------------------------------------------------------------------|
| `↑` / `k`   | move selection up                                                   |
| `↓` / `j`   | move selection down                                                 |
| `Enter`     | confirm the selected language and continue to the login/token screen |
| `Ctrl+C`    | quit tgit                                                            |

---

## 1. Login screen (GitHub)

Shown on first launch (or after `g` from the main screen), until a working token is saved.

| Key         | Action                                                            |
|-------------|----------------------------------------------------------------------|
| `Enter`     | verify the entered token against the GitHub API and sign in          |
| `Ctrl+O`    | open the token-creation link in the default browser                  |
| `Esc`       | skip sign-in — work locally without GitHub (you can sign in later with `g`) |
| `Ctrl+C`    | quit tgit                                                             |

While a previously saved token is being silently verified (at startup), only `Esc`
(skip the wait) and `Ctrl+C` are available.

---

## 2. "Repository not found" screen

Shown instead of the main screen if the current folder isn't a git repository (no `.git`
found). Offers to open one of the recent tgit projects or clone a repository right into the
current folder.

| Key         | Action                                                            |
|-------------|----------------------------------------------------------------------|
| `↑` / `k`   | move up the list of recent projects                                  |
| `↓` / `j`   | move down the list of recent projects                                |
| `Enter`     | open the selected recent project (switches to its main screen)       |
| `c`         | switch to entering a URL to clone a repository into the current folder |
| `g`         | sign in to GitHub / change token (unavailable while entering a URL)  |
| `Ctrl+C`    | quit tgit                                                             |

**In URL-entry mode (after `c`):**

| Key         | Action                                                            |
|-------------|----------------------------------------------------------------------|
| `Enter`     | clone the repository at the entered URL into the current folder      |
| `Esc`       | go back to the list of recent projects                               |
| `Ctrl+C`    | quit tgit                                                             |

After successfully opening a recent project or cloning, tgit immediately shows the main screen
for that repository.

---

## 3. Main screen

Four panels: **Branches**, **Files**, **Log**, **Diff**. The active panel is highlighted with a border.

### Navigation and general actions

| Key                 | Action                                                            |
|---------------------|------------------------------------------------------------------|
| `Tab`               | next panel (Branches → Files → Log → Diff → Branches…)          |
| `Shift+Tab`         | previous panel                                                    |
| `↑` / `k`           | up the active panel's list (in Diff — scroll up)                  |
| `↓` / `j`           | down the active panel's list (in Diff — scroll down)              |
| `r`                 | refresh repository data (branches, files, log)                    |
| `g`                 | sign in to GitHub / change token                                   |
| `Ctrl+C`            | quit                                                               |

### Context-dependent keys (depend on the active panel)

| Key         | In the "Files" panel                       | In the "Branches" panel              | In "Log" / "Diff" |
|-------------|---------------------------------------------|--------------------------------------|--------------------|
| `space`     | stage / unstage the file under the cursor    | —                                     | — |
| `enter`     | same as `space`                              | checkout the selected branch          | — |
| `y`         | —                                             | —                                     | in Log — copy the full commit hash to the clipboard |

### Actions and dialogs (work from any panel)

| Key       | Opens / performs                                                                              |
|-----------|-------------------------------------------------------------------------------------------------|
| `c`       | new commit — a message-input modal (needs staged files, otherwise shows a hint)                 |
| `b`       | branch switcher — a modal with a filter and the ability to create a new branch                  |
| `s`       | **Stash** — a modal listing stashes                                                             |
| `S`       | quick pop of the latest stash, without opening the Stash modal                                  |
| `d`       | **Doctor** — scans the repository for common problems                                           |
| `p`       | push (uses the saved GitHub token for HTTPS repositories on github.com)                         |
| `P`       | pull                                                                                             |
| `f`       | fetch --all                                                                                       |
| `u`       | update tgit (only when "update available" shows in the top bar) — git pull + rebuild + restart   |

While an action is running (push/pull/checkout/commit/…), a spinner shows in the status line;
new keyboard actions are ignored until the current one finishes.

---

## 4. Commit modal (`c`)

| Key       | Action                                       |
|-----------|-----------------------------------------------|
| any character | types into the commit message              |
| `Enter`   | create a commit from the staged files          |
| `Esc`     | cancel and return to the main screen           |

---

## 5. Branch switcher (`b`)

| Key       | Action                                                                     |
|-----------|-------------------------------------------------------------------------------|
| any character | filters the branch list by substring (case-insensitive)                   |
| `↑` / `↓` | navigate the filtered list                                                     |
| `Enter`   | checkout the selected branch; if there's no match — create a new branch with the entered name and switch to it |
| `Esc`     | cancel and return to the main screen                                           |

---

## 6. Doctor (`d`)

### Issue list

| Key          | Action                                                     |
|--------------|----------------------------------------------------------------|
| `↑`/`↓`, `j`/`k` | select an issue in the list                                 |
| `Enter`, `f` | go to confirming the fix for the selected issue                  |
| `Esc`, `q`   | close Doctor and return to the main screen                        |

### Fix confirmation

Fixing may delete files and/or modify `.gitignore` — hence a separate confirmation step:

| Key          | Action                                     |
|--------------|-------------------------------------------------|
| `y`, `Enter` | confirm and fix                                  |
| `n`, `Esc`   | cancel, return to the issue list                  |

---

## 7. Stash (`s`)

### Stash list

| Key          | Action                                                                       |
|--------------|-------------------------------------------------------------------------------------|
| `↑`/`↓`, `j`/`k` | select a stash in the list (a preview of its changed files shows on the right/below) |
| `n`          | stash the current uncommitted changes into a new entry                               |
| `Enter`, `p` | **pop** — restore the files from the stash into the working directory and remove the entry |
| `a`          | **apply** — restore the files but keep the entry in the stash                        |
| `x`          | **drop** — go to confirming deletion of the entry without applying it                |
| `Esc`, `q`   | close Stash and return to the main screen                                            |

### Drop confirmation

Deleting a stash without applying it is permanent — hence a separate confirmation:

| Key          | Action                                     |
|--------------|-------------------------------------------------|
| `y`, `Enter` | confirm and delete the stash                     |
| `n`, `Esc`   | cancel, return to the stash list                  |

---

## Cheat sheet (main screen, the most common ones)

```
Tab / Shift+Tab   switch panel
↑↓ or jk          navigate / scroll
space             stage / unstage
enter             stage (Files) · checkout (Branches)
c  b  s  d        commit · branch · stash · doctor
p  P  f           push · pull · fetch
S                 quick pop of the latest stash
y                 copy commit hash (in Log)
r                 refresh
g                 GitHub sign-in
u                 update tgit (when available)
Ctrl+C            quit
```
