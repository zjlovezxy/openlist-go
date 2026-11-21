package app

import (
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/egoist/mygo/ui"
)

// isMobile reports whether the window is currently rendering for a mobile device
// (either on Android/iOS, forced by OPENLIST_MOBILE=1, or resized narrow).
func (a *App) isMobile(c *ui.Context) bool {
	if runtime.GOOS == "android" || runtime.GOOS == "ios" || os.Getenv("OPENLIST_MOBILE") == "1" {
		return true
	}
	w, _ := c.Size()
	return w > 0 && w < 680
}

// mobileShell renders the mobile layout: Safe Area aware TopBar + Router View + Bottom TabBar.
func (a *App) mobileShell(c *ui.Context, t *ui.Theme, p *palette) {
	a.sampleSpeed(c)
	top, _, bottom, _ := c.SafeArea()

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		// 1. Mobile Top Bar
		a.mobileTopBar(c, t, p, top)

		// 2. Main content view
		ui.Column(c).Grow(1).FillWidth().Children(func() {
			a.router.View(c, func(r *ui.Route) {
				switch {
				case r.Match("/files"), r.Match("/files/{path...}"):
					dir := dirOfRoute(r.Param("path"))
					r.Title(dirLabel(dir))
					a.filesPage(c, t, p, dir)
				case r.Match("/transfers"):
					r.Title("传输")
					a.transfersPage(c, t, p)
				case r.Match("/shares"):
					r.Title("我的分享")
					a.sharesPage(c, t, p)
				case r.Match("/offline"):
					r.Title("离线下载")
					a.offlinePage(c, t, p)
				case r.Match("/recycle"):
					r.Title("回收站")
					a.recyclePage(c, t, p)
				case r.Match("/search"):
					r.Title("搜索")
					a.searchPage(c, t, p, a.router.Query("q"))
				case r.Match("/settings"):
					r.Title("设置")
					a.settingsPage(c, t, p)
				}
			})
		})

		// 3. Mobile Bottom Navigation Bar
		a.mobileBottomBar(c, t, p, bottom)
	})
}

