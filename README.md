<div align="center">

<img src="https://play.rajkumaar.co.in/apple-touch-icon.png" alt="Play Store API" width="84" height="84" />

# Play Store API

**Every app on Play. One GET request.**

Real-time Google Play metadata — versions, installs, ratings, screenshots — as clean JSON.
No key. No signup. Open source.

[**Try it live → play.rajkumaar.co.in**](https://play.rajkumaar.co.in)

[![build image](https://github.com/rajkumaar23/playstore-api/actions/workflows/build.yaml/badge.svg)](https://github.com/rajkumaar23/playstore-api/actions/workflows/build.yaml)
[![nightly test](https://github.com/rajkumaar23/playstore-api/actions/workflows/test.yaml/badge.svg)](https://github.com/rajkumaar23/playstore-api/actions/workflows/test.yaml)
[![License: MIT](https://img.shields.io/badge/license-MIT-2ee584)](LICENSE)
![Go](https://img.shields.io/badge/go-1.26-00add8?logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-ghcr.io-2496ed?logo=docker&logoColor=white)

</div>

---

## Show me

```bash
curl "https://play.rajkumaar.co.in/json?id=com.dd.doordash"
```

```json
{
  "packageID": "com.dd.doordash",
  "name": "DoorDash: Food, Grocery, More",
  "version": "15.299.7",
  "downloads": "50,000,000+",
  "downloadsExact": 91787576,
  "lastUpdated": "Nov 3, 2026",
  "launchDate": "Mar 26, 2015",
  "developer": "DoorDash",
  "category": "Food & Drink",
  "rating": "4.7",
  "noOfUsersRated": "6,303,190",
  "logo": "https://play-lh.googleusercontent.com/…",
  "banner": "https://play-lh.googleusercontent.com/…",
  "screenshots": ["https://play-lh.googleusercontent.com/…", "…"],
  "website": "https://www.doordash.com/",
  "privacyPolicy": "https://www.doordash.com/privacy/",
  "supportEmail": "support@doordash.com",
  "latestUpdateMessage": "…",
  "description": "Delivery anywhere you are. …"
}
```

All **19 fields**, in one object, scraped on demand and cached. The [playground](https://play.rajkumaar.co.in) runs these requests live in your browser — no curl required.

## Endpoints

| Method | Path | Returns |
|---|---|---|
| `GET` | `/json?id=<pkg>[&gl=CC]` | the full object above |
| `GET` | `/<field>?id=<pkg>[&gl=CC]` | one field, [shields.io](https://shields.io) badge format |

`id` accepts `[A-Za-z0-9._-]{1,200}`. `gl` is an optional ISO country code — ratings are region-specific, so pass one (e.g. `&gl=IN`) when the default returns `0`.

Any response field works as `<field>` — `version`, `downloads`, `downloadsExact`, `rating`, `noOfUsersRated`, `lastUpdated`, `launchDate`, `developer`, `category`, and the rest of the 19.

## Badges in your README

```markdown
![Downloads](https://img.shields.io/endpoint?color=success&url=https://play.rajkumaar.co.in/downloads?id=com.dd.doordash)
![Version](https://img.shields.io/endpoint?color=blue&url=https://play.rajkumaar.co.in/version?id=com.dd.doordash)
![Rating](https://img.shields.io/endpoint?color=orange&url=https://play.rajkumaar.co.in/rating?id=com.dd.doordash)
```

![Downloads](https://img.shields.io/endpoint?color=success&url=https://play.rajkumaar.co.in/downloads?id=com.dd.doordash)
![Version](https://img.shields.io/endpoint?color=blue&url=https://play.rajkumaar.co.in/version?id=com.dd.doordash)
![Rating](https://img.shields.io/endpoint?color=orange&url=https://play.rajkumaar.co.in/rating?id=com.dd.doordash)

These badges are being served by this very API as you read this. Add `&style=for-the-badge&color=…` for styling — see [shields.io](https://shields.io).

## Self-host it

```bash
git clone https://github.com/rajkumaar23/playstore-api && cd playstore-api
docker compose up -d          # API on :5003, Redis, Prometheus on :5004
```

Or run the image against your own Redis:

```bash
docker run -d -p 5003:5003 \
  -e REDIS_ADDRESS=your-redis:6379 \
  ghcr.io/rajkumaar23/playstore-api:main
```

| Env | Default | Meaning |
|---|---|---|
| `SERVER_PORT` | `5003` | HTTP port |
| `METRICS_PORT` | `5004` | Prometheus `/metrics` |
| `REDIS_ADDRESS` | — | `host:port` of your Redis |
| `CACHE_TTL` | `24h` | how long responses are cached |
| `DEFAULT_GL` | `US` | fallback country code |
| `GIN_MODE` | — | `release` in production |

## Develop

```bash
go run ./cmd/playstore-api     # needs REDIS_ADDRESS
go test ./...                  # unit + security regression suites
```

## Notes

- Unofficial — this reads public Play Store pages; be gentle, the cache does the rest.
- `rating` / `noOfUsersRated` can be `0` for the default region; use `&gl=CC`.
- New package IDs trigger a live scrape on first request; subsequent hits are cached.

## License

[MIT](LICENSE) · Built by [Rajkumar](https://rajkumaar.co.in)
