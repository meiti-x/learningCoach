package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

	"fyne.io/systray"
)

//go:embed assets/tray-idle.png
var iconIdle []byte

//go:embed assets/tray-run.png
var iconRun []byte

//go:embed assets/tray-ok.png
var iconOK []byte

//go:embed assets/tray-err.png
var iconErr []byte

func runTray(cfg Config) {
	writeStatus(cfg, "idle", "tray started")
	systray.Run(func() { onTrayReady(cfg) }, func() {})
}

func onTrayReady(cfg Config) {
	systray.SetTitle("TechTips")
	systray.SetTooltip("Tech Tips: idle")
	systray.SetIcon(iconIdle)

	// Click tray icon opens this English menu (ItemIsMenu=true).
	mStatus := systray.AddMenuItem("Status: idle", "Open status panel")
	systray.AddSeparator()
	mNow := systray.AddMenuItem("Show Tip Now", "Fetch a tip and open popup")
	mDemo := systray.AddMenuItem("Demo Popup", "Open sample popup")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Tech Tips")

	var mu sync.Mutex
	busy := false

	startChild := func(args ...string) {
		mode := "tip"
		if len(args) > 0 {
			mode = args[0]
		}
		if mode == "tip" {
			mu.Lock()
			if busy {
				mu.Unlock()
				return
			}
			busy = true
			mu.Unlock()
			writeStatus(cfg, "running", "fetching tip")
		}
		go func() {
			if mode == "tip" {
				defer func() {
					mu.Lock()
					busy = false
					mu.Unlock()
				}()
			}
			exe, err := os.Executable()
			if err != nil {
				writeStatus(cfg, "error", err.Error())
				return
			}
			cmd := exec.Command(exe, args...)
			cmd.Env = os.Environ()
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				log.Printf("child: %v", err)
				s := readStatus(cfg)
				if s.State != "error" || s.Message == "" || s.Message == "child running" || s.Message == "fetching tip" {
					writeStatus(cfg, "error", err.Error())
				}
				return
			}
			if mode == "-demo" {
				writeStatus(cfg, "ok", "demo closed")
			}
		}()
	}

	go func() {
		t := time.NewTicker(15 * time.Minute)
		defer t.Stop()
		for range t.C {
			startChild()
		}
	}()

	go func() {
		lastLine, lastState := "", ""
		for {
			s := readStatus(cfg)
			if s.State != lastState {
				switch s.State {
				case "running":
					systray.SetIcon(iconRun)
				case "ok":
					systray.SetIcon(iconOK)
				case "error":
					systray.SetIcon(iconErr)
				default:
					systray.SetIcon(iconIdle)
				}
				lastState = s.State
			}
			msg := s.Message
			if msg == "" {
				msg = s.State
			}
			line := fmt.Sprintf("Status: %s — %s", s.State, msg)
			if !s.At.IsZero() {
				line = fmt.Sprintf("%s (%s)", line, s.At.Format("15:04"))
			}
			if line != lastLine {
				mStatus.SetTitle(line)
				systray.SetTooltip("Tech Tips: " + msg)
				lastLine = line
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// IMPORTANT: onReady must return or dbusmenu GetLayout hangs forever
	// (fyne waits on initialMenuBuilt.Done after onReady returns).
	go func() {
		for {
			select {
			case <-mStatus.ClickedCh:
				startChild("-panel")
			case <-mNow.ClickedCh:
				startChild()
			case <-mDemo.ClickedCh:
				startChild("-demo")
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}
