package fakepan

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

func (s *Server) openlistRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth/login", s.olLogin)
	mux.HandleFunc("/api/me", s.olMe)
	mux.HandleFunc("/api/public/settings", s.olSettings)
	mux.HandleFunc("/api/fs/list", s.olList)
	mux.HandleFunc("/api/fs/get", s.olGet)
	mux.HandleFunc("/api/fs/search", s.olSearch)
	mux.HandleFunc("/api/fs/mkdir", s.olMkdir)
	mux.HandleFunc("/api/fs/rename", s.olRename)
	mux.HandleFunc("/api/fs/move", s.olMove)
	mux.HandleFunc("/api/fs/copy", s.olCopy)
	mux.HandleFunc("/api/fs/remove", s.olRemove)
	mux.HandleFunc("/api/fs/put", s.olPut)
	mux.HandleFunc("/api/fs/add_offline_download", s.olAddOffline)
	mux.HandleFunc("/api/task/offline_download/undone", s.olOfflineUndone)
	mux.HandleFunc("/api/task/offline_download/done", s.olOfflineDone)
	mux.HandleFunc("/api/task/offline_download/cancel", s.olOfflineCancel)
	mux.HandleFunc("/api/task/offline_download/delete", s.olOfflineDelete)
	mux.HandleFunc("/d/", s.olDownload)
}

func (s *Server) olLogin(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"token": "openlist-fake-jwt-token-demo",
		},
	})
}

func (s *Server) olMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"id":        1,
			"username":  "admin",
			"base_path": "/",
			"role":      2,
			"disabled":  false,
		},
	})
}

func (s *Server) olSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"version":    "v3.39.0",
			"site_title": "OpenList 网盘",
		},
	})
}

func (s *Server) olList(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	dir := path.Clean("/" + req.Path)

	s.mu.Lock()
	defer s.mu.Unlock()

	n := s.nodes[dir]
	if n == nil || !n.IsDir {
		writeJSON(w, map[string]any{
			"code":    404,
			"message": "目录不存在: " + dir,
		})
		return
	}

	var content []map[string]any
	for _, child := range s.children(dir) {
		mtimeStr := time.Unix(child.Mtime, 0).Format(time.RFC3339)
		thumb := ""
		if child.Category == 3 || child.Category == 1 {
			thumb = s.URL + "/thumb?id=" + strconv.FormatInt(child.FsID, 10)
		}
		rawURL := s.URL + "/d" + child.Path
		content = append(content, map[string]any{
			"name":     path.Base(child.Path),
			"size":     child.Size,
			"is_dir":   child.IsDir,
			"modified": mtimeStr,
			"created":  mtimeStr,
			"thumb":    thumb,
			"type":     child.Category,
			"raw_url":  rawURL,
		})
	}

	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"content": content,
			"total":   len(content),
			"write":   true,
		},
	})
}

func (s *Server) olGet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := path.Clean("/" + req.Path)

	s.mu.Lock()
	defer s.mu.Unlock()

	n := s.nodes[targetPath]
	if n == nil {
		writeJSON(w, map[string]any{
			"code":    404,
			"message": "文件不存在",
		})
		return
	}

	mtimeStr := time.Unix(n.Mtime, 0).Format(time.RFC3339)
	thumb := ""
	if n.Category == 3 || n.Category == 1 {
		thumb = s.URL + "/thumb?id=" + strconv.FormatInt(n.FsID, 10)
	}
	rawURL := s.URL + "/d" + n.Path

	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"name":     path.Base(n.Path),
			"size":     n.Size,
			"is_dir":   n.IsDir,
			"modified": mtimeStr,
			"created":  mtimeStr,
			"thumb":    thumb,
			"type":     n.Category,
			"raw_url":  rawURL,
			"provider": "Local",
		},
	})
}

