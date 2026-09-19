// Kiểm tra mọi configs/agents/*.yaml load được (cùng ticket.Load như router).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"agentx/internal/router"
	"agentx/internal/ticket"
)

func main() {
	dir := "configs/agents"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "đọc thư mục %s: %v\n", dir, err)
		os.Exit(1)
	}

	var n int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		tkt, err := ticket.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, err)
			os.Exit(1)
		}
		n++
		pageIDs := tkt.MessengerChannels()
		fmt.Printf("OK   %s  name=%q  engine=%q  messenger_pages=%d\n",
			e.Name(), tkt.Name(), tkt.Engine(), len(pageIDs))
	}

	if n == 0 {
		fmt.Fprintf(os.Stderr, "không có file *.yaml trong %s\n", dir)
		os.Exit(1)
	}

	rt := router.New(dir)
	if _, ok := rt.Resolve(router.ChannelMessenger, "679362898589478"); !ok {
		fmt.Fprintf(os.Stderr, "cảnh báo: router không resolve page_id demo 679362898589478 (thiếu demo.yaml?)\n")
		os.Exit(1)
	}

	fmt.Printf("\nTổng %d file YAML — router scan OK.\n", n)
}
