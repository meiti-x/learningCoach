package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	log.SetFlags(0)
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	switch arg {
	case "-tray":
		cfg, err := loadConfig()
		if err != nil {
			log.Fatalf("config: %v", err)
		}
		runTray(cfg)
		return
	case "-panel":
		cfg := loadConfigSoft()
		if err := showStatusPanel(cfg); err != nil {
			log.Fatalf("panel: %v", err)
		}
		return
	case "-demo":
		cfg := loadConfigSoft()
		writeStatus(cfg, "running", "demo popup")
		sample := "وقتی توی Go از یه اسلایس بزرگ یه بخش کوچیک رو جدا می‌کنیم (Re-slicing)، دیتای جدیدی توی حافظه کپی نمیشه.\n\n```go\n// Bad: pins the entire underlying array\nfunc getHeader(data []byte) []byte {\n    return data[:4]\n}\n```\n\nتوی سرویس‌های ۲۴ ساعته خیلی سریع رم رو می‌بلعه\n\n🔗 [Go Wiki: SliceTricks](https://github.com/golang/go/wiki/SliceTricks)\n\n#backend\n#golang"
		for {
			next, err := showPopup(cfg, sample)
			if err != nil {
				writeStatus(cfg, "error", err.Error())
				log.Fatalf("popup: %v", err)
			}
			if !next {
				writeStatus(cfg, "ok", "demo closed")
				return
			}
			// demo "next" tries a real tip; falls back to sample on soft empty
			if post, err := generateTip(cfg); err == nil && post != "" {
				sample = post
				continue
			}
			writeStatus(cfg, "ok", "demo closed")
			return
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := runTipOnce(cfg); err != nil {
		os.Exit(1)
	}
}

func slotLabel(s sourceSlot) string {
	switch s.kind {
	case "se":
		return "se/" + s.tag
	case "gh":
		return s.repo + "/" + s.path
	case "rss":
		return s.path
	default:
		return fmt.Sprintf("%v", s)
	}
}