func (s *Server) olDownload(w http.ResponseWriter, r *http.Request) {
	targetPath, err := url.PathUnescape(strings.TrimPrefix(r.URL.Path, "/d"))
	if err != nil {
		targetPath = strings.TrimPrefix(r.URL.Path, "/d")
	}
	targetPath = path.Clean("/" + targetPath)

	s.mu.Lock()
	n := s.nodes[targetPath]
	s.mu.Unlock()

	if n == nil || n.IsDir {
		http.NotFound(w, r)
		return
	}

	var content io.ReadSeeker
	if n.Data != nil {
		content = bytes.NewReader(n.Data)
	} else {
		content = io.NewSectionReader(zeros{}, 0, n.Size)
	}
	http.ServeContent(w, r, path.Base(n.Path), time.Unix(n.Mtime, 0), content)
}

func (s *Server) olSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Parent   string `json:"parent"`
		Keywords string `json:"keywords"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	key := strings.ToLower(req.Keywords)
	parent := path.Clean("/" + req.Parent)

	s.mu.Lock()
	defer s.mu.Unlock()

	var content []map[string]any
	for p, n := range s.nodes {
		if p != "/" && strings.HasPrefix(p, strings.TrimSuffix(parent, "/")+"/") && strings.Contains(strings.ToLower(path.Base(p)), key) {
			mtimeStr := time.Unix(n.Mtime, 0).Format(time.RFC3339)
			content = append(content, map[string]any{
				"name":     path.Base(n.Path),
				"size":     n.Size,
				"is_dir":   n.IsDir,
				"modified": mtimeStr,
				"type":     n.Category,
				"raw_url":  s.URL + "/d" + n.Path,
			})
		}
	}
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"content": content,
			"total":   len(content),
		},
	})
}

func (s *Server) olMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.Add(Node{Path: req.Path, IsDir: true})
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.Lock()
	defer s.mu.Unlock()
	oldPath := path.Clean("/" + req.Path)
	if n := s.nodes[oldPath]; n != nil {
		newPath := path.Join(path.Dir(oldPath), req.Name)
		delete(s.nodes, oldPath)
		n.Path = newPath
		s.nodes[newPath] = n
	}
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olMove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SrcDir string   `json:"src_dir"`
		DstDir string   `json:"dst_dir"`
		Names  []string `json:"names"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range req.Names {
		oldP := path.Join(req.SrcDir, name)
		newP := path.Join(req.DstDir, name)
		if n := s.nodes[oldP]; n != nil {
			delete(s.nodes, oldP)
			n.Path = newP
			s.nodes[newP] = n
		}
	}
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olCopy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SrcDir string   `json:"src_dir"`
		DstDir string   `json:"dst_dir"`
		Names  []string `json:"names"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range req.Names {
		oldP := path.Join(req.SrcDir, name)
		newP := path.Join(req.DstDir, name)
		if n := s.nodes[oldP]; n != nil {
			cp := *n
			cp.Path = newP
			s.nextID++
			cp.FsID = s.nextID
			s.nodes[newP] = &cp
		}
	}
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olRemove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir   string   `json:"dir"`
		Names []string `json:"names"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, name := range req.Names {
		targetP := path.Join(req.Dir, name)
		delete(s.nodes, targetP)
	}
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olPut(w http.ResponseWriter, r *http.Request) {
	fp := r.Header.Get("File-Path")
	p, err := url.PathUnescape(fp)
	if err != nil {
		p = fp
	}
	p = path.Clean("/" + p)
	data, _ := io.ReadAll(r.Body)
	s.Add(Node{
		Path:  p,
		Data:  data,
		Size:  int64(len(data)),
		Mtime: time.Now().Unix(),
	})
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olAddOffline(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": map[string]any{
			"task": map[string]any{
				"id":       "task_offline_001",
				"name":     "离线下载任务",
				"progress": 0,
			},
		},
	})
}

func (s *Server) olOfflineUndone(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data": []map[string]any{
			{
				"id":       "task_offline_001",
				"name":     "sample-download.iso",
				"progress": 68.5,
				"state":    1,
				"status":   "running",
			},
		},
	})
}

func (s *Server) olOfflineDone(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"code":    200,
		"message": "success",
		"data":    []any{},
	})
}

func (s *Server) olOfflineCancel(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}

func (s *Server) olOfflineDelete(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{"code": 200, "message": "success", "data": nil})
}
