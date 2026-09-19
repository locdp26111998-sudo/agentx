package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMessenger(t *testing.T) {
	dir := t.TempDir()
	writeAgent(t, dir, "spa-hoa.yaml", `
name: Spa Hoa
channels:
  messenger:
    - page_id: "111111111"
soul: test
messenger:
  verify_token: test-token
`)
	writeAgent(t, dir, "phong-kham-minh.yaml", `
name: Phòng khám Minh
channels:
  messenger:
    - page_id: "222222222"
soul: test
messenger:
  verify_token: test-token
`)

	rt := New(dir)

	ag, ok := rt.Resolve(ChannelMessenger, "111111111")
	if !ok || ag.ID != "spa-hoa" {
		t.Fatalf("spa-hoa: got id=%q ok=%v", ag.ID, ok)
	}

	ag, ok = rt.Resolve(ChannelMessenger, "222222222")
	if !ok || ag.ID != "phong-kham-minh" {
		t.Fatalf("phong-kham-minh: got id=%q ok=%v", ag.ID, ok)
	}

	_, ok = rt.Resolve(ChannelMessenger, "999999999")
	if ok {
		t.Fatal("unknown page_id should not match")
	}
}

func TestResolveZalo(t *testing.T) {
	dir := t.TempDir()
	writeAgent(t, dir, "agent-zalo.yaml", `
name: Zalo Agent
channels:
  zalo:
    - oa_id: "444444444"
soul: test
`)

	rt := New(dir)
	ag, ok := rt.Resolve(ChannelZalo, "444444444")
	if !ok || ag.ID != "agent-zalo" {
		t.Fatalf("zalo: got id=%q ok=%v", ag.ID, ok)
	}
}

func writeAgent(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
