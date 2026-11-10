package main

import (
	"encoding/base64"
	_ "embed"
	"fmt"
	"html"
	"os"
	"os/exec"
	"time"

	webview "github.com/webview/webview_go"
)

func showStatusPanel(cfg Config) error {
	fontB64 := base64.StdEncoding.EncodeToString(vazirWoff2)
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Tech Tips")
	w.SetSize(420, 320, webview.HintFixed)

	w.Bind("refreshStatus", func() string {
		return statusHTML(cfg)
	})
	w.Bind("runTip", func() {
		startDetached(cfg)
	})
	w.Bind("runDemo", func() {
		startDetached(cfg, "-demo")
	})
	w.Bind("closeWindow", func() {
		w.Terminate()
	})

	w.SetHtml(buildPanelHTML(fontB64, statusHTML(cfg)))
	w.Run()
	return nil
}

func startDetached(cfg Config, args ...string) {
	exe, err := os.Executable()
	if err != nil {
		writeStatus(cfg, "error", err.Error())
		return
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		writeStatus(cfg, "error", err.Error())
	}
}

func statusHTML(cfg Config) string {
	s := readStatus(cfg)
	msg := s.Message
	if msg == "" {
		msg = s.State
	}
	at := ""
	if !s.At.IsZero() {
		at = s.At.Format(time.RFC822)
	}
	return fmt.Sprintf(
		`<div class="state %s">%s</div><div class="msg">%s</div><div class="at">%s</div>`,
		html.EscapeString(s.State),
		html.EscapeString(s.State),
		html.EscapeString(msg),
		html.EscapeString(at),
	)
}

func buildPanelHTML(fontB64, statusBlock string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8"/>
<style>
@font-face {
  font-family: 'Vazirmatn';
  src: url('data:font/woff2;base64,%s') format('woff2');
  font-display: swap;
}
html, body {
  margin: 0; height: 100%%; background: #1e1e2e; color: #cdd6f4;
  font-family: 'Vazirmatn', sans-serif; direction: ltr;
}
#wrap { box-sizing: border-box; padding: 18px; height: 100%%; display: flex; flex-direction: column; gap: 14px; }
h1 { margin: 0; font-size: 18px; font-weight: 600; }
#status { background: #11111b; border-radius: 10px; padding: 14px; flex: 1; }
.state { font-size: 22px; font-weight: 700; text-transform: uppercase; margin-bottom: 8px; }
.state.idle { color: #89b4fa; }
.state.running { color: #f9e2af; }
.state.ok { color: #a6e3a1; }
.state.error { color: #f38ba8; }
.msg { line-height: 1.5; }
.at { margin-top: 10px; color: #6c7086; font-size: 12px; }
.row { display: flex; gap: 8px; }
button {
  flex: 1; border: 0; border-radius: 10px; padding: 11px 10px; cursor: pointer;
  font-family: inherit; font-size: 14px; background: #89b4fa; color: #1e1e2e;
}
button.secondary { background: #313244; color: #cdd6f4; }
button.ghost { background: transparent; color: #a6adc8; border: 1px solid #45475a; }
</style>
</head>
<body>
<div id="wrap">
  <h1>Tech Tips</h1>
  <div id="status">%s</div>
  <div class="row">
    <button type="button" id="tip">Show Tip Now</button>
    <button type="button" id="demo" class="secondary">Demo Popup</button>
  </div>
  <div class="row">
    <button type="button" id="refresh" class="ghost">Refresh Status</button>
    <button type="button" id="close" class="ghost">Close</button>
  </div>
</div>
<script>
async function paint() {
  try { document.getElementById('status').innerHTML = await refreshStatus(); } catch (e) {}
}
document.getElementById('tip').onclick = () => runTip();
document.getElementById('demo').onclick = () => runDemo();
document.getElementById('refresh').onclick = () => paint();
document.getElementById('close').onclick = () => closeWindow();
setInterval(paint, 2000);
</script>
</body>
</html>`, fontB64, statusBlock)
}
