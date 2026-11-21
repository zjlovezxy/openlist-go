package migrate

import (
	"github.com/pengstem/baidunetgo/internal/baidu"
)

// Account is an account another client is signed in with.
type Account struct {
	From  string // the client, for the user
	Name  string // the account's name, if the client keeps it
	Creds baidu.Credentials
}

// Label names the account for the user.
func (a Account) Label() string {
	if a.Name == "" {
		return a.From
	}
	return a.From + "（" + a.Name + "）"
}

// Find returns accounts found.
func Find() []Account {
	return nil
}