// mobileTopBar is the top header for mobile phones, adapting per page.
func (a *App) mobileTopBar(c *ui.Context, t *ui.Theme, p *palette, top float32) {
	dir := a.currentDir()
	curPath := a.router.Path()
	isSubFiles := dir != "" && dir != "/"

	ui.Column(c).FillWidth().Background(p.Card).BorderWidth(0, 0, 1, 0).BorderColor(p.Hairline).Shrink(0).Children(func() {
		if top > 0 {
			ui.Box(c).FillWidth().Height(top)
		}

		ui.Row(c).Height(50).PaddingX(12).AlignItems(ui.Center).Gap(8).Children(func() {
			if curPath == "/search" {
				// Search input bar
				back := ui.ButtonBase(c).Size(36, 36).Radius(18).Center().TextColor(t.Text)
				if back.Hovered() {
					back.Background(p.Hover)
				}
				back.Children(func() {
					ui.Icon(c, icon("chevron-left")).FontSize(20)
				})
				if back.Clicked() {
					a.router.Back()
				}

				in := ui.TextInputBase(c, &a.query).Placeholder("搜索网盘文件...").Grow(1).Height(34)
				if in.Submitted() && a.query != "" {
					a.search(a.query)
				}

				if a.query != "" {
					clearBtn := ui.ButtonBase(c).Size(26, 26).Radius(13).Center().TextColor(t.TextMuted)
					clearBtn.Children(func() { ui.Icon(c, icon("x")).FontSize(14) })
					if clearBtn.Clicked() {
						a.query = ""
					}
				}

				searchBtn := ui.ButtonBase(c).Padding(5, 12).Radius(8).Background(t.Accent).TextColor(t.AccentText)
				searchBtn.Children(func() { ui.Text(c, "搜索").FontSize(13).FontWeight(600) })
				if searchBtn.Clicked() && a.query != "" {
					a.search(a.query)
				}
				return
			}

			// Leading: Back button if in subfolder, else icon
			if isSubFiles {
				back := ui.ButtonBase(c).Size(36, 36).Radius(18).Center().TextColor(t.Text)
				if back.Hovered() {
					back.Background(p.Hover)
				}
				back.Children(func() {
					ui.Icon(c, icon("chevron-left")).FontSize(20)
				})
				if back.Clicked() {
					parent := path.Dir(dir)
					a.router.Push(filesRoute(parent))
				}
			} else {
				ui.Box(c).Size(28, 28).Radius(7).Background(t.Accent).Center().Children(func() {
					ui.Icon(c, icon("hard-drive")).FontSize(16).TextColor(ui.Hex("#ffffff"))
				})
			}

			// Title & Subtitle
			title := a.router.Title()
			if isSubFiles {
				title = path.Base(dir)
			}
			if title == "" {
				title = "OpenList"
			}

			ui.Column(c).Grow(1).Basis(0).Gap(1).Children(func() {
				ui.Text(c, title).FontSize(16).FontWeight(600).SingleLine()
				if isSubFiles {
					ui.Text(c, dir).FontSize(11).TextColor(t.TextMuted).SingleLine().TruncateStart()
				} else if sub := a.subtitle(); sub != "" {
					ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
				}
			})

			// Trailing buttons
			if dir != "" {
				// Search button
				searchBtn := ui.ButtonBase(c).Size(36, 36).Radius(8).Center().TextColor(t.TextMuted)
				if searchBtn.Hovered() {
					searchBtn.Background(p.Hover).TextColor(t.Text)
				}
				searchBtn.Children(func() {
					ui.Icon(c, icon("search")).FontSize(18)
				})
				if searchBtn.Clicked() {
					a.router.Push("/search")
				}

				// Grid / List View Toggle
				viewIcon := "layout-grid"
				if a.gridView {
					viewIcon = "list"
				}
				viewBtn := ui.ButtonBase(c).Size(36, 36).Radius(8).Center().TextColor(t.TextMuted)
				if viewBtn.Hovered() {
					viewBtn.Background(p.Hover).TextColor(t.Text)
				}
				viewBtn.Children(func() {
					ui.Icon(c, icon(viewIcon)).FontSize(18)
				})
				if viewBtn.Clicked() {
					a.gridView = !a.gridView
				}

				// Add (+) button with menu
				addBtn := ui.ButtonBase(c).Size(34, 34).Radius(17).Center().Background(t.Accent).TextColor(t.AccentText)
				addBtn.Children(func() {
					ui.Icon(c, icon("plus")).FontSize(18)
				})
				menuButton(addBtn, func(m *ui.Menu) {
					if m.Item("新建文件夹").Chosen() {
						l := a.listingFor(dir)
						a.newFolder(l, addBtn.Bounds())
					}
					if m.Item("上传文件").Chosen() {
						a.pickUpload(dir, addBtn.Bounds())
					}
					m.Separator()
					if m.Item("刷新").Chosen() {
						a.reload(dir)
					}
				})
			} else if curPath == "/transfers" {
				m := a.xfer
				if m != nil {
					if a.activeTransfers() > 0 {
						pauseBtn := ui.ButtonBase(c).Padding(4, 10).Radius(6).Background(p.Hover).TextColor(t.Text)
						pauseBtn.Children(func() { ui.Text(c, "全部暂停").FontSize(12) })
						if pauseBtn.Clicked() {
							m.PauseAll()
						}
					} else {
						resumeBtn := ui.ButtonBase(c).Padding(4, 10).Radius(6).Background(p.Hover).TextColor(t.Text)
						resumeBtn.Children(func() { ui.Text(c, "全部继续").FontSize(12) })
						if resumeBtn.Clicked() {
							m.ResumeAll()
						}
					}
				}
			} else if curPath == "/offline" {
				addBtn := ui.ButtonBase(c).Size(34, 34).Radius(17).Center().Background(t.Accent).TextColor(t.AccentText)
				addBtn.Children(func() { ui.Icon(c, icon("plus")).FontSize(18) })
				if addBtn.Clicked() {
					a.ask(dlgOffline, "/", nil)
					a.offline.picker = picker{dir: "/"}
				}
			} else if curPath == "/recycle" {
				if len(a.recycle.items) > 0 {
					cleanBtn := ui.ButtonBase(c).Padding(4, 10).Radius(6).Background(t.Danger.Alpha(0.12)).TextColor(t.Danger)
					cleanBtn.Children(func() { ui.Text(c, "清空").FontSize(12).FontWeight(600) })
					if cleanBtn.Clicked() {
						a.recycle.confirm = true
					}
				}
			} else if curPath == "/shares" {
				addBtn := ui.ButtonBase(c).Padding(4, 10).Radius(6).Background(t.Accent).TextColor(t.AccentText)
				addBtn.Children(func() { ui.Text(c, "转存链接").FontSize(12).FontWeight(600) })
				if addBtn.Clicked() {
					a.openLink("", false)
				}
			}
		})
	})
}

