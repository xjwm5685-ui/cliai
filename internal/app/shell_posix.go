package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	bashIntegrationMarker = "# >>> cliai bash integration >>>"
	zshIntegrationMarker  = "# >>> cliai zsh integration >>>"
	integrationEndMarker  = "# <<< cliai shell integration <<<"
)

func runShellInstallPOSIX(shell string, stdout io.Writer, stderr io.Writer) int {
	snippet, err := shellInitSnippet(shell)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	profilePath, err := shellProfilePath(shell)
	if err != nil {
		fmt.Fprintf(stderr, "resolve %s profile: %v\n", shell, err)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		fmt.Fprintf(stderr, "create profile directory: %v\n", err)
		return 1
	}

	if err := upsertShellIntegration(profilePath, shellIntegrationBlock(shell, snippet)); err != nil {
		fmt.Fprintf(stderr, "write %s integration: %v\n", shell, err)
		return 1
	}

	fmt.Fprintf(stdout, "Installed cliai %s integration to %s\n", shell, profilePath)
	fmt.Fprintf(stdout, "Reload your shell with: source %s\n", profilePath)
	return 0
}

func shellInitSnippet(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "powershell":
		return powershellSnippet(), nil
	case "bash":
		return bashSnippet(), nil
	case "zsh":
		return zshSnippet(), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

func shellProfilePath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell profile: %s", shell)
	}
}

func shellIntegrationBlock(shell string, snippet string) string {
	marker := bashIntegrationMarker
	if shell == "zsh" {
		marker = zshIntegrationMarker
	}

	return marker + "\n" + snippet + "\n" + integrationEndMarker + "\n"
}

func upsertShellIntegration(filePath string, block string) error {
	existing, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(existing)
	startIndex := strings.Index(content, strings.SplitN(block, "\n", 2)[0])
	endIndex := strings.Index(content, integrationEndMarker)
	if startIndex >= 0 && endIndex >= 0 && endIndex >= startIndex {
		endIndex += len(integrationEndMarker)
		updated := content[:startIndex] + block
		if endIndex < len(content) {
			updated += strings.TrimLeft(content[endIndex:], "\r\n")
			if !strings.HasSuffix(updated, "\n") {
				updated += "\n"
			}
		}
		return os.WriteFile(filePath, []byte(updated), 0o644)
	}

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		content += "\n"
	}
	content += block
	return os.WriteFile(filePath, []byte(content), 0o644)
}

func bashSnippet() string {
	return `if [[ -z "${__CLIAI_BASH_LOADED:-}" ]]; then
  __CLIAI_BASH_LOADED=1

  __cliai_bash_query() {
    local buffer="$1"
    if [[ -z "${buffer//[[:space:]]/}" ]]; then
      return 0
    fi

    CLIAI_SHELL=bash cliai predict --shell bash --command-only --cwd "$PWD" -- "$buffer" 2>/dev/null
  }

  __cliai_bash_accept() {
    local buffer="${READLINE_LINE}"
    local suggestion

    suggestion="$(__cliai_bash_query "${buffer}")" || return 0
    if [[ -z "${suggestion}" || "${suggestion}" == "${buffer}" ]]; then
      return 0
    fi

    READLINE_LINE="${suggestion}"
    READLINE_POINT=${#READLINE_LINE}
  }

  __cliai_bash_accept_word() {
    local buffer="${READLINE_LINE}"
    local suggestion suffix trimmed leading word addition

    suggestion="$(__cliai_bash_query "${buffer}")" || return 0
    if [[ -z "${suggestion}" || "${suggestion}" == "${buffer}" || "${suggestion}" != "${buffer}"* ]]; then
      return 0
    fi

    suffix="${suggestion#${buffer}}"
    leading="${suffix%%[^[:space:]]*}"
    trimmed="${suffix#"${leading}"}"
    word="${trimmed%%[[:space:]]*}"
    addition="${leading}${word}"
    if [[ -z "${addition}" ]]; then
      addition="${suffix}"
    fi

    READLINE_LINE="${buffer}${addition}"
    READLINE_POINT=${#READLINE_LINE}
  }

  csg() { cliai predict --shell bash --limit 5 -- "$*"; }
  csi() { cliai predict --shell bash --interactive --copy -- "$*"; }
  csc() { cliai predict --shell bash --command-only -- "$*"; }

  bind -x '"\e[1;3C":__cliai_bash_accept'
  bind -x '"\e[1;4C":__cliai_bash_accept_word'
  bind -x '"\e\e[C":__cliai_bash_accept'
  bind -x '"\ef":__cliai_bash_accept_word'
fi`
}

