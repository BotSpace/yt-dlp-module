# youtube-module

Botmother tashqi moduli — YouTube'dan video/audio yuklab olish.

`youtube.Download` node YouTube havoladan to'g'ridan-to'g'ri media URL va
metadata (sarlavha, davomiylik, muqova) oladi. Ichida [yt-dlp](https://github.com/yt-dlp/yt-dlp)
ishlaydi. Fayl hostlanmaydi — progressive (yagona oqim) URL qaytariladi va
Telegram uni URL orqali bevosita yuboradi.

## Lokal test

```bash
go run .              # yt-dlp lokal o'rnatilgan bo'lsin: brew install yt-dlp

curl http://localhost:8100/health
curl -X POST http://localhost:8100/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"node.execute","id":1,
       "params":{"type":"youtube.Download",
                 "data":{"url":"https://youtu.be/dQw4w9WgXcQ","format":"360"}}}'
```

## Docker

```bash
docker build -t youtube-module .
docker run -p 8100:8100 youtube-module
```

`yt-dlp` + `ffmpeg` image ichiga (alpine community repo) o'rnatiladi.

## Node

### `youtube.Download` (action)

| Field | Tavsif |
|---|---|
| **url** | YouTube havola (`{{message.text}}` yoki literal) |
| **format** | best / 720p / 360p / audio (m4a) |

**Chiqish state'lari:** `yt_url`, `yt_title`, `yt_duration`, `yt_thumbnail`, `yt_error`
**Chiqish edge'lari:** `success` / `error`

## Misol flow

```
Xabar kelganda (trigger)
  → YouTube yuklab olish (url: {{message.text}}, format: 360)
  → Video yuborish (url: {{yt_url}}, caption: {{yt_title}})
```

## Cheklovlar

- Faqat **progressive** (video+audio yagona oqim) URL — host/merge yo'q.
- Telegram URL orqali **~20MB** gacha yuboradi; kattaroq fayl uchun modul faylni
  o'zi yuklab olib serve qilishi kerak (keyingi bosqich).
- YouTube ba'zi videolarga login/yosh-cheklov qo'yadi — bunda `error` chiqadi.