// mobileBottomBar is the bottom tab bar with safe-area support.
func (a *App) mobileBottomBar(c *ui.Context, t *ui.Theme, p *palette, bottom float32) {
	tabs := []struct {
		path  string
		label string
		icon  string
	}{
		{"/files", "文件", "folder"},
		{"/transfers", "传输", "cloud-download"},
		{"/shares", "分享", "share-2"},
		{"/offline", "离线", "clock"},
		{"/settings", "设置", "settings"},
	}

	curPath := a.router.Path()
	activeTab := "/files"
	for _, tab := range tabs {
		if curPath == tab.path || strings.HasPrefix(curPath, tab.path+"/") {
			activeTab = tab.path
			break
		}
	}

	ui.Column(c).FillWidth().Background(p.Card).BorderWidth(1, 0, 0, 0).BorderColor(p.Hairline).Shrink(0).Children(func() {
		ui.Row(c).Height(54).FillWidth().AlignItems(ui.Center).Children(func() {
			for _, tab := range tabs {
				on := tab.path == activeTab
				tb := tab
				item := ui.ButtonBase(c.Key("mobile-tab-" + tb.path)).Grow(1).Height(54).Center().
					TextColor(map[bool]ui.Color{true: t.Accent, false: t.TextMuted}[on])
				if item.Hovered() {
					item.Background(p.Hover)
				}
				item.Children(func() {
					ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
						iconBox := ui.Box(c).Size(24, 24).Center()
						iconBox.Children(func() {
							ui.Icon(c, icon(tb.icon)).FontSize(20)
							if tb.path == "/transfers" {
								if n := a.activeTransfers(); n > 0 {
									ui.Box(c).Absolute().Top(-4).Right(-8).Height(14).MinWidth(14).Radius(7).
										PaddingX(3).Background(t.Accent).Center().Children(func() {
										ui.Text(c, itoa(n)).FontSize(9).FontWeight(700).TextColor(ui.Hex("#ffffff"))
									})
								}
							}
						})
						ui.Text(c, tb.label).FontSize(11).FontWeight(map[bool]int{true: 600, false: 400}[on])
					})
				})
				if item.Clicked() && activeTab != tb.path {
					a.router.Push(tb.path)
				}
			}
		})
		if bottom > 0 {
			ui.Box(c).FillWidth().Height(bottom)
		}
	})
}

// mobileFileList shows a touch-optimized file list for mobile phones.
func (a *App) mobileFileList(c *ui.Context, t *ui.Theme, p *palette, l *listing) {
	a.bindListState(l)
	now := c.Now()
	mobileRowH := float32(58)

	list := ui.List(c, &l.list, len(l.files)+1).Grow(1).Padding(4, 10, 20)
	list.Rows(func(row ui.ListRow) {
		c := row.Context
		if row.Index == len(l.files) {
			ui.Column(c).Padding(16, 0, 24).Center().Children(func() {
				count := itoa(len(l.files)) + " 个项目"
				if len(l.files) == 0 {
					count = "文件夹为空"
				}
				ui.Text(c, count).FontSize(12).TextColor(t.TextMuted)
			})
			return
		}
		f := &l.files[row.Index]
		r := ui.Row(c).Height(mobileRowH).PaddingX(12).Gap(12).AlignItems(ui.Center).Radius(10).TextColor(t.Text)
		passing, gone := a.passing(r, f, t, now)
		draft := f.FsID == draftID
		a.awaited(r, l, f, now)
		if draft {
			l.draftAt = r.Bounds()
		}

		switch {
		case passing:
			r.Height(mobileRowH * (1 - smoothstep(0.7, 1, gone)))
		case draft:
			r.Background(t.Background.Mix(t.Accent, 0.08))
		case row.Selected():
			r.Background(t.Background.Mix(t.Accent, 0.12))
		case r.Hovered():
			r.Background(p.Hover)
		case row.Index%2 == 1:
			r.Background(t.Background.Mix(t.Text, 0.02))
		default:
			r.Background(t.Background)
		}

		if in := entering(r, l, row.Index, now, rowStagger); in < 1 {
			r.Opacity(in).Translate(0, (1-in)*10)
		}

		r.Children(func() {
			// Icon
			a.fileIcon(c, t, p, f, 32)

			// Details
			ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
				if draft {
					a.makerField(c, t, l, false)
				} else if a.renamingIn(l, f) {
					a.nameField(c, t, l, f, false)
				} else {
					ui.Text(c, f.Name).FontSize(14.5).FontWeight(500).SingleLine()
				}
				sub := stampTime(f.Mtime, now)
				if !f.IsDir {
					sub = humanSize(f.Size) + " · " + sub
				} else {
					sub = "文件夹 · " + sub
				}
				ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
			})

			// Trailing buttons: ellipsis menu and folder indicator
			if !passing && !draft {
				actionBtn := ui.ButtonBase(c.Key(f.FsID)).Size(36, 36).Radius(18).Center().TextColor(t.TextMuted)
				if actionBtn.Hovered() {
					actionBtn.Background(p.Hover).TextColor(t.Text)
				}
				actionBtn.Children(func() {
					ui.Icon(c, icon("ellipsis")).FontSize(18)
				})
				menuButton(actionBtn, func(m *ui.Menu) {
					a.buildFileMenu(m, l, *f)
				})

				if f.IsDir {
					ui.Icon(c, icon("chevron-right")).FontSize(16).TextColor(p.Faint)
				}
			}
		})

		if !passing && !draft {
			contextMenu(r, func(m *ui.Menu) {
				a.buildFileMenu(m, l, *f)
			})
		}

		// Single tap on row opens it directly
		if r.Clicked() && !passing && !draft && !a.renamingIn(l, f) {
			a.open(*f)
		}
	})
}
