package openlist_test

import (
	"context"
	"testing"

	"github.com/pengstem/baidunetgo/internal/fakepan"
	"github.com/pengstem/baidunetgo/internal/openlist"
)

func TestOpenListClient(t *testing.T) {
	srv := fakepan.New()
	defer srv.Close()

	ctx := context.Background()
	cr := openlist.Credentials{
		Server:   srv.URL,
		Token:    "fake-jwt-token",
		Username: "admin",
	}
	c := openlist.OpenAt(cr, srv.URL)

	// Test User
	u, err := c.User(ctx)
	if err != nil {
		t.Fatalf("User() failed: %v", err)
	}
	if u.Name != "admin" {
		t.Errorf("expected username admin, got %s", u.Name)
	}
	if u.VIPLabel() != "管理员" {
		t.Errorf("expected VIPLabel 管理员, got %s", u.VIPLabel())
	}

	// Test Quota
	q, err := c.Quota(ctx)
	if err != nil {
		t.Fatalf("Quota() failed: %v", err)
	}
	if q.Total <= 0 {
		t.Errorf("expected positive total quota, got %d", q.Total)
	}

	// Test List
	files, err := c.List(ctx, "/", openlist.Order{})
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("expected non-empty root directory listing")
	}

	// Test Mkdir
	err = c.Mkdir(ctx, "/testdir")
	if err != nil {
		t.Fatalf("Mkdir() failed: %v", err)
	}

	// Test Rename
	err = c.Rename(ctx, "/testdir", "testdir_renamed")
	if err != nil {
		t.Fatalf("Rename() failed: %v", err)
	}

	// Test Search
	found, _, err := c.Search(ctx, "/", "README")
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(found) == 0 {
		t.Errorf("expected to find README in search, got 0 results")
	}

	// Test Delete
	err = c.Delete(ctx, []string{"/testdir_renamed"})
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}
}
