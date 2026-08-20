#!/usr/bin/env bash
# install.sh — проверяет зависимости, собирает tgit и кладёт его в PATH,
# чтобы после установки был доступен просто как команда `tgit`.
#
# Использование:
#   ./install.sh                 # установить в ~/.local/bin (без sudo)
#   ./install.sh --system        # установить в /usr/local/bin (нужен sudo)
#   ./install.sh --prefix=DIR    # установить в произвольный каталог
#
# Поддерживаются Linux и macOS. Для Windows используйте install.ps1 (PowerShell).

set -euo pipefail

# ---------- вывод ----------

if [ -t 1 ] && command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ]; then
	c_red=$(tput setaf 1); c_green=$(tput setaf 2); c_yellow=$(tput setaf 3)
	c_bold=$(tput bold); c_reset=$(tput sgr0)
else
	c_red=""; c_green=""; c_yellow=""; c_bold=""; c_reset=""
fi

info()  { printf '%s\n' "${c_bold}==>${c_reset} $*"; }
ok()    { printf '%s\n' "${c_green}✓${c_reset} $*"; }
warn()  { printf '%s\n' "${c_yellow}!${c_reset} $*" >&2; }
err()   { printf '%s\n' "${c_red}✗${c_reset} $*" >&2; }
die()   { err "$*"; exit 1; }

# ---------- аргументы ----------

PREFIX="$HOME/.local/bin"
SYSTEM_INSTALL=0

for arg in "$@"; do
	case "$arg" in
	--system)
		SYSTEM_INSTALL=1
		PREFIX="/usr/local/bin"
		;;
	--prefix=*)
		PREFIX="${arg#--prefix=}"
		;;
	-h | --help)
		sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		die "Неизвестный аргумент: $arg (см. --help)"
		;;
	esac
done

# ---------- определение ОС ----------

case "$(uname -s)" in
Linux*) OS=linux ;;
Darwin*) OS=darwin ;;
MINGW* | MSYS* | CYGWIN*)
	die "Обнаружена Windows-среда. Используй install.ps1 (PowerShell): .\\install.ps1 -AddToPath"
	;;
*)
	die "Неизвестная ОС: $(uname -s). Поддержаны Linux и macOS."
	;;
esac

BIN_NAME="tgit"
OUT_PATH="$PREFIX/$BIN_NAME"

# ---------- проверка git ----------

info "Проверяю git..."
if ! command -v git >/dev/null 2>&1; then
	err "git не найден в PATH."
	case "$OS" in
	darwin) echo "  Установить: brew install git  (или xcode-select --install)" ;;
	linux) echo "  Установить: sudo apt install git   /   sudo dnf install git   /   sudo pacman -S git" ;;
	esac
	exit 1
fi
ok "git найден: $(git --version)"

# ---------- проверка go ----------

info "Проверяю Go..."
if ! command -v go >/dev/null 2>&1; then
	err "Go не найден в PATH."
	case "$OS" in
	darwin) echo "  Установить: brew install go" ;;
	linux) echo "  Установить: sudo apt install golang   /   sudo dnf install golang   /   sudo pacman -S go" ;;
	esac
	echo "  Либо скачать с официальной страницы загрузок Go для вашей ОС."
	exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REQUIRED_GO="$(grep -E '^go [0-9]' "$SCRIPT_DIR/go.mod" | awk '{print $2}')"

version_ge() {
	# $1 >= $2 ? для версий вида X.Y[.Z]
	local IFS=.
	local -a a=($1) b=($2)
	local i ai bi
	for i in 0 1 2; do
		ai=${a[i]:-0}; bi=${b[i]:-0}
		if [ "$ai" -gt "$bi" ]; then return 0; fi
		if [ "$ai" -lt "$bi" ]; then return 1; fi
	done
	return 0
}

CURRENT_GO_RAW="$(go env GOVERSION)"
CURRENT_GO="$(echo "$CURRENT_GO_RAW" | sed -E 's/^go([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/')"

if ! version_ge "$CURRENT_GO" "$REQUIRED_GO"; then
	die "Нужен Go >= $REQUIRED_GO, установлен $CURRENT_GO_RAW."
fi
ok "Go найден: $CURRENT_GO_RAW (нужен >= $REQUIRED_GO)"

# ---------- сборка ----------

info "Собираю tgit..."
VERSION="$(git -C "$SCRIPT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)"

if [ "$SYSTEM_INSTALL" -eq 1 ]; then
	TMP_BIN="$(mktemp -t tgit-build.XXXXXX)"
	(cd "$SCRIPT_DIR" && go build -ldflags "-X main.version=$VERSION" -o "$TMP_BIN" .)
	ok "Сборка готова, версия: $VERSION"

	info "Устанавливаю в $OUT_PATH (нужны права root)..."
	sudo mkdir -p "$PREFIX"
	sudo install -m 755 "$TMP_BIN" "$OUT_PATH"
	rm -f "$TMP_BIN"
else
	mkdir -p "$PREFIX"
	(cd "$SCRIPT_DIR" && go build -ldflags "-X main.version=$VERSION" -o "$OUT_PATH" .)
	chmod 755 "$OUT_PATH"
	ok "Сборка готова, версия: $VERSION"
fi

ok "tgit установлен: $OUT_PATH"

# ---------- проверка ----------

INSTALLED_VERSION_OUTPUT="$("$OUT_PATH" --version 2>&1 || true)"
if [ -n "$INSTALLED_VERSION_OUTPUT" ]; then
	ok "Проверка запуска: $INSTALLED_VERSION_OUTPUT"
else
	warn "Не удалось запустить $OUT_PATH --version — проверьте вручную."
fi

# ---------- PATH ----------

case ":$PATH:" in
*":$PREFIX:"*)
	ok "$PREFIX уже в PATH — команда 'tgit' готова к использованию."
	;;
*)
	warn "$PREFIX отсутствует в PATH."
	echo
	case "${SHELL:-}" in
	*/fish)
		echo "  Выполните один раз — fish сохранит это сам, перезапуск терминала не нужен:"
		echo
		echo "    fish_add_path $PREFIX"
		;;
	*/zsh)
		echo "  Добавьте строку в ~/.zshrc и перезапустите терминал (или выполните: source ~/.zshrc):"
		echo
		echo "    echo 'export PATH=\"$PREFIX:\$PATH\"' >> ~/.zshrc"
		;;
	*)
		echo "  Добавьте строку в конфиг вашей оболочки и перезапустите терминал:"
		echo
		echo "    echo 'export PATH=\"$PREFIX:\$PATH\"' >> ~/.bashrc   # или ~/.zshrc, ~/.profile"
		;;
	esac
	echo
	;;
esac

echo
ok "Готово. Запуск: ${c_bold}tgit${c_reset} из каталога любого git-репозитория."
