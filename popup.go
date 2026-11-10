package main

import (
	"encoding/base64"
	_ "embed"
	"fmt"
	"html"
	"regexp"
	"strings"

	webview "github.com/webview/webview_go"
)

//go:embed assets/Vazirmatn-Regular.woff2
var vazirWoff2 []byte

func showPopup(cfg Config, post string) (wantNext bool, err error) {
	fontB64 := base64.StdEncoding.EncodeToString(vazirWoff2)
	w := webview.New(false)
	defer w.Destroy()
	w.SetTitle("Tech Tips")
	w.SetSize(620, 580, webview.HintFixed)

	current := post
	w.Bind("publishToTelegram", func() string {
		if err := publishTelegram(cfg, current); err != nil {
			return err.Error()
		}
		return "ok"
	})
	w.Bind("nextTip", func() {
		wantNext = true
		w.Terminate()
	})
	w.Bind("closeWindow", func() {
		w.Terminate()
	})

	w.SetHtml(buildPopupHTML(fontB64, post))
	w.Run()
	return wantNext, nil
}

func buildPopupHTML(fontB64, post string) string {
	bodyHTML := markdownToHTML(post)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
<meta charset="utf-8"/>
<link href="https://cdn.jsdelivr.net/gh/rastikerdar/vazirmatn@v33.003/Vazirmatn-font-face.css" rel="stylesheet" type="text/css"/>
<link href="https://cdn.jsdelivr.net/npm/jetbrains-mono@1.0.6/css/jetbrains-mono.min.css" rel="stylesheet"/>
<link href="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.9.0/build/styles/github-dark.min.css" rel="stylesheet"/>
<script src="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.9.0/build/highlight.min.js"></script>
<style>
@font-face {
  font-family: 'Vazirmatn';
  src: url('data:font/woff2;base64,%s') format('woff2');
  font-weight: 400;
  font-style: normal;
  font-display: swap;
}
html, body {
  margin: 0; padding: 0; height: 100%%;
  background: #1e1e2e; color: #cdd6f4;
  font-family: 'Vazirmatn', sans-serif !important;
  direction: rtl;
}
#wrap {
  display: flex; flex-direction: column; height: 100%%;
  box-sizing: border-box; padding: 16px 16px 12px;
}
#content {
  flex: 1; overflow: auto; line-height: 1.8; font-size: 15px;
  padding-right: 4px; scroll-behavior: smooth;
}
#content a { color: #89b4fa; }
#content p { margin: 0 0 0.9em; }
pre, code {
  direction: ltr !important;
  text-align: left !important;
  font-family: 'JetBrains Mono', 'Fira Code', monospace !important;
}
pre {
  background: #11111b; border-radius: 8px; padding: 12px;
  overflow-x: auto; margin: 0.9em 0;
}
code { font-size: 13px; }
:not(pre) > code {
  background: #313244; padding: 0.1em 0.35em; border-radius: 4px;
}
#bar { padding-top: 10px; display: flex; gap: 8px; }
button {
  flex: 1; border: 0; border-radius: 10px; padding: 12px 14px;
  background: #89b4fa; color: #1e1e2e; cursor: pointer; font-size: 15px;
  font-family: 'Vazirmatn', sans-serif !important;
  direction: rtl;
  transition: background .15s ease;
}
button.secondary { background: #313244; color: #cdd6f4; }
button:disabled { cursor: default; }
button.loading { background: #6c7086; color: #cdd6f4; opacity: 1; }
button.ok { background: #a6e3a1; color: #1e1e2e; }
button.err { background: #f38ba8; color: #11111b; }
button.loading::before {
  content: '';
  display: inline-block;
  width: 12px; height: 12px; margin-left: 8px;
  border: 2px solid #cdd6f4; border-top-color: transparent;
  border-radius: 50%%; vertical-align: -2px;
  animation: spin .7s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
</head>
<body>
<div id="wrap">
  <div id="content">%s</div>
  <div id="bar">
    <button id="btnNext" class="secondary" type="button">نکته بعدی</button>
    <button id="btn" type="button">انتشار در تلگرام</button>
  </div>
</div>
<script>
document.querySelectorAll('pre code').forEach((el) => { try { hljs.highlightElement(el); } catch (e) {} });
const btn = document.getElementById('btn');
const btnNext = document.getElementById('btnNext');
btnNext.addEventListener('click', () => {
  if (btnNext.disabled) return;
  btnNext.disabled = true;
  btn.disabled = true;
  btnNext.className = 'secondary loading';
  btnNext.textContent = 'در حال ساخت…';
  nextTip();
});
btn.addEventListener('click', async () => {
  if (btn.disabled) return;
  btn.disabled = true;
  btnNext.disabled = true;
  btn.className = 'loading';
  btn.textContent = 'در حال ارسال…';
  try {
    const res = await publishToTelegram();
    if (res !== 'ok') { throw new Error(res || 'publish failed'); }
    btn.className = 'ok';
    btn.textContent = 'منتشر شد ✅';
    setTimeout(() => { closeWindow(); }, 1500);
  } catch (e) {
    btn.disabled = false;
    btnNext.disabled = false;
    btn.className = 'err';
    btn.textContent = 'خطا — دوباره تلاش کن';
  }
});
</script>
</body>
</html>`, fontB64, bodyHTML)
}

var (
	fenceRe = regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)\\n(.*?)```")
	linkRe  = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func markdownToHTML(md string) string {
	type slot struct{ html string }
	var slots []slot
	md = fenceRe.ReplaceAllStringFunc(md, func(m string) string {
		sub := fenceRe.FindStringSubmatch(m)
		lang := html.EscapeString(sub[1])
		code := html.EscapeString(strings.TrimRight(sub[2], "\n"))
		idx := len(slots)
		slots = append(slots, slot{html: fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, lang, code)})
		return fmt.Sprintf("@@FENCE%d@@", idx)
	})

	var paras []string
	for _, block := range strings.Split(md, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if strings.HasPrefix(block, "@@FENCE") && strings.HasSuffix(block, "@@") {
			paras = append(paras, slots[atoiFence(block)].html)
			continue
		}
		paras = append(paras, "<p>"+inlineFormat(block)+"</p>")
	}
	return strings.Join(paras, "\n")
}

func atoiFence(token string) int {
	var n int
	fmt.Sscanf(token, "@@FENCE%d@@", &n)
	return n
}

func inlineFormat(raw string) string {
	// Protect inline code, escape rest, restore links/code.
	type piece struct{ kind, val, extra string }
	var parts []piece
	rest := raw
	for {
		ci := strings.Index(rest, "`")
		li := strings.Index(rest, "[")
		if ci < 0 && li < 0 {
			if rest != "" {
				parts = append(parts, piece{kind: "t", val: rest})
			}
			break
		}
		useCode := ci >= 0 && (li < 0 || ci < li)
		if useCode {
			if ci > 0 {
				parts = append(parts, piece{kind: "t", val: rest[:ci]})
			}
			end := strings.Index(rest[ci+1:], "`")
			if end < 0 {
				parts = append(parts, piece{kind: "t", val: rest[ci:]})
				break
			}
			parts = append(parts, piece{kind: "c", val: rest[ci+1 : ci+1+end]})
			rest = rest[ci+1+end+1:]
			continue
		}
		if li > 0 {
			parts = append(parts, piece{kind: "t", val: rest[:li]})
		}
		m := linkRe.FindStringSubmatchIndex(rest[li:])
		if m == nil {
			parts = append(parts, piece{kind: "t", val: rest[li : li+1]})
			rest = rest[li+1:]
			continue
		}
		parts = append(parts, piece{
			kind:  "a",
			val:   rest[li+m[2] : li+m[3]],
			extra: rest[li+m[4] : li+m[5]],
		})
		rest = rest[li+m[1]:]
	}
	var b strings.Builder
	for _, p := range parts {
		switch p.kind {
		case "c":
			b.WriteString("<code>")
			b.WriteString(html.EscapeString(p.val))
			b.WriteString("</code>")
		case "a":
			b.WriteString(`<a href="`)
			b.WriteString(html.EscapeString(p.extra))
			b.WriteString(`" target="_blank" rel="noopener">`)
			b.WriteString(html.EscapeString(p.val))
			b.WriteString("</a>")
		default:
			b.WriteString(html.EscapeString(p.val))
		}
	}
	return strings.ReplaceAll(b.String(), "\n", "<br/>")
}
