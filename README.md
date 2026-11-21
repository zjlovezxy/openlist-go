# OpenListGo

一个用纯 Go 写的 OpenList / AList 桌面客户端：界面由 [MyGo](https://github.com/egoist/mygo) 的原生 `ui` 包在 GPU 上绘制（没有 WebView、没有 cgo），对接 [OpenList REST API](https://doc.oplist.org/guide)（兼容 AList v3）。外观照 macOS / 现代化原生桌面应用设计：侧边栏透出窗口材质、彩色图标块、条纹列表、圆角分组。

## 特性

- **登录**：支持 OpenList / AList 服务器地址 + 账号密码登录，或直接通过 Token / JWT 凭据登录；内置演示模式（无需账号即刻体验）。凭据用 AES-GCM 加密保存，密钥存放在系统钥匙串。
- **浏览**：列表 / 网格两种视图，按名称、大小、日期排序，面包屑路径导航、前进后退、全盘搜索；文件详情（`⌘I`）显示类型、大小、路径和修改时间。
- **不落盘预览**：文件直接读入内存流畅预览
  - 图片：支持 PNG / JPG / WebP / GIF / SVG 等，缩略图直出，滚轮缩放与平移
  - 文本与代码：语法高亮，UTF-8 / GB18030 自动识别，Markdown 实时渲染
  - PDF：基于 WebAssembly 逐页无依赖渲染
  - 音视频：本地流式回环播放或调用本地播放器
- **文件管理**：新建文件夹、重命名、移动、复制、删除与路径选择器
- **传输引擎**：多连接分块下载、断点续传、分片流式上传
- **离线下载**：对接 OpenList 离线下载任务接口，支持查看与管理云端离线任务
- **原生与多端**：通过 MyGo 单一代码库原生支持 Linux、macOS 与 Windows 三端构建。

## 开发与构建

```bash
go tool mygo dev                       # 带热重载运行
OPENLIST_FAKE=1 go run .               # 连接内置假网盘快速调试（免账号）
go tool mygo build -platform linux/amd64   # 构建 Linux amd64 版本（包含 deb/rpm/tar.gz）
```

环境变量：
- `OPENLIST_FAKE=1`：连接内置模拟服务端进行离线测试
- `OPENLIST_CAPTURE=out.png`：窗口启动后自动截屏并退出（配合 `OPENLIST_CAPTURE_AFTER=3s`）


### 发布

每次推送到 `main` 都会自动发版（`.github/workflows/release.yml`）：跑测试 → 版本号自动加一 → 把更新说明写进 [CHANGELOG.md](CHANGELOG.md) 并提交、打 tag → 在 GitHub 上打包 Linux（amd64 / arm64 的 .deb 与 tar.gz）、Windows（amd64 / arm64 安装包与免安装 zip）、macOS（通用版 .dmg）→ 发布 Release。

- 默认升补丁位（0.1.0 → 0.1.1）；提交信息里写 `#minor` / `#major` 升次版本 / 主版本，或在 Actions 里手动 Run workflow 选择；直接改 `mygo.json` 的 `version` 则用你写的版本。
- 更新说明：`CHANGELOG.md` 的 `## Unreleased` 下写了内容就用它，否则按提交记录自动分成「新增 / 修复 / 改进」。
- 只改了 Markdown / docs，或最后一个提交带 `[skip release]`，不发版。
- 发版会往 `main` 推一个 `Release vX.Y.Z` 提交，本地推送前先 `git pull --rebase`。

### 目录

| 路径 | 内容 |
|---|---|
| `app/` | 原生界面：登录、浏览、预览、对话框、传输、分享、设置、动效与着色器 |
| `internal/baidu` | 网盘客户端：登录（含二维码）、列表、直链、文件操作、分享、离线 |
| `internal/auth` | 加密保存登录状态，密钥放在系统钥匙串 |
| `internal/migrate`、`internal/pcsimport` | 沿用百度网盘 Linux 客户端、BaiduPCS-Go 的登录 |
| `internal/remote` | 基于 HTTP Range 的内存读取，预览与边播 |
| `internal/stream` | 给 mpv 用的本地回环服务，把请求拆成有界的分块 |
| `internal/pdf` | PDFium（WebAssembly）渲染 |
| `internal/player` | 查找并启动 mpv |
| `internal/transfer` | 下载 / 上传队列 |
| `internal/tune` | 测速与传输参数调优 |
| `internal/pcsport` | 移植自 BaiduPCS-Go internal 的上传流程 |
| `internal/fakepan` | 测试与演示用的假网盘服务 |
| `internal/selftest` | 真实账号自检 |
| `tools/release` | 发版：算版本号、写更新日志 |
| `tools/appicon` | 绘制应用图标 `resources/icon.png` |
| `docs/course` | 软件工程课程材料：需求、设计、项目管理、AI 开发心得、汇报讲稿 |

## 致谢

见 [NOTICE](NOTICE)。
