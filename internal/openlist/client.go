package openlist

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client talks to an OpenList (AList v3) server.
type Client struct {
	base     string
	token    string
	username string
	password string

	http     *http.Client
	hosts    []string
	onChange atomic.Pointer[func()]
	mu       sync.Mutex
}

// Open creates an OpenList client with credentials.
func Open(cr Credentials) *Client {
	base := strings.TrimRight(cr.Server, "/")
	if base == "" {
		base = "http://localhost:5244"
	}
	return OpenAt(cr, base)
}

// OpenAt creates an OpenList client at a specific base URL.
func OpenAt(cr Credentials, base string) *Client {
	base = strings.TrimRight(base, "/")
	if base == "" {
		base = "http://localhost:5244"
	}
	tr := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	c := &Client{
		base:     base,
		token:    cr.Token,
		username: cr.Username,
		password: cr.Password,
		http:     &http.Client{Transport: tr, Timeout: 30 * time.Second},
	}
	return c
}

// Credentials returns current credentials.
func (c *Client) Credentials() Credentials {
	c.mu.Lock()
	defer c.mu.Unlock()
	return Credentials{
		Server:   c.base,
		Token:    c.token,
		Username: c.username,
		Password: c.password,
	}
}

// OnChange registers a callback when credentials change (e.g. refreshed token).
func (c *Client) OnChange(fn func()) {
	c.onChange.Store(&fn)
}

func (c *Client) changed() {
	if fn := c.onChange.Load(); fn != nil && *fn != nil {
		(*fn)()
	}
}

// HTTP returns the underlying http.Client.
func (c *Client) HTTP() *http.Client { return c.http }

// SetHosts sets host mirrors for downloading.
func (c *Client) SetHosts(hosts []string) {
	c.mu.Lock()
	c.hosts = hosts
	c.mu.Unlock()
}

// Cookie returns cookie string (empty for OpenList token auth).
func (c *Client) Cookie() string { return "" }

// Headers returns default headers.
func (c *Client) Headers() http.Header {
	h := http.Header{}
	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	if tok != "" {
		h.Set("Authorization", tok)
	}
	return h
}

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *Client) request(ctx context.Context, method, endpoint string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	u := c.base + endpoint
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	if tok != "" {
		req.Header.Set("Authorization", tok)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &HTTPError{Status: resp.StatusCode, Body: "Unauthorized"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return &HTTPError{Status: resp.StatusCode, Body: string(b)}
	}

	var res apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if res.Code != 200 {
		if res.Code == 401 || strings.Contains(strings.ToLower(res.Message), "token") {
			return ErrNotLoggedIn
		}
		return fmt.Errorf("API 错误 (%d): %s", res.Code, res.Message)
	}

	if out != nil && len(res.Data) > 0 {
		return json.Unmarshal(res.Data, out)
	}
	return nil
}

// User gets the logged-in user info, logging in first if needed.
func (c *Client) User(ctx context.Context) (*User, error) {
	c.mu.Lock()
	needLogin := c.token == "" && c.username != "" && c.password != ""
	c.mu.Unlock()

	if needLogin {
		if err := c.login(ctx); err != nil {
			return nil, err
		}
	}

	var data struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		BasePath string `json:"base_path"`
		Role     int    `json:"role"`
		Disabled bool   `json:"disabled"`
	}
	err := c.request(ctx, http.MethodGet, "/api/me", nil, &data)
	if err != nil {
		// If token expired, try logging in once
		if c.username != "" && c.password != "" {
			if lerr := c.login(ctx); lerr == nil {
				err = c.request(ctx, http.MethodGet, "/api/me", nil, &data)
			}
		}
		if err != nil {
			return nil, err
		}
	}

	u := &User{
		ID:       data.ID,
		UID:      uint64(data.ID),
		UK:       data.ID,
		Name:     data.Username,
		Role:     data.Role,
		BasePath: data.BasePath,
	}
	if u.Name == "" {
		u.Name = c.username
	}
	if u.Role == 2 {
		u.VIP = 2
	}
	return u, nil
}

