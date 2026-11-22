package app

import (
	"github.com/egoist/mygo/ui"

	"github.com/pengstem/baidunetgo/internal/tune"
)

// transferConfig returns the transfer settings in force.
func (a *App) transferConfig() tune.Config {
	return tune.Config{
		Threads:  a.settings.threads(),
		Parallel: a.settings.parallel(),
		Upload:   a.settings.upload(),
		Block:    4 << 20,
		Ahead:    4,
	}
}

// tuneSection is the settings of transfers.
func (a *App) tuneSection(c *ui.Context, t *ui.Theme, p *palette) {
	s := &a.settings
	section(c, t, p, "传输", func() {
		settingRow(c, t, p, "下载位置", "", tildePath(s.downloadDir()), func() {
			if pill(c, t, p, "", "更改…", false).Clicked() {
				a.pickDownloadDir()
			}
		})

		count := func(name, detail string, v *int, presets []int) {
			val := *v
			if val <= 0 {
				switch name {
				case "每个文件的连接数":
					val = s.threads()
				case "同时传输的文件数":
					val = s.parallel()
				case "每个上传的连接数":
					val = s.upload()
				}
			}
			settingRow(c, t, p, name, "", detail, func() {
				if numberChoice(c, t, p, &val, presets, "", name) {
					*v = val
					a.applySettings()
				}
			})
		}

		count("每个文件的连接数", "单文件多线程并发分块数", &s.Threads, []int{1, 2, 4, 8, 16, 32})
		count("同时传输的文件数", "最大并发下载任务数", &s.Parallel, []int{1, 2, 3, 5, 8, 10})
		count("每个上传的连接数", "单文件分块上传连接数", &s.Upload, []int{1, 2, 4, 8, 16})

		limit := func(name string, mbps *int) {
			settingRow(c, t, p, name, "MB/s", "", func() {
				if numberChoice(c, t, p, mbps, []int{0, 1, 2, 5, 10, 20, 50, 100}, "不限速", name) {
					a.applySettings()
				}
			})
		}
		limit("下载限速", &s.DownLimit)
		limit("上传限速", &s.UpLimit)
	})
}

func (a *App) useStreaming(cfg tune.Config) {
	if a.stream != nil {
		a.stream.Ahead = cfg.Ahead
	}
}
