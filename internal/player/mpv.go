// Package player plays netdisk media in external players (mpv, PotPlayer,
// VLC, IINA, etc.), which stream media from the link.
package player

import (
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrNoPlayer is what Find returns when no supported external player is found.
var ErrNoPlayer = errors.New("未找到外部播放器，请在设置中指定它的位置或使用系统默认播放器")

// ErrNoMPV is kept for backward compatibility.
var ErrNoMPV = ErrNoPlayer

// Find returns the path of a player: custom when set, else searches for
// mpv, potplayer, vlc, or iina from PATH or where platform installers put them.
func Find(custom string) (string, error) {
	if custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
		return "", ErrNoPlayer
	}
	for _, name := range []string{"mpv", "vlc", "potplayer", "iina"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/mpv.app/Contents/MacOS/mpv",
			"/Applications/IINA.app/Contents/MacOS/iina-cli",
			"/Applications/VLC.app/Contents/MacOS/VLC",
			"/opt/homebrew/bin/mpv",
			"/usr/local/bin/mpv",
			"/opt/homebrew/bin/vlc",
		}
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if dir := os.Getenv(env); dir != "" {
				candidates = append(candidates,
					filepath.Join(dir, "mpv", "mpv.exe"),
					filepath.Join(dir, "DAUM", "PotPlayer", "PotPlayer64.exe"),
					filepath.Join(dir, "DAUM", "PotPlayer", "PotPlayer.exe"),
					filepath.Join(dir, "VideoLAN", "VLC", "vlc.exe"),
				)
			}
		}
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates,
				filepath.Join(home, "scoop", "shims", "mpv.exe"),
				filepath.Join(home, "scoop", "shims", "potplayer.exe"),
				filepath.Join(home, "scoop", "shims", "vlc.exe"),
			)
		}
	default:
		candidates = []string{
			"/usr/bin/mpv",
			"/usr/local/bin/mpv",
			"/snap/bin/mpv",
			"/var/lib/flatpak/exports/bin/io.mpv.Mpv",
			"/usr/bin/vlc",
			"/usr/local/bin/vlc",
			"/snap/bin/vlc",
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", ErrNoPlayer
}

// Args returns mpv's arguments to play url with the headers it needs,
// titled title.
func Args(url, title string, h http.Header) []string {
	args := []string{
		// Wait for the media before creating the window: an immediate window
		// can trigger a Wayland/Vulkan color-surface conflict when video starts.
		"--force-window=yes",
		"--keep-open=no",
		// Re-read on seeks: cached seeks can lose already-played PGS subtitles
		// with mpv/uosc. Forward buffering remains enabled.
		"--demuxer-seekable-cache=no",
		"--force-media-title=" + title,
		"--title=" + title + " — OpenListGo",
	}
	if ua := h.Get("User-Agent"); ua != "" {
		args = append(args, "--user-agent="+ua)
	}
	// One header at a time: the list option splits on commas.
	for name, values := range h {
		if name == "User-Agent" {
			continue
		}
		for _, v := range values {
			args = append(args, "--http-header-fields-append="+name+": "+v)
		}
	}
	return append(args, "--", url)
}

// PlayerArgs returns the arguments appropriate for playerPath.
func PlayerArgs(playerPath, url, title string, h http.Header) []string {
	base := strings.ToLower(filepath.Base(playerPath))
	if strings.Contains(base, "mpv") {
		return Args(url, title, h)
	}
	return []string{url}
}

// Play starts player playing url. It returns once player has started; player lives
// on after the app quits.
func Play(playerPath, url, title string, h http.Header) error {
	cmd := exec.Command(playerPath, PlayerArgs(playerPath, url, title, h)...)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
