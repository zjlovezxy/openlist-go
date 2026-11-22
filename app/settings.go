package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// settings are the user's choices, kept as JSON in the app's data
// directory.
type settings struct {
	LastServer     string `json:"last_server,omitempty"`      // last used OpenList server URL
	LastUsername   string `json:"last_username,omitempty"`    // last used OpenList username
	MPV            string `json:"mpv,omitempty"`              // external player path, found automatically when empty
	DownloadDir    string `json:"download_dir,omitempty"`     // where downloads go, Downloads when empty
	Threads        int    `json:"threads,omitempty"`          // connections per download
	Parallel       int    `json:"parallel,omitempty"`         // downloads at once
	Upload         int    `json:"upload,omitempty"`           // connections per upload
	PreviewMB      int    `json:"preview_mb,omitempty"`       // legacy preview limit
	PreviewImageMB int    `json:"preview_image_mb,omitempty"` // largest picture previewed, in MB
	PreviewTextMB  int    `json:"preview_text_mb,omitempty"`  // largest text previewed, in MB
	PreviewPDFMB   int    `json:"preview_pdf_mb,omitempty"`   // largest PDF previewed, in MB
	DownLimit      int    `json:"down_limit,omitempty"`       // MB/s all downloads together may take, 0 for any
	UpLimit        int    `json:"up_limit,omitempty"`         // MB/s all uploads together may take, 0 for any
	// SkipImport stops signing in with another client's account by
	// itself, once the user signed out.
	SkipImport bool `json:"skip_pcs_import,omitempty"`
	// ShareDest is the folder shares were last saved into.
	ShareDest string `json:"share_dest,omitempty"`
	// SeenShare is the last share link found on the clipboard, and its
	// password, opened once and not again.
	SeenShare string `json:"seen_share,omitempty"`
}

func settingsPath() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "settings.json")
}

func loadSettings() settings {
	var s settings
	if p := settingsPath(); p != "" {
		if data, err := os.ReadFile(p); err == nil {
			json.Unmarshal(data, &s)
		}
	}
	return s
}

func (s settings) save() error {
	p := settingsPath()
	if p == "" {
		return nil
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

func (s settings) threads() int {
	if s.Threads <= 0 {
		return 4
	}
	return s.Threads
}

func (s settings) parallel() int {
	if s.Parallel <= 0 {
		return 3
	}
	return s.Parallel
}

func (s settings) upload() int {
	if s.Upload <= 0 {
		return 2
	}
	return s.Upload
}

func (s settings) imageLimitMB() int {
	if s.PreviewImageMB > 0 {
		return s.PreviewImageMB
	}
	if s.PreviewMB > 0 {
		return s.PreviewMB
	}
	return 50
}

func (s settings) textLimitMB() int {
	if s.PreviewTextMB > 0 {
		return s.PreviewTextMB
	}
	return 8
}

func (s settings) pdfLimitMB() int {
	if s.PreviewPDFMB > 0 {
		return s.PreviewPDFMB
	}
	return 100
}

func (s settings) downloadDir() string {
	if s.DownloadDir != "" {
		return s.DownloadDir
	}
	if d, err := mygo.App.Path(mygo.PathDownloads); err == nil {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads")
}