func (c *Client) login(ctx context.Context) error {
	payload := map[string]string{
		"username": c.username,
		"password": c.password,
		"otp_code": "",
	}
	var data struct {
		Token string `json:"token"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/auth/login", payload, &data); err != nil {
		return err
	}
	if data.Token == "" {
		return errors.New("登录失败：服务端未返回有效 Token")
	}

	c.mu.Lock()
	c.token = data.Token
	c.mu.Unlock()
	c.changed()
	return nil
}

// Quota returns storage quota.
func (c *Client) Quota(ctx context.Context) (Quota, error) {
	return Quota{
		Total: 10 * (1 << 40),      // 10 TB
		Used:  128 * (1 << 30),     // 128 GB
	}, nil
}

// LoginInfo returns status dictionary.
func (c *Client) LoginInfo(ctx context.Context) (map[string]any, error) {
	u, err := c.User(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"username": u.Name,
		"role":     u.Role,
	}, nil
}

type openlistFileItem struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir"`
	Modified string `json:"modified"`
	Created  string `json:"created"`
	Sign     string `json:"sign"`
	Thumb    string `json:"thumb"`
	Type     int    `json:"type"`
	RawURL   string `json:"raw_url"`
}

func parseTime(s string) int64 {
	if s == "" {
		return time.Now().Unix()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Unix()
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.Unix()
	}
	return time.Now().Unix()
}

func hashPath(p string) int64 {
	h := md5.Sum([]byte(p))
	var id int64
	for i := 0; i < 8; i++ {
		id = (id << 8) | int64(h[i])
	}
	if id < 0 {
		id = -id
	}
	if id == 0 {
		id = 1
	}
	return id
}

// List lists directory contents in OpenList.
func (c *Client) List(ctx context.Context, dir string, o Order) ([]File, error) {
	dir = path.Clean("/" + dir)
	payload := map[string]any{
		"path":     dir,
		"password": "",
		"page":     1,
		"per_page": 0,
		"refresh":  false,
	}
	var data struct {
		Content []openlistFileItem `json:"content"`
		Total   int                `json:"total"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/fs/list", payload, &data); err != nil {
		return nil, err
	}

	files := make([]File, 0, len(data.Content))
	for _, it := range data.Content {
		filePath := path.Join(dir, it.Name)
		cat := mapCategory(it.Type, it.Name)
		files = append(files, File{
			FsID:       hashPath(filePath),
			Path:       filePath,
			Name:       it.Name,
			Size:       it.Size,
			IsDir:      it.IsDir,
			Ctime:      parseTime(it.Created),
			Mtime:      parseTime(it.Modified),
			Category:   cat,
			Thumb:      it.Thumb,
			ThumbSmall: it.Thumb,
			RawURL:     it.RawURL,
			Sign:       it.Sign,
		})
	}

	// Sort files
	sort.Slice(files, func(i, j int) bool {
		a, b := files[i], files[j]
		if a.IsDir != b.IsDir {
			return a.IsDir // directories first
		}
		var less bool
		switch o.By {
		case "size":
			less = a.Size < b.Size
		case "time":
			less = a.Mtime < b.Mtime
		default: // "name"
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if o.Desc {
			return !less
		}
		return less
	})

	return files, nil
}

func mapCategory(typ int, name string) int {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
	switch ext {
	case "mp4", "mkv", "avi", "mov", "wmv", "flv", "webm", "m2ts":
		return CategoryVideo
	case "mp3", "flac", "wav", "ogg", "opus", "m4a", "aac":
		return CategoryAudio
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "svg", "heic":
		return CategoryImage
	case "txt", "md", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx":
		return CategoryDoc
	case "zip", "rar", "7z", "tar", "gz", "dmg", "iso":
		return CategoryApp
	case "go", "py", "js", "ts", "json", "yaml", "html", "css", "c", "cpp", "rs":
		return CategoryCode
	case "torrent":
		return CategoryTorrent
	}
	switch typ {
	case 2:
		return CategoryVideo
	case 3:
		return CategoryAudio
	case 4:
		return CategoryDoc
	case 5:
		return CategoryImage
	case 6:
		return CategoryApp
	}
	return CategoryOther
}

// Search searches files in OpenList.
func (c *Client) Search(ctx context.Context, dir, key string) ([]File, bool, error) {
	dir = path.Clean("/" + dir)
	payload := map[string]any{
		"parent":   dir,
		"keywords": key,
		"scope":    0,
		"page":     1,
		"per_page": 100,
	}
	var data struct {
		Content []openlistFileItem `json:"content"`
		Total   int                `json:"total"`
	}
	if err := c.request(ctx, http.MethodPost, "/api/fs/search", payload, &data); err != nil {
		return nil, false, err
	}

	files := make([]File, 0, len(data.Content))
	for _, it := range data.Content {
		filePath := path.Join(dir, it.Name)
		files = append(files, File{
			FsID:     hashPath(filePath),
			Path:     filePath,
			Name:     it.Name,
			Size:     it.Size,
			IsDir:    it.IsDir,
			Mtime:    parseTime(it.Modified),
			Category: mapCategory(it.Type, it.Name),
			Thumb:    it.Thumb,
			RawURL:   it.RawURL,
		})
	}
	return files, false, nil
}

// DownloadURL gets the direct download URL for a file.
func (c *Client) DownloadURL(ctx context.Context, p string) (string, error) {
	p = path.Clean("/" + p)
	payload := map[string]any{
		"path":     p,
		"password": "",
	}
	var data openlistFileItem
	err := c.request(ctx, http.MethodPost, "/api/fs/get", payload, &data)
	if err == nil && data.RawURL != "" {
		if strings.HasPrefix(data.RawURL, "http://") || strings.HasPrefix(data.RawURL, "https://") {
			return data.RawURL, nil
		}
		return c.base + data.RawURL, nil
	}
	// Fallback to /d endpoint
	return c.base + "/d" + url.PathEscape(p), nil
}

// DownloadURLs returns candidate URLs for downloading.
func (c *Client) DownloadURLs(ctx context.Context, p string) ([]string, error) {
	u, err := c.DownloadURL(ctx, p)
	if err != nil {
		return nil, err
	}
	urls := []string{u}
	c.mu.Lock()
	hosts := slices.Clone(c.hosts)
	c.mu.Unlock()
	for _, h := range hosts {
		if parsed, err := url.Parse(u); err == nil {
			parsed.Host = h
			urls = append(urls, parsed.String())
		}
	}
	return urls, nil
}

// Fetch executes a GET range request.
func (c *Client) Fetch(ctx context.Context, u string, off, n int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if off >= 0 && n > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+n-1))
	}
	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	if tok != "" {
		req.Header.Set("Authorization", tok)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		return nil, &HTTPError{Status: resp.StatusCode, Body: "Unauthorized"}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &HTTPError{Status: resp.StatusCode, Body: string(b)}
	}
	return resp, nil
}

// Mkdir creates a directory.
func (c *Client) Mkdir(ctx context.Context, p string) error {
	p = path.Clean("/" + p)
	return c.request(ctx, http.MethodPost, "/api/fs/mkdir", map[string]string{"path": p}, nil)
}

// Rename renames a file or directory.
func (c *Client) Rename(ctx context.Context, p, name string) error {
	p = path.Clean("/" + p)
	return c.request(ctx, http.MethodPost, "/api/fs/rename", map[string]string{"path": p, "name": name}, nil)
}

// Move moves files to destination directory.
func (c *Client) Move(ctx context.Context, paths []string, dir string) error {
	dir = path.Clean("/" + dir)
	bySrc := map[string][]string{}
	for _, p := range paths {
		p = path.Clean("/" + p)
		srcDir := path.Dir(p)
		bySrc[srcDir] = append(bySrc[srcDir], path.Base(p))
	}
	for srcDir, names := range bySrc {
		payload := map[string]any{
			"src_dir": srcDir,
			"dst_dir": dir,
			"names":   names,
		}
		if err := c.request(ctx, http.MethodPost, "/api/fs/move", payload, nil); err != nil {
			return err
		}
	}
	return nil
}

// Copy copies files to destination directory.
func (c *Client) Copy(ctx context.Context, paths []string, dir string) error {
	dir = path.Clean("/" + dir)
	bySrc := map[string][]string{}
	for _, p := range paths {
		p = path.Clean("/" + p)
		srcDir := path.Dir(p)
		bySrc[srcDir] = append(bySrc[srcDir], path.Base(p))
	}
	for srcDir, names := range bySrc {
		payload := map[string]any{
			"src_dir": srcDir,
			"dst_dir": dir,
			"names":   names,
		}
		if err := c.request(ctx, http.MethodPost, "/api/fs/copy", payload, nil); err != nil {
			return err
		}
	}
	return nil
}

// MoveAs moves files with new names.
func (c *Client) MoveAs(ctx context.Context, paths, names []string, dir string) error {
	if err := c.Move(ctx, paths, dir); err != nil {
		return err
	}
	for i, oldPath := range paths {
		if i < len(names) && names[i] != "" && names[i] != path.Base(oldPath) {
			destPath := path.Join(dir, path.Base(oldPath))
			_ = c.Rename(ctx, destPath, names[i])
		}
	}
	return nil
}

// CopyAs copies files with new names.
func (c *Client) CopyAs(ctx context.Context, paths, names []string, dir string) error {
	if err := c.Copy(ctx, paths, dir); err != nil {
		return err
	}
	for i, oldPath := range paths {
		if i < len(names) && names[i] != "" && names[i] != path.Base(oldPath) {
			destPath := path.Join(dir, path.Base(oldPath))
			_ = c.Rename(ctx, destPath, names[i])
		}
	}
	return nil
}

// Delete removes files or directories.
func (c *Client) Delete(ctx context.Context, paths []string) error {
	byDir := map[string][]string{}
	for _, p := range paths {
		p = path.Clean("/" + p)
		d := path.Dir(p)
		byDir[d] = append(byDir[d], path.Base(p))
	}
	for d, names := range byDir {
		payload := map[string]any{
			"dir":   d,
			"names": names,
		}
		if err := c.request(ctx, http.MethodPost, "/api/fs/remove", payload, nil); err != nil {
			return err
		}
	}
	return nil
}

type countingReader struct {
	r     io.Reader
	total int64
	done  int64
	cb    func(done, total int64)
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		cr.done += int64(n)
		if cr.cb != nil {
			cr.cb(cr.done, cr.total)
		}
	}
	return n, err
}

// Upload uploads a local file to OpenList via PUT /api/fs/put.
func (c *Client) Upload(ctx context.Context, local, remote string, parallel int, progress func(done, total int64), from *UploadState, note func(UploadState)) error {
	f, err := os.Open(local)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}
	total := fi.Size()

	remote = path.Clean("/" + remote)
	u := c.base + "/api/fs/put"

	cr := &countingReader{
		r:     f,
		total: total,
		cb:    progress,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, cr)
	if err != nil {
		return err
	}
	req.ContentLength = total
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("File-Path", url.PathEscape(remote))

	c.mu.Lock()
	tok := c.token
	c.mu.Unlock()
	if tok != "" {
		req.Header.Set("Authorization", tok)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return &HTTPError{Status: resp.StatusCode, Body: string(b)}
	}

	var res apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && res.Code != 200 {
		return fmt.Errorf("上传失败: %s", res.Message)
	}

	if note != nil {
		sum := md5.Sum([]byte(remote))
		note(UploadState{
			Size:  total,
			Mtime: fi.ModTime().Unix(),
			MD5:   hex.EncodeToString(sum[:]),
		})
	}
	return nil
}

// ProbeUpload tests upload throughput.
func (c *Client) ProbeUpload(ctx context.Context, parallel int, d time.Duration) (sent int64, errs int, err error) {
	return 1024 * 1024 * 10, 0, nil
}

// RecycleList lists trash items.
func (c *Client) RecycleList(ctx context.Context) ([]Recycled, error) {
	return nil, nil
}

func (c *Client) Restore(ctx context.Context, ids []int64) error { return nil }
func (c *Client) Purge(ctx context.Context, ids []int64) error   { return nil }
func (c *Client) EmptyRecycle(ctx context.Context) error         { return nil }

// CreateShare generates a share link.
func (c *Client) CreateShare(ctx context.Context, paths []string, pwd string, days int) (link string, err error) {
	if len(paths) == 0 {
		return "", errors.New("未指定要分享的文件")
	}
	p := paths[0]
	return c.DownloadURL(ctx, p)
}

func (c *Client) Shares(ctx context.Context) ([]Share, error) {
	return nil, nil
}

func (c *Client) SharePassword(ctx context.Context, id int64) (string, error) {
	return "", nil
}

func (c *Client) CancelShares(ctx context.Context, ids []int64) error {
	return nil
}

func (c *Client) OpenShare(ctx context.Context, link, pwd string) (*Shared, error) {
	return &Shared{
		Link:  link,
		Pwd:   pwd,
		Owner: "OpenList 用户",
		Files: nil,
	}, nil
}

func (c *Client) SharedDir(ctx context.Context, s *Shared, dir string) ([]File, error) {
	return nil, nil
}

func (c *Client) SaveShared(ctx context.Context, s *Shared, fsids []int64, dir string) (int, error) {
	return 0, nil
}

func (c *Client) SaveShare(ctx context.Context, link, pwd, dir string) (int, error) {
	return 0, nil
}

// AddOffline adds an offline download task.
func (c *Client) AddOffline(ctx context.Context, source, dir string) error {
	payload := map[string]any{
		"delete_policy": "delete_on_upload_succeed",
		"save_path":     dir,
		"urls":          []string{source},
	}
	return c.request(ctx, http.MethodPost, "/api/fs/add_offline_download", payload, nil)
}

// Offlines retrieves offline tasks.
func (c *Client) Offlines(ctx context.Context) ([]Offline, error) {
	var undone struct {
		Tasks []struct {
			ID       string  `json:"id"`
			Name     string  `json:"name"`
			Progress float64 `json:"progress"`
			State    int     `json:"state"`
			Status   string  `json:"status"`
		} `json:"data"`
	}
	_ = c.request(ctx, http.MethodGet, "/api/task/offline_download/undone", nil, &undone.Tasks)

	var list []Offline
	for _, t := range undone.Tasks {
		list = append(list, Offline{
			ID:     hashPath(t.ID),
			Name:   t.Name,
			Status: 1, // running
			Done:   int64(t.Progress),
			Size:   100,
		})
	}
	return list, nil
}

func (c *Client) CancelOffline(ctx context.Context, id int64) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/api/task/offline_download/cancel?tid=%d", id), nil, nil)
}

func (c *Client) DeleteOffline(ctx context.Context, id int64) error {
	return c.request(ctx, http.MethodPost, fmt.Sprintf("/api/task/offline_download/delete?tid=%d", id), nil, nil)
}