func zshSnippet() string {
	return `if [[ -z "${__CLIAI_ZSH_LOADED:-}" ]]; then
  typeset -g __CLIAI_ZSH_LOADED=1
  typeset -g __cliai_zsh_suggestion=""
  typeset -g __cliai_zsh_last_buffer=""
  typeset -g __cliai_zsh_highlight_style="${CLIAI_ZSH_HIGHLIGHT_STYLE:-fg=8}"

  __cliai_zsh_clear() {
    POSTDISPLAY=""
    region_highlight=()
    __cliai_zsh_suggestion=""
  }

  __cliai_zsh_query() {
    emulate -L zsh
    local buffer="$1"
    if [[ -z "${buffer//[[:space:]]/}" ]]; then
      return 0
    fi

    CLIAI_SHELL=zsh cliai predict --shell zsh --command-only --cwd "$PWD" -- "$buffer" 2>/dev/null
  }

  __cliai_zsh_refresh() {
    emulate -L zsh

    if (( CURSOR != ${#BUFFER} )); then
      __cliai_zsh_last_buffer="$BUFFER"
      __cliai_zsh_clear
      return
    fi

    if [[ "$BUFFER" == "$__cliai_zsh_last_buffer" ]]; then
      return
    fi
    __cliai_zsh_last_buffer="$BUFFER"

    if [[ -z "${BUFFER//[[:space:]]/}" ]]; then
      __cliai_zsh_clear
      return
    fi

    local suggestion="$(__cliai_zsh_query "$BUFFER")"
    if [[ -z "$suggestion" || "$suggestion" == "$BUFFER" || "$suggestion" != "$BUFFER"* ]]; then
      __cliai_zsh_clear
      return
    fi

    local suffix="${suggestion#$BUFFER}"
    __cliai_zsh_suggestion="$suggestion"
    POSTDISPLAY="$suffix"
    region_highlight=("${#BUFFER} $(( ${#BUFFER} + ${#POSTDISPLAY} )) ${__cliai_zsh_highlight_style}")
  }

  __cliai_zsh_accept() {
    emulate -L zsh
    if [[ -n "$__cliai_zsh_suggestion" && "$__cliai_zsh_suggestion" == "$BUFFER"* && $CURSOR -eq ${#BUFFER} ]]; then
      BUFFER="$__cliai_zsh_suggestion"
      CURSOR=${#BUFFER}
      __cliai_zsh_last_buffer=""
      __cliai_zsh_refresh
    else
      zle forward-char
    fi
  }

  __cliai_zsh_accept_word() {
    emulate -L zsh
    if [[ -z "$__cliai_zsh_suggestion" || "$__cliai_zsh_suggestion" != "$BUFFER"* || $CURSOR -ne ${#BUFFER} ]]; then
      zle forward-word
      return
    fi

    local suffix="${__cliai_zsh_suggestion#$BUFFER}"
    local leading="${suffix%%[^[:space:]]*}"
    local trimmed="${suffix#"${leading}"}"
    local word="${trimmed%%[[:space:]]*}"
    local addition="${leading}${word}"
    if [[ -z "$addition" ]]; then
      addition="$suffix"
    fi

    BUFFER+="$addition"
    CURSOR=${#BUFFER}
    __cliai_zsh_last_buffer=""
    __cliai_zsh_refresh
  }

  csg() { cliai predict --shell zsh --limit 5 -- "$*"; }
  csi() { cliai predict --shell zsh --interactive --copy -- "$*"; }
  csc() { cliai predict --shell zsh --command-only -- "$*"; }

  autoload -Uz add-zle-hook-widget
  add-zle-hook-widget line-pre-redraw __cliai_zsh_refresh
  add-zle-hook-widget line-init __cliai_zsh_refresh
  add-zle-hook-widget keymap-select __cliai_zsh_refresh

  zle -N __cliai_zsh_accept
  zle -N __cliai_zsh_accept_word

  bindkey '^[[C' __cliai_zsh_accept
  bindkey '^[[1;3C' __cliai_zsh_accept
  bindkey '^[f' __cliai_zsh_accept_word
  bindkey '^[[1;4C' __cliai_zsh_accept_word
fi`
}
