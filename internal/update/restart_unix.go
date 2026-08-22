//go:build !windows

package update

import (
	"os"
	"syscall"
)

// restart заменяет образ текущего процесса на exe (тот же PID), унаследовав
// терминал как есть. В отличие от os/exec.Command(...).Start(), который
// породил бы НОВЫЙ процесс, а старый tgit пришлось бы отдельно завершать —
// syscall.Exec не создаёт нового процесса вообще, так что оболочка,
// запустившая tgit, никогда не видит смерть "старого" процесса и не
// перехватывает управление терминалом обратно себе. Именно эта гонка (новый
// процесс успевал стартовать, но ещё не был foreground process group
// терминала) давала "error entering raw mode: input/output error" (EIO —
// tcsetattr() из процесса не в foreground) при перезапуске после
// обновления — и на macOS, и на Linux, поскольку это общее свойство POSIX
// job control, а не особенность конкретной ОС.
func restart(exe string) error {
	argv := append([]string{exe}, os.Args[1:]...)
	return syscall.Exec(exe, argv, os.Environ())
}
