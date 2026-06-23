# youtube-module

Botmother tashqi moduli — YouTube'dan video/audio yuklab olish.

`youtube.Download` node YouTube havoladan video/audio'ni [yt-dlp](https://github.com/yt-dlp/yt-dlp)
bilan **yuklab oladi**, platformaga saqlaydi (`c.UploadFile`) va fayl **UUID**'sini
qaytaradi. Send* node shu UUID bilan yuboradi.

> Nega URL emas? Google to'g'ridan media URL'ni Telegram serveridan bloklaydi —
> shuning uchun faylning o'zi yuklanadi.

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

**Chiqish state'lari:** `yt_file` (UUID), `yt_title`, `yt_duration`, `yt_thumbnail`, `yt_error`
**Chiqish edge'lari:** `success` / `error`

## Misol flow

```
Xabar kelganda (trigger)
  → YouTube yuklab olish (url: {{message.text}}, format: 360)
  → Video yuborish (fayl: {{yt_file}}, caption: {{yt_title}})
```

## Cheklovlar

- Video amalda **~360p** progressive bilan cheklanadi (yagona oqim).
- Butun fayl xotiraga o'qiladi — juda katta fayllar uchun streaming kerak bo'lishi mumkin.
- YouTube ba'zi videolarga login/yosh-cheklov qo'yadi — bunda `error` chiqadi.
- **Bot-tekshiruvi:** YouTube datacenter IP'larni "Sign in to confirm you're not
  a bot" bilan bloklaydi. Modul `player_client=android,web` bilan buni odatda
  chetlab o'tadi. Agar baribir bloklasa — cookie kerak: `yt-dlp --cookies` yoki
  `--cookies-from-browser`. Buni qo'shish keyingi bosqich (credential orqali).
