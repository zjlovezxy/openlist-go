package openlist

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// Credentials holds authentication information for OpenList.
type Credentials struct {
	Server   string `json:"server"`
	Token    string `json:"token"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// Legacy compatibility fields if needed
	BDUSS string `json:"bduss,omitempty"`
	UID   uint64 `json:"uid,omitempty"`
}

// ParseCredentials parses credentials from a string, supporting JSON,
// URL format (e.g. "http://user:pass@localhost:5244"), or token.
func ParseCredentials(text string) (Credentials, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Credentials{}, errors.New("凭据不能为空")
	}

	// Try JSON first
	if strings.HasPrefix(text, "{") {
		var cr Credentials
		if err := json.Unmarshal([]byte(text), &cr); err == nil && cr.Server != "" {
			return cr, nil
		}
	}

	// Try URL
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		u, err := url.Parse(text)
		if err == nil {
			cr := Credentials{}
			if u.User != nil {
				cr.Username = u.User.Username()
				cr.Password, _ = u.User.Password()
				u.User = nil
			}
			cr.Server = strings.TrimRight(u.String(), "/")
			return cr, nil
		}
	}

	// If it contains a token or server
	return Credentials{
		Server: "http://localhost:5244",
		Token:  text,
	}, nil
}
