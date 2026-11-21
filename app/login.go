package app

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/pengstem/baidunetgo/internal/baidu"
	"github.com/pengstem/baidunetgo/internal/fakepan"
)

// loginState is the sign-in screen.
type loginState struct {
	tab int // 0: 账号密码, 1: Token 登录

	server   string
	username string
	password string
	token    string

	busy bool
	err  string
}

func (l *loginState) stopQR() {}

// loginView is the sign-in screen for OpenList.
func (a *App) loginView(c *ui.Context, t *ui.Theme, p *palette) {
	l := &a.login
	if l.server == "" {
		l.server = "http://localhost:5244"
	}
	if l.username == "" && l.tab == 0 {
		l.username = "admin"
	}

	root := ui.Box(c).Fill().Background(t.Background).Clip()
	root.Children(func() {
		// The top drags the window.
		ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(barHeight(c)).DragWindow()

		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Center().Children(func() {
			ui.Column(c).Key("login").Width(360).Gap(16).AlignItems(ui.Center).Transition(popIn).Children(func() {
				ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
					logoMark(c, t, p, 56).Margin(0, 0, 4)
					ui.Text(c, "登录 OpenList").FontSize(22).FontWeight(700)
					ui.Text(c, "连接自建或公有 OpenList / AList 网盘服务").FontSize(12).TextColor(t.TextMuted)
				})

				segmented(c, t, p, &l.tab, "账号密码登录", "Token 凭据登录").FillWidth()

				if l.tab == 0 {
					a.passwordPanel(c, t, p)
				} else {
					a.tokenPanel(c, t, p)
				}

				if l.err != "" {
					ui.Text(c, l.err).TextColor(t.Danger).FontSize(12).TextAlign(ui.Center)
				}

				a.loginFooter(c, t, p)
			})
		})
	})
}

func (a *App) passwordPanel(c *ui.Context, t *ui.Theme, p *palette) {
	l := &a.login
	card := ui.Column(c).FillWidth().Gap(10).Padding(16).Radius(12).
		Background(t.Surface).Border(1, p.Hairline).Shadow(0, 4, 16, -4, p.Shadow)

	card.Children(func() {
		ui.Text(c, "服务器地址").FontSize(12).TextColor(t.TextMuted).FontWeight(500)
		ui.TextInput(c, &l.server).Placeholder("http://localhost:5244").Height(34).FillWidth()

		ui.Text(c, "用户名").FontSize(12).TextColor(t.TextMuted).FontWeight(500)
		ui.TextInput(c, &l.username).Placeholder("admin").Height(34).FillWidth()

		ui.Text(c, "密码").FontSize(12).TextColor(t.TextMuted).FontWeight(500)
		ui.TextInput(c, &l.password).Password().Placeholder("请输入密码").Height(34).FillWidth()

		btn := pill(c, t, p, "user", "登 录", true).Height(34).FillWidth().
			Disabled(l.busy || strings.TrimSpace(l.server) == "" || strings.TrimSpace(l.username) == "")

		if l.busy {
			btn.Children(func() { ui.Spinner(c).TextColor(t.AccentText) })
		}

		if btn.Clicked() {
			srv := strings.TrimRight(strings.TrimSpace(l.server), "/")
			if !strings.HasPrefix(srv, "http://") && !strings.HasPrefix(srv, "https://") {
				srv = "http://" + srv
			}
			l.err, l.busy = "", true
			cr := baidu.Credentials{
				Server:   srv,
				Username: strings.TrimSpace(l.username),
				Password: l.password,
			}
			go a.signIn(cr, true)
		}
	})
}

func (a *App) tokenPanel(c *ui.Context, t *ui.Theme, p *palette) {
	l := &a.login
	card := ui.Column(c).FillWidth().Gap(10).Padding(16).Radius(12).
		Background(t.Surface).Border(1, p.Hairline).Shadow(0, 4, 16, -4, p.Shadow)

	card.Children(func() {
		ui.Text(c, "服务器地址").FontSize(12).TextColor(t.TextMuted).FontWeight(500)
		ui.TextInput(c, &l.server).Placeholder("http://localhost:5244").Height(34).FillWidth()

		ui.Text(c, "OpenList Token / JWT").FontSize(12).TextColor(t.TextMuted).FontWeight(500)
		ui.TextArea(c, &l.token).Placeholder("openlist-...").Lines(3, 4).Font("monospace").
			FontSize(12).FillWidth()

		btn := pill(c, t, p, "link", "Token 登录", true).Height(34).FillWidth().
			Disabled(l.busy || strings.TrimSpace(l.server) == "" || strings.TrimSpace(l.token) == "")

		if l.busy {
			btn.Children(func() { ui.Spinner(c).TextColor(t.AccentText) })
		}

		if btn.Clicked() {
			srv := strings.TrimRight(strings.TrimSpace(l.server), "/")
			if !strings.HasPrefix(srv, "http://") && !strings.HasPrefix(srv, "https://") {
				srv = "http://" + srv
			}
			l.err, l.busy = "", true
			cr := baidu.Credentials{
				Server: srv,
				Token:  strings.TrimSpace(l.token),
			}
			go a.signIn(cr, true)
		}
	})
}

func (a *App) loginFooter(c *ui.Context, t *ui.Theme, p *palette) {
	demoBtn := ui.ButtonBase(c).Label("进入演示网盘 (无需账号)").Height(26).PaddingX(10).Radius(4).
		TextColor(t.Accent).Disabled(a.login.busy)
	if demoBtn.Hovered() {
		demoBtn.TextColor(t.AccentHover)
	}
	demoBtn.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, icon("play")).FontSize(13)
			ui.Text(c, "进入演示网盘 (无需账号)").FontSize(12).FontWeight(500)
		})
	})
	if demoBtn.Clicked() {
		srv := fakepan.New()
		a.fake = true
		go a.signInWith(baidu.OpenAt(baidu.Credentials{Server: srv.URL, Token: "fake-jwt", Username: "admin"}, srv.URL), false)
	}

	ui.Text(c, "凭据加密保存在本机系统钥匙串").TextColor(t.TextMuted).FontSize(11)
}

// logoMark is the app's mark.
func logoMark(c *ui.Context, t *ui.Theme, p *palette, size float32) ui.Element {
	cell := size * 0.22
	gap := size * 0.08
	top, bottom := ui.Hex("#2563eb"), ui.Hex("#1d4ed8") // Modern vibrant blue
	return ui.Box(c).Size(size, size).Draw(func(pt *ui.Painter, r ui.Rect) {
		pt.Shadow(r, size*0.24, 0, size*0.04, size*0.12, 0, p.Shadow)
		pt.FillGradient(r, ui.LinearGradient{From: top, To: bottom, Angle: 180}, size*0.24)
		x0 := r.X + (r.W-2*cell-gap)/2
		y0 := r.Y + (r.H-2*cell-gap)/2
		for i := range 4 {
			color := ui.Hex("#ffffff")
			if i == 3 {
				color = color.Alpha(0.6)
			}
			x := x0 + float32(i%2)*(cell+gap)
			y := y0 + float32(i/2)*(cell+gap)
			pt.Fill(ui.Rect{X: x, Y: y, W: cell, H: cell}, color, size*0.06)
		}
	})
}
