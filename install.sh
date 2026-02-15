#!/bin/sh
set -eu

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

bin_dir="$HOME/.fitz/bin"
bin_path="$bin_dir/fitz"
asset="fitz_${os}_${arch}"
url="https://github.com/alexneyler/fitz/releases/latest/download/$asset"

mkdir -p "$bin_dir"
tmp="$(mktemp "$bin_dir/.fitz.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
curl -fsSL "$url" -o "$tmp"
chmod 755 "$tmp"
mv "$tmp" "$bin_path"

rc_file="${HOME}/.bashrc"
shell_name="${SHELL##*/}"
if [ "$shell_name" = "zsh" ] || [ -f "${HOME}/.zshrc" ]; then
  rc_file="${HOME}/.zshrc"
  completion_line='eval "$(fitz completion zsh)"'
else
  completion_line='eval "$(fitz completion bash)"'
fi

ensure_line() {
  line="$1"
  file="$2"
  touch "$file"
  grep -Fqx "$line" "$file" || printf '%s\n' "$line" >> "$file"
}

ensure_line 'export PATH="$HOME/.fitz/bin:$PATH"' "$rc_file"
ensure_line "$completion_line" "$rc_file"
