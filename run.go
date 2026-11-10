package main

import (
	"fmt"
	"log"
	"strings"
)

func runTipOnce(cfg Config) error {
	for {
		post, err := generateTip(cfg)
		if err != nil {
			return err
		}
		if post == "" {
			return nil
		}
		next, err := showPopup(cfg, post)
		if err != nil {
			writeStatus(cfg, "error", err.Error())
			return fmt.Errorf("popup: %w", err)
		}
		if !next {
			writeStatus(cfg, "ok", "last tip shown")
			return nil
		}
		writeStatus(cfg, "running", "fetching next tip")
	}
}

func generateTip(cfg Config) (string, error) {
	writeStatus(cfg, "running", "fetching tip")
	st, err := loadState(cfg.StateFilePath)
	if err != nil {
		writeStatus(cfg, "error", err.Error())
		return "", err
	}

	slots := allSlots()
	start := st.SourceIndex % len(slots)
	const maxSlots, perSlot = 8, 1

	var post string
	var chosen TipSource
	for i := 0; i < maxSlots; i++ {
		slot := slots[(start+i)%len(slots)]
		cands, err := fetchCandidates(slot, st, perSlot)
		if err != nil {
			log.Printf("source %s: %v", slotLabel(slot), err)
			continue
		}
		for _, c := range cands {
			text, err := rewriteWithGemini(cfg, c)
			if err != nil {
				log.Printf("gemini %s: %v", c.ID, err)
				msg := err.Error()
				if strings.Contains(msg, "429") || strings.Contains(strings.ToLower(msg), "quota") {
					writeStatus(cfg, "error", "Gemini quota exceeded — retry later")
					_ = st.AdvanceSource(len(slots))
					return "", nil
				}
				if strings.Contains(msg, "503") || strings.Contains(msg, "UNAVAILABLE") {
					writeStatus(cfg, "error", "Gemini overloaded — retry later")
					_ = st.AdvanceSource(len(slots))
					return "", nil
				}
				continue
			}
			if strings.EqualFold(strings.TrimSpace(text), "SKIP") {
				_ = st.Mark(c.ID)
				continue
			}
			post = text
			chosen = c
			break
		}
		if post != "" {
			break
		}
	}
	if post == "" {
		writeStatus(cfg, "ok", "no tip this round")
		_ = st.AdvanceSource(len(slots))
		log.Println("no tip this round")
		return "", nil
	}

	if err := st.Mark(chosen.ID); err != nil {
		log.Printf("mark: %v", err)
	}
	_ = st.AdvanceSource(len(slots))
	writeStatus(cfg, "running", "showing popup")
	return post, nil
}
