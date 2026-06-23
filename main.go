// youtube-module — Botmother tashqi modul: YouTube'dan video/audio yuklab olish.
//
// Node turi:
//   - youtube.Download — action: YouTube havoladan to'g'ridan-to'g'ri media URL +
//     metadata (sarlavha, davomiylik, thumbnail) chiqaradi. Natija state'ga
//     yoziladi; flow uni native "video/audio yuborish" node bilan yetkazadi.
//
// Ishlash printsipi: yt-dlp (de-facto standart) chaqiriladi. Fayl hostlanmaydi —
// progressive (yagona oqim) googlevideo URL'i qaytariladi, Telegram uni URL
// orqali bevosita yuboradi.
//
// ponytail: faqat progressive (video+audio bitta oqim) URL — host/merge yo'q.
// Telegram URL orqali ~20MB gacha yuboradi; kattaroq fayl uchun yuklab+host
// kerak — bu keyingi bosqich (README §Cheklovlar).
package main

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	botmodule "github.com/BotSpace/botmodule-go"
)

const moduleID = "youtube"

// yt-dlp format selektorlari — select field'dagi value → yt-dlp -f argumenti.
// Hammasi progressive (yagona oqim) yoki audio-only: URL orqali to'g'ridan yuboriladi.
//
// MUHIM: YouTube 360p'dan yuqorida progressive oqim bermaydi (yuqorisi DASH —
// alohida video/audio). Shuning uchun har bir selektor balandlik bo'yicha
// progressive'ni qidiradi, topmasa eng yaxshi mavjud progressive'ga TUSHADI
// (oxirgi fallback `b[vcodec!=none][acodec!=none]` ~doim itag 18 = 360p mp4).
// Ya'ni so'ralgan sifat "best-effort" — error o'rniga eng yaqin progressive.
var formatMap = map[string]string{
	"best":  "b[vcodec!=none][acodec!=none]",
	"720":   "b[height<=720][vcodec!=none][acodec!=none]/b[vcodec!=none][acodec!=none]",
	"360":   "b[height<=360][vcodec!=none][acodec!=none]/b[vcodec!=none][acodec!=none]",
	"audio": "ba[ext=m4a]/ba/b[acodec!=none]", // android client sof audio bermasa progressive (audioli) ga tushadi
}

func main() {
	m := botmodule.New(moduleID, "YouTube")
	m.Version = "0.1.0"
	m.Docs = docs

	m.AddNode(botmodule.Node{
		Type:        "youtube.Download",
		Title:       "YouTube yuklab olish",
		Description: "YouTube havoladan media URL va metadata oladi (yt-dlp)",
		Category:    "integrations",
		Icon:        "globe",
		Color:       "integration-sky",
		Width:       200,
		Content: []botmodule.Field{
			{
				Type:        "text",
				Key:         "url",
				Label:       "YouTube havola",
				Placeholder: "{{message.text}}",
				HelpText:    "youtube.com/watch?v=... yoki youtu.be/...",
			},
			{
				Type:  "select",
				Key:   "format",
				Label: "Format",
				Options: []botmodule.SelectOption{
					{Value: "best", Label: "Video — eng yaxshi (mp4)"},
					{Value: "720", Label: "Video — 720p gacha"},
					{Value: "360", Label: "Video — 360p gacha (kichik)"},
					{Value: "audio", Label: "Faqat audio (m4a)"},
				},
			},
		},
		Defaults: map[string]any{
			"url":    "{{message.text}}",
			"format": "best",
		},
		Outputs: []botmodule.Output{
			{Name: "success", Label: "Topildi", Variant: "success"},
			{Name: "error", Label: "Xato", Variant: "danger"},
		},
		ProducesState: []string{"yt_url", "yt_title", "yt_duration", "yt_thumbnail", "yt_error"},
		Execute:       executeDownload,
	})

	m.Serve(":8100")
}

