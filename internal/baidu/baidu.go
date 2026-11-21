package baidu

import (
	"context"
	"errors"

	"github.com/pengstem/baidunetgo/internal/openlist"
)

type Client = openlist.Client
type File = openlist.File
type Order = openlist.Order
type Quota = openlist.Quota
type User = openlist.User
type UploadState = openlist.UploadState
type Recycled = openlist.Recycled
type Offline = openlist.Offline
type Share = openlist.Share
type Shared = openlist.Shared
type Credentials = openlist.Credentials
type HTTPError = openlist.HTTPError
type QRState = openlist.QRState
type QRLogin = openlist.QRLogin

const (
	CategoryOther   = openlist.CategoryOther
	CategoryVideo   = openlist.CategoryVideo
	CategoryAudio   = openlist.CategoryAudio
	CategoryImage   = openlist.CategoryImage
	CategoryDoc     = openlist.CategoryDoc
	CategoryApp     = openlist.CategoryApp
	CategoryCode    = openlist.CategoryCode
	CategoryTorrent = openlist.CategoryTorrent

	QRWaiting   = openlist.QRWaiting
	QRScanned   = openlist.QRScanned
	QRConfirmed = openlist.QRConfirmed
	QRExpired   = openlist.QRExpired

	UserAgent = "OpenListGo/0.1.0"
)

var (
	ErrNotLoggedIn   = openlist.ErrNotLoggedIn
	ErrExists        = openlist.ErrExists
	ErrNoSpace       = openlist.ErrNoSpace
	ErrSharePassword = openlist.ErrSharePassword
	ErrShareGone     = errors.New("分享已失效")
)

func Open(cr Credentials) *Client { return openlist.Open(cr) }
func OpenAt(cr Credentials, base string) *Client { return openlist.OpenAt(cr, base) }
func ParseCredentials(s string) (Credentials, error) { return openlist.ParseCredentials(s) }
func ParseShareLink(text string) (link, pwd string, ok bool) { return openlist.ParseShareLink(text) }
func Expired(err error) bool { return openlist.Expired(err) }
func Host(link string) string { return openlist.Host(link) }
func NewQRLogin(ctx context.Context) (*QRLogin, error) { return openlist.NewQRLogin(ctx) }
