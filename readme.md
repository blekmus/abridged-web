# abridged-web

Abridged Anime but now on the Web!

I suppose this the natural evolution of a half-decade long project to build the ideal interface for watching abridged anime.


## what is it

It started off as a [TUI](https://github.com/blekmus/abridged-cli-old) built on top of [ink](https://github.com/vadimdemedes/ink) back when the library was still not quite matured. Unlike now, with it powering CC and Gemini and all. 

Frustrated with its limitations I moved on to [blessed](https://github.com/chjj/blessed), another javascript based TUI library. I was sticking to JS because I was interested in how far I could push its TUI ecosystem before I would call it quits. Blessed was both a boon and a challenge at the same time. It was more flexible than ink, but not flexible enough. Getting it to do the weirdly complex rendering patterns I wanted meant forking the package and patching it directly. That became [abridged-cli](https://github.com/blekmus/abridged-cli).

There was also a brief detour into [electron-abridged](https://github.com/blekmus/electron-abridged), an attempt at a desktop UI to make metadata retrieval and manual assignment less painful. That didn't go anywhere tho. I did end up with some cool illustrations thanks to it.

This was when I daily drove linux, then after a few years I moved on to a macbook and got introduced to [Raycast](https://www.raycast.com/) and its plugin ecosystem. After building a few quality of life plugins and contributing on their plugin marketplace, I decided to port the abridged-cli into it. That became [raycast-abridged](https://github.com/blekmus/raycast-abridged), and for a while that was good enough. 

The idea of a web version floated around for a while without going anywhere until coding agents entered the picture. Pointing `codex` at the existing metadata reading logic and asking it to port everything to Go produced a near-complete implementation on the first prompt. It made a few uncalled for assumptions, but the result was sure as hell close enough. That was enough of a push to make me revisit the web version again. One that would allow not only me, but other people to watch the collection too. And here we are.

## How It Works

The app is a single Go binary that embeds a Vite/Preact frontend. At startup, the backend reads a JSON metadata file from the archive root and holds the catalogue in memory. It exposes a small set of JSON endpoints for browsing and per-series watch pages, and serves video and thumbnail files directly from the configured archive path via HTTP range requests.

The frontend is a single-page app compiled at build time and embedded into the binary via Go's embed package. In production, the backend serves it statically. In development, the frontend runs separately through Vite's dev server and proxies API calls to the Go process.

The whole thing ships as a single Docker image with no external dependencies beyond the mounted archive. Media files are expected to already be transcoded and structured using the [abridged-convert](https://github.com/blekmus/abridged-convert) CLI. `abridged-covert` is a companion CLI that acts as a conversion layer; because most of my abridged archive lives in MKV files, which needed transcoding before the browser could touch them.


## Development

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `ARCHIVE_ROOT` | `../archive` | Path to the archive directory |
| `APP_ENV` | `development` | Set to `production` for production logging/runtime mode |

Needs an archive directory accessible at runtime with `ARCHIVE_ROOT/.abridged-video-metadata.json` at its root, and read access to the videos, thumbnails, and metadata for the process user.
 
When running via Docker, the container runs as UID/GID `10001` — the mounted archive directory and metadata file both need to be readable by that user.
 
### Production
 
The Compose file pulls the published image from GHCR. Update the archive path in `docker-compose.yml`:
 
```yaml
volumes:
  - /srv/abridged/archive:/archive:ro
```
 
```sh
docker compose up -d
```
 
Runs on `http://localhost:8080`.
 
### Development
 
Frontend and backend run separately in two terminals:
 
```sh
bun run dev
```
 
```sh
go run ./main.go
```

### Building
 
Running the docker build locally; 
```sh
docker build -t abridged-web .
```
 
```sh
docker run --rm -p 8080:8080 \
  -e ARCHIVE_ROOT=/archive \
  -v /path/to/archive:/archive:ro \
  abridged-web
```
 
Running the go server manually;
 
```sh
bun run build
APP_ENV=production go build -o abridged ./main.go
```
 
```sh
PORT=3000 ARCHIVE_ROOT=/path/to/archive APP_ENV=production ./abridged
```
