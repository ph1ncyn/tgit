#!/usr/bin/env bash
# install.sh — checks dependencies, builds tgit, and puts it on PATH so it's
# available afterwards as just the `tgit` command.
#
# Usage:
#   ./install.sh                 # install to ~/.local/bin (no sudo)
#   ./install.sh --system        # install to /usr/local/bin (needs sudo)
#   ./install.sh --prefix=DIR    # install to an arbitrary directory
#
# Linux and macOS are supported. For Windows use install.ps1 (PowerShell).

set -euo pipefail

# ---------- output ----------

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

# ---------- arguments ----------

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
		sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		die "Unknown argument: $arg (see --help)"
		;;
	esac
done

# ---------- OS detection ----------

case "$(uname -s)" in
Linux*) OS=linux ;;
Darwin*) OS=darwin ;;
MINGW* | MSYS* | CYGWIN*)
	die "Windows environment detected. Use install.ps1 (PowerShell): .\\install.ps1 -AddToPath"
	;;
*)
	die "Unknown OS: $(uname -s). Linux and macOS are supported."
	;;
esac

BIN_NAME="tgit"
OUT_PATH="$PREFIX/$BIN_NAME"

# ---------- checking git ----------

info "Checking git..."
if ! command -v git >/dev/null 2>&1; then
	err "git not found in PATH."
	case "$OS" in
	darwin) echo "  Install: brew install git  (or xcode-select --install)" ;;
	linux) echo "  Install: sudo apt install git   /   sudo dnf install git   /   sudo pacman -S git" ;;
	esac
	exit 1
fi
ok "git found: $(git --version)"

# ---------- checking go ----------

info "Checking Go..."
if ! command -v go >/dev/null 2>&1; then
	err "Go not found in PATH."
	case "$OS" in
	darwin) echo "  Install: brew install go" ;;
	linux) echo "  Install: sudo apt install golang   /   sudo dnf install golang   /   sudo pacman -S go" ;;
	esac
	echo "  Or download it from the official Go downloads page for your OS."
	exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REQUIRED_GO="$(grep -E '^go [0-9]' "$SCRIPT_DIR/go.mod" | awk '{print $2}')"

version_ge() {
	# is $1 >= $2 ? for versions of the form X.Y[.Z]
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
	die "Go >= $REQUIRED_GO is required, found $CURRENT_GO_RAW."
fi
ok "Go found: $CURRENT_GO_RAW (need >= $REQUIRED_GO)"

# ---------- build ----------

info "Building tgit..."
VERSION="$(git -C "$SCRIPT_DIR" describe --tags --always --dirty 2>/dev/null || echo dev)"

if [ "$SYSTEM_INSTALL" -eq 1 ]; then
	TMP_BIN="$(mktemp -t tgit-build.XXXXXX)"
	(cd "$SCRIPT_DIR" && go build -ldflags "-X main.version=$VERSION" -o "$TMP_BIN" .)
	ok "Build ready, version: $VERSION"

	info "Installing to $OUT_PATH (needs root privileges)..."
	sudo mkdir -p "$PREFIX"
	sudo install -m 755 "$TMP_BIN" "$OUT_PATH"
	rm -f "$TMP_BIN"
else
	mkdir -p "$PREFIX"
	(cd "$SCRIPT_DIR" && go build -ldflags "-X main.version=$VERSION" -o "$OUT_PATH" .)
	chmod 755 "$OUT_PATH"
	ok "Build ready, version: $VERSION"
fi

ok "tgit installed: $OUT_PATH"

# ---------- verification ----------

INSTALLED_VERSION_OUTPUT="$("$OUT_PATH" --version 2>&1 || true)"
if [ -n "$INSTALLED_VERSION_OUTPUT" ]; then
	ok "Launch check: $INSTALLED_VERSION_OUTPUT"
else
	warn "Could not run $OUT_PATH --version — please check manually."
fi

# ---------- PATH ----------

case ":$PATH:" in
*":$PREFIX:"*)
	ok "$PREFIX is already on PATH — the 'tgit' command is ready to use."
	;;
*)
	warn "$PREFIX is not on PATH."
	echo
	case "${SHELL:-}" in
	*/fish)
		echo "  Run this once — fish saves it automatically, no terminal restart needed:"
		echo
		echo "    fish_add_path $PREFIX"
		;;
	*/zsh)
		echo "  Add this line to ~/.zshrc and restart your terminal (or run: source ~/.zshrc):"
		echo
		echo "    echo 'export PATH=\"$PREFIX:\$PATH\"' >> ~/.zshrc"
		;;
	*)
		echo "  Add this line to your shell config and restart your terminal:"
		echo
		echo "    echo 'export PATH=\"$PREFIX:\$PATH\"' >> ~/.bashrc   # or ~/.zshrc, ~/.profile"
		;;
	esac
	echo
	;;
esac

echo
ok "Done. Run: ${c_bold}tgit${c_reset} from inside any git repository."
