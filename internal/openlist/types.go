package openlist

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// Categories of files, as the netdisk sorts them.
const (
	CategoryOther   = 0
	CategoryVideo   = 1
	CategoryAudio   = 2
	CategoryImage   = 3
	CategoryDoc     = 4
	CategoryApp     = 5
	CategoryCode    = 6
	CategoryTorrent = 7
)

// Common errors.
var (
	ErrNotLoggedIn   = errors.New("未登录或 Token 已失效")
	ErrExists        = errors.New("文件已存在")
	ErrNoSpace       = errors.New("存储空间不足")
	ErrSharePassword = errors.New("提取码错误")
)

// HTTPError is an HTTP error status.
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body)
}

// Expired reports whether an error means a link or token must be renewed.
func Expired(err error) bool {
	if err == nil {
		return false
	}
	var he *HTTPError
	if errors.As(err, &he) {
		return he.Status == http.StatusUnauthorized || he.Status == http.StatusForbidden
	}
	return errors.Is(err, ErrNotLoggedIn)
}

// File represents a file or directory in OpenList.
type File struct {
	FsID       int64  `json:"fs_id"`
	Path       string `json:"path"`
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"is_dir"`
	Ctime      int64  `json:"created"`
	Mtime      int64  `json:"modified"`
	MD5        string `json:"md5"`
	Category   int    `json:"category"`
	Thumb      string `json:"thumb"`
	ThumbSmall string `json:"thumb_small"`
	RawURL     string `json:"raw_url,omitempty"`
	Sign       string `json:"sign,omitempty"`
}

// Order specifies sorting order.
type Order struct {
	By   string // "name", "time", "size"
	Desc bool
}

// Quota holds disk quota information.
type Quota struct {
	Total int64 `json:"total"`
	Used  int64 `json:"used"`
}

func (q Quota) Free() int64 {
	if q.Total <= 0 {
		return 1 << 40 // 1TB default display if unlimited
	}
	return q.Total - q.Used
}

// User represents the signed-in user.
type User struct {
	ID       int64     `json:"id"`
	UID      uint64    `json:"uid"`
	UK       int64     `json:"uk"`
	Name     string    `json:"username"`
	Avatar   string    `json:"avatar"`
	Role     int       `json:"role"`
	BasePath string    `json:"base_path"`
	VIP      int       `json:"vip"`
	VIPUntil time.Time `json:"vip_until"`
}

func (u *User) VIPLabel() string {
	if u.Role == 2 || u.VIP == 2 {
		return "管理员"
	}
	if u.VIP == 1 {
		return "VIP"
	}
	return ""
}

// UploadState holds resumable upload state.
type UploadState struct {
	Size   int64    `json:"size"`
	Mtime  int64    `json:"mtime"`
	MD5    string   `json:"md5"`
	Slice  string   `json:"slice,omitempty"`
	CRC32  uint32   `json:"crc32,omitempty"`
	Blocks []string `json:"blocks,omitempty"`
	ID     string   `json:"id,omitempty"`
	Parts  []string `json:"parts,omitempty"`
}

// Recycled represents a file in trash.
type Recycled struct {
	File
	LeftDays int
}

// Share is a link sharing files.
type Share struct {
	ID       int64   `json:"shareId"`
	Link     string  `json:"shortlink"`
	Path     string  `json:"typicalPath"`
	Category int     `json:"typicalCategory"`
	Status   int     `json:"status"`
	Expires  int64   `json:"expiredTime"` // unix time, 0 for never
	Views    int     `json:"vCnt"`
	Created  int64   `json:"ctime"`
	FsIDs    []int64 `json:"fsIds"`
}

func (s *Share) Expired(now time.Time) bool {
	return s.Status != 0 || (s.Expires > 0 && s.Expires < now.Unix())
}

// Shared is what a share link shares.
type Shared struct {
	Link    string
	Pwd     string
	ID      int64
	UK      int64
	Owner   string
	Avatar  string
	Created int64
	Expires int64
	Mine    bool
	Files   []File
}

// Offline is a background offline download task.
type Offline struct {
	ID       int64
	Name     string
	Source   string
	SavePath string
	Size     int64
	Done     int64
	Status   int // 0 done, 1 running, 2..7 failed
	Created  int64
}

func (o *Offline) StatusText() string {
	switch o.Status {
	case 0:
		return "已完成"
	case 1:
		return "下载中"
	case 3:
		return "资源不存在"
	case 4:
		return "下载超时"
	case 5:
		return "下载失败"
	case 6:
		return "空间不足"
	case 7:
		return "已取消"
	}
	return "系统错误"
}

// QRState is where a QR login is.
type QRState int

const (
	QRWaiting QRState = iota
	QRScanned
	QRConfirmed
	QRExpired
)

// QRLogin dummy structure for OpenList (can show a placeholder or notice).
type QRLogin struct {
	Image []byte
}

func NewQRLogin(ctx context.Context) (*QRLogin, error) {
	return &QRLogin{}, errors.New("OpenList 暂不支持二维码扫码，请使用账号密码或 Token 登录")
}

func (q *QRLogin) Poll(ctx context.Context) (QRState, error) {
	return QRExpired, errors.New("不支持扫码")
}

func (q *QRLogin) Finish(ctx context.Context) (Credentials, error) {
	return Credentials{}, errors.New("不支持扫码")
}

// Host extracts the host from a URL string.
func Host(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return u.Host
}

var (
	linkRe = regexp.MustCompile(`https?://[^\s]+`)
	pwdRe  = regexp.MustCompile(`(?:提取码|密码|code|pwd)[:：\s]*([a-zA-Z0-9]{4,})`)
)

// ParseShareLink parses a share link and optional password from text.
func ParseShareLink(text string) (link, pwd string, ok bool) {
	if m := linkRe.FindString(text); m != "" {
		link = m
		if pm := pwdRe.FindStringSubmatch(text); len(pm) > 1 {
			pwd = pm[1]
		}
		return link, pwd, true
	}
	return "", "", false
}
