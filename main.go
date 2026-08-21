package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tgit/internal/config"
	"tgit/internal/gitrepo"
	"tgit/internal/ui"
	"tgit/internal/update"
)

// version подставляется при сборке через -ldflags "-X main.version=...";
// используется install.sh, чтобы проверить установленный бинарник без запуска TUI.
var version = "dev"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("tgit " + version)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tgit:", err)
		os.Exit(1)
	}

	// Ошибка "не git-репозиторий" не фатальна: показываем интерфейс всё равно —
	// вместо главного экрана появится экран с предложением открыть недавний
	// проект tgit или склонировать репозиторий сюда (см. internal/ui.noRepoModel).
	repo, _ := gitrepo.Open(wd)
	if repo != nil {
		_ = config.AddRecent(repo.Root)
	}

	app := ui.NewApp(repo, wd)

	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tgit:", err)
		os.Exit(1)
	}

	// Обновление применяется, пока tgit ещё работает (git pull + go build +
	// подмена файла), но перезапуск в новый бинарник откладывается до сюда —
	// после p.Run() терминал уже гарантированно возвращён в обычный режим,
	// так что новый процесс унаследует его чистым, без разрыва raw/alt-screen.
	if fm, ok := finalModel.(ui.App); ok {
		if exe := fm.RestartExe(); exe != "" {
			if err := update.Restart(exe); err != nil {
				fmt.Fprintln(os.Stderr, "tgit: restart after update failed:", err)
				os.Exit(1)
			}
		}
	}
}