func executeDownload(c *botmodule.ExecuteCtx) botmodule.Result {
	url := strings.TrimSpace(c.String("url"))
	if !isYouTubeURL(url) {
		return errResult("YouTube havola noto'g'ri yoki bo'sh")
	}

	fmtKey := c.String("format")
	ytFormat, ok := formatMap[fmtKey]
	if !ok {
		ytFormat = formatMap["best"]
	}

	// Bitta yt-dlp chaqiruvi: tanlangan format uchun metadata + to'g'ridan URL.
	// --print har biri alohida qatorda chiqadi (tartib saqlanadi).
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "yt-dlp",
		"--no-warnings", "--no-playlist", "--quiet",
		// YouTube datacenter IP'larni "Sign in to confirm you're not a bot" bilan
		// bloklaydi; android client cookie'siz ham odatda o'tadi (web fallback).
		"--extractor-args", "youtube:player_client=android,web",
		"-f", ytFormat,
		"--print", "%(title)s",
		"--print", "%(duration)s",
		"--print", "%(thumbnail)s",
		"--print", "%(urls)s",
		url,
	)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return errResult("yt-dlp vaqt tugadi (90s)")
		}
		msg := strings.TrimSpace(string(stderr(err)))
		if msg == "" {
			msg = err.Error()
		}
		return errResult("yt-dlp xatosi: " + truncate(msg, 300))
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) < 4 {
		return errResult("yt-dlp javobi tushunarsiz (format topilmadi)")
	}
	title := lines[0]
	durationSec, _ := strconv.Atoi(strings.TrimSpace(lines[1]))
	thumbnail := lines[2]
	mediaURL := strings.TrimSpace(lines[3])

	// %(urls)s bir nechta qator bo'lishi mumkin (video+audio alohida bo'lsa).
	// Progressive format tanlaganmiz — birinchi (yagona) URL yetarli.
	if i := strings.IndexByte(mediaURL, '\n'); i >= 0 {
		mediaURL = mediaURL[:i]
	}
	if mediaURL == "" {
		return errResult("media URL topilmadi")
	}

	return botmodule.Result{
		ContextUpdates: map[string]any{
			"yt_url":       mediaURL,
			"yt_title":     title,
			"yt_duration":  durationSec,
			"yt_thumbnail": thumbnail,
			"yt_error":     "",
		},
		ExitOutput: "success",
	}
}

// isYouTubeURL — minimal trust-boundary tekshiruvi (yt-dlp'ga ixtiyoriy URL bermaslik).
func isYouTubeURL(s string) bool {
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return false
	}
	return strings.Contains(s, "youtube.com/") || strings.Contains(s, "youtu.be/")
}

// stderr — exec.ExitError ichidagi stderr matnini oladi.
func stderr(err error) []byte {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.Stderr
	}
	return nil
}

func errResult(msg string) botmodule.Result {
	return botmodule.Result{
		ContextUpdates: map[string]any{
			"yt_url":   "",
			"yt_error": msg,
		},
		ExitOutput: "error",
		Error:      msg, // UI error list + alert'da ko'rsatiladi
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

const docs = `# YouTube

YouTube havoladan video yoki audio'ning to'g'ridan-to'g'ri media URL'ini va
metadata'sini ([yt-dlp](https://github.com/yt-dlp/yt-dlp) orqali) oladi.

## Node turi

### ` + "`youtube.Download`" + ` (action)

| Field | Tavsif |
|---|---|
| **url** | YouTube havola (` + "`{{message.text}}`" + ` yoki literal) |
| **format** | Video (best/720p/360p) yoki faqat audio (m4a) |

**Chiqish state'lari:**

- ` + "`yt_url`" + ` — to'g'ridan media URL (Telegram URL orqali yuboradi)
- ` + "`yt_title`" + ` — video sarlavhasi
- ` + "`yt_duration`" + ` — davomiylik (soniya)
- ` + "`yt_thumbnail`" + ` — muqova rasm URL'i
- ` + "`yt_error`" + ` — xato matni (muvaffaqiyatda bo'sh)

**Chiqish edge'lari:** ` + "`success`" + ` / ` + "`error`" + `

## Misol flow

` + "```" + `
Xabar kelganda (trigger)
  → YouTube yuklab olish (url: {{message.text}}, format: 360)
  → Video yuborish (url: {{yt_url}}, caption: {{yt_title}})
` + "```" + `

## Cheklovlar

Faqat progressive (yagona oqim) URL qaytariladi — fayl hostlanmaydi. Telegram
URL orqali ~20MB gacha yuboradi; kattaroq fayllar uchun modul faylni yuklab
olib o'zi serve qilishi kerak (keyingi bosqich).
`
