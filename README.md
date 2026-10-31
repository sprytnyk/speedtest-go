<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".logo/logo2.png">
  <img alt="LibreSpeed" src=".logo/logo3.png" width="360">
</picture>

# LibreSpeed

No Flash, No Java, No WebSocket, No Bullshit.

A self-hosted speed test in a single Go binary. The browser measures download, upload, ping and jitter with
XMLHttpRequest and Web Workers, and the web UI is built into the binary, so there is nothing else to install.

This is a fork of [librespeed/speedtest-go](https://github.com/librespeed/speedtest-go).

![Speed test](docs/screenshots/speedtest.png)

| Stability test | Statistics |
| --- | --- |
| ![Stability test](docs/screenshots/stability.png) | ![Statistics, latest tests](docs/screenshots/stats-list.png) |

<details>
<summary>A single test on the statistics page</summary>

![Statistics, one test](docs/screenshots/stats-detail.png)

</details>

## Features

* Download and upload speed, ping and jitter, with a live speed chart
* Latency and jitter while downloading and uploading
* Jitter as the average change between consecutive pings, shown with the smallest and largest change
* IP address, ISP and country of the client
* Connection stability test at `/stability.html`, with latency over time, jitter, failed requests, an alert threshold
  and CSV export. It runs idle or under load, where it pings while idle, then downloading, then uploading, and shows
  the latency of each phase
* Optional result storage, with a shareable result image, a JSON API and a password-protected statistics page
* Multiple test servers to choose from
* English and Ukrainian, picked from the browser's language or a switch on every page
* Layouts for desktop and phone
* Optional IP redaction for stored results
* Supports the [PROXY protocol](https://www.haproxy.org/download/2.3/doc/proxy-protocol.txt)
* Answers the PHP backend's endpoints too, with their `.php` suffixes and `/backend/` prefix

The pages work in current versions of Firefox, Chrome, Edge and Safari, on desktop and mobile.

## Installation

### Prebuilt binaries

1. Download the archive for your platform from the [releases](https://github.com/sprytnyk/speedtest-go/releases/)
   page and unpack it.
2. Edit `settings.toml` (see [Configuration](#configuration)).
3. Run `./speedtest-backend`. It reads `settings.toml` from the current directory, or the file given with
   `-c /path/to/settings.toml`. `./speedtest-backend -version` prints the version.

The test is then served at `http://<host>:8989/`.

### Docker

Every release publishes images for `linux/amd64`, `linux/arm64`, `linux/arm/v7` and `linux/386` to the GitHub
Container Registry, tagged with the version, such as `v3.1.0`, and with `latest`:

```
docker run -d -p 8989:8989 ghcr.io/sprytnyk/speedtest-go:latest
```

The image contains the repository's `settings.toml`, which keeps results in memory. To change settings, mount your
own file over `/app/settings.toml`, or use environment variables:

```
docker run -d -p 8989:8989 -e SPEEDTEST_DATABASE_TYPE=sqlite -e SPEEDTEST_DATABASE_FILE=/data/speedtest.db \
  -v speedtest-data:/data ghcr.io/sprytnyk/speedtest-go:latest
```

### systemd

[`systemd/`](systemd/) has a hardened unit with socket activation and setup steps. Two things to know:

* `MemoryDenyWriteExecute=true` works with the release binaries, but not with binaries packed with UPX, which unpack
  themselves into executable memory.
* With `DynamicUser=true`, a SQLite or BoltDB file needs a writable place that survives restarts. Add
  `StateDirectory=speedtest` to the unit and set `database_file="/var/lib/speedtest/speedtest.db"`.

### From source

Building needs Go 1.27 or later:

```
git clone https://github.com/sprytnyk/speedtest-go
cd speedtest-go
go build -ldflags "-w -s -X main.version=$(git describe --tags)" -trimpath -o speedtest-backend .
```

The binary embeds the web UI from `web/assets`, so you only copy `speedtest-backend` and `settings.toml` to the
server.

`scripts/build-release.sh` builds the release archives and their `SHA256SUMS` for every platform into `dist/`. The
*Release builds* workflow runs it for every version tag, and from the Actions tab for any branch, and keeps the
archives as a downloadable artifact of the run. The builds are reproducible, so the same commit gives the same
checksums on any machine with the same Go toolchain.

## Configuration

The server reads its settings from `settings.toml`. The file in this repository lists every key with a comment. An
environment variable named `SPEEDTEST_` plus the key in capitals, such as `SPEEDTEST_LISTEN_PORT=8080`, overrides
the key in the file.

| Key | Meaning |
| --- | --- |
| `bind_address`, `listen_port` | Where to listen. An empty address listens on all interfaces; the port defaults to 8989. |
| `url_base` | Path prefix when the test is served below the site root, such as `/librespeed`. |
| `proxyprotocol_port` | Extra port that expects the PROXY protocol, `0` to disable. |
| `assets_path` | Directory with your own copy of the web UI. When it is empty or missing, the server uses the built-in UI. |
| `ip_lookup_url` | ISP lookup API, see [ISP lookup and privacy](#isp-lookup-and-privacy). |
| `database_type` | Where results are stored: `none`, `memory`, `bolt`, `sqlite`, `mysql`, `postgresql` or `mssql`. |
| `database_file` | Database file for `bolt` and `sqlite`. |
| `database_hostname`, `database_port`, `database_name`, `database_username`, `database_password` | Connection settings for MySQL, PostgreSQL and MSSQL. Only MSSQL uses `database_port`. |
| `statistics_password` | Password of the statistics page. The page stays locked while it is `PASSWORD` or empty. |
| `redact_ip_addresses` | Store results without the client's IP address and hostname. |
| `enable_tls`, `tls_cert_file`, `tls_key_file` | Serve HTTPS directly. The server always uses HTTP/1.1. |

### Multiple servers

To let users choose between test servers, put a `server-list.json` into your `assets_path` directory:

```json
[
  {
    "name": "Kyiv, UA",
    "server": "https://kyiv.example.com/",
    "dlURL": "garbage",
    "ulURL": "empty",
    "pingURL": "empty",
    "getIpURL": "getIP",
    "sponsorName": "Example ISP",
    "sponsorURL": "https://example.com/"
  }
]
```

The page pings every server and selects the closest. Without the file, the test runs against the server that
serves the page.

## Getting accurate results

The numbers in this section come from a 100 Mbps wired line about 45 ms from the server, tested in Brave and
Chromium. The article [Self-hosted speed tests need HTTP/1.1 and
BBR](https://sprytnyk.dev/posts/self-hosted-speed-tests-http1-bbr/) has the full measurements.

### Serve the test over HTTP/1.1

Over HTTP/2, the browser sends every download stream through one TCP connection. On that line one connection reached
30 to 45 Mbps, so the page showed 36 to 42 Mbps. Six HTTP/1.1 connections reached 94 Mbps. The latency under
load was wrong as well. The page's pings shared the connection and waited behind download data, so they showed 220
to 350 ms instead of 44 ms.

Over HTTP/1.1 the page downloads with 4 connections and leaves one of the browser's 6 per server for the pings. The
server only speaks HTTP/1.1, so this matters behind a reverse proxy. In Caddy, set the protocols in the global
options. The setting applies to every site on the same port:

```
{
	servers {
		protocols h1
	}
}
```

HTTP/3 doesn't use one TCP connection either, but the results depended on the browser. Firefox downloaded at 91 to
105 Mbps, but its pings under load showed 1,050 to 1,300 ms. Brave reached 87 to 95 Mbps when forced onto HTTP/3,
and 47 to 51 Mbps when it chose HTTP/3 itself.

### Use BBR on the server

Linux uses cubic congestion control by default. When a cubic connection loses packets, it slows down, and at 45 ms
it takes seconds to speed up again. `time_auto` ends a fast test after 6 to 9 seconds, so the test ended before the
connections caught up. With BBR, the page's average rose from 62 to 84 Mbps. One connection reached 57 to 90 Mbps,
up from 30 to 36 Mbps with cubic. To switch, run as root:

```
echo tcp_bbr > /etc/modules-load.d/bbr.conf
echo 'net.ipv4.tcp_congestion_control = bbr' > /etc/sysctl.d/60-bbr.conf
modprobe tcp_bbr
sysctl -p /etc/sysctl.d/60-bbr.conf
```

The change applies to new connections. With BBR the latency under load rose from about 60 ms to between 120 and
150 ms, because the test now fills the line and the line's queue fills with it.

### Raise the TCP buffers for fast, distant users

A connection can only send as much data as its buffer holds before the other side confirms it arrived. Its speed
is limited to about the buffer size divided by the ping. By default Linux lets each connection use up to 4 MB for
sending and 6 MB for receiving, and keeps part of that for itself. 100 Mbps at 45 ms needs about 0.6 MB, but 1 Gbps
at 100 ms needs about 12.5 MB. On [#872](https://github.com/librespeed/speedtest/issues/872#issuecomment-6362608909),
a user fixed slow results on a very fast connection by raising the buffers along with BBR. The test uses several
connections, so 16 MB each is enough for 1 Gbps at well over 100 ms. To raise them, run as root:

```
cat > /etc/sysctl.d/60-tcp-buffers.conf <<'EOF'
net.ipv4.tcp_rmem = 4096 131072 16777216
net.ipv4.tcp_wmem = 4096 16384 16777216
EOF
sysctl -p /etc/sysctl.d/60-tcp-buffers.conf
```

The `net.ipv4` settings cover IPv6 too.

### Keep the grace times

The worker ignores the first 1.5 seconds of the download and 3 seconds of the upload, while connections speed up.
With `time_dlGraceTime` and `time_ulGraceTime` set to `0` in `settings.json`, the test counts that start. The page's
average was 49 Mbps with both at `0`, and 62 Mbps with the defaults.

## Storing results

The server stores a result when the page sends it at the end of a test. That gives every test an ID, the **Share**
button with a result image, and the statistics page. To turn it on:

1. Choose a database with `database_type`. `sqlite` is the simplest. It only needs `database_file` and creates its
   tables itself.
2. Set `statistics_password` to open the statistics page at `/stats/`.
3. Copy `web/assets` to a directory of your own, set `assets_path` to it, and add a `settings.json` there:

    ```json
    { "telemetry_level": "full" }
    ```

    `basic` stores the results only. `full` also stores the test log, which the statistics page shows.
4. Put your contact address into the privacy policy in `index.html` in place of `krlsv@pm.me`. The page shows the
   policy before a user starts a test.

You can then open a stored test at:

* `/results/?id=<id>`: the result image, in Ukrainian with `&lang=uk`
* `/results/json?id=<id>`: the result as JSON
* `/stats/`: the latest 100 tests, and every detail of a single test. You can delete tests there one by one, a
  selection at a time, or all at once

### Databases

* **SQLite** and **BoltDB** need nothing but the file, and SQLite runs in WAL mode.
* For **PostgreSQL**, **MySQL** and **MSSQL**, create the database and import the schema from
  `database/{postgresql,mysql,mssql}`, for example `psql speedtest < database/postgresql/telemetry_postgresql.sql`.
* **MySQL** takes the Go driver's address format in `database_hostname`, not a bare hostname:

    ```toml
    database_hostname="tcp(db.example.com:3306)"        # TCP, port is required
    database_hostname="unix(/run/mysqld/mysqld.sock)"   # Unix socket
    database_hostname=""                                # TCP to 127.0.0.1:3306
    ```

    A bare value such as `localhost` fails at startup with `default addr for network 'localhost' unknown`.

* The server looks up results by `uuid`, so the schemas index that column. A database made from an older schema
  needs the index added by hand, except SQLite, which adds it on startup:

    ```sql
    -- PostgreSQL
    CREATE INDEX speedtest_users_uuid_idx ON speedtest_users (uuid);
    -- MySQL
    CREATE INDEX speedtest_users_uuid_idx ON speedtest_users (uuid(26));
    -- MSSQL (nvarchar(max) columns cannot be indexed)
    ALTER TABLE speedtest_users ALTER COLUMN uuid nvarchar(64) NULL;
    CREATE NONCLUSTERED INDEX IX_speedtest_users_uuid ON speedtest_users (uuid);
    ```

## ISP lookup and privacy

The client's ISP and country come from the API set in `ip_lookup_url`. The `settings.toml` in this repository, and
so the Docker image, uses [ip.sprytnyk.dev](https://ip.sprytnyk.dev/openapi.json). Without the key, there are no
lookups. The server sends only the client's IP address, with no API key, cookies or other identifiers. It never
sends private or special addresses, such as CGNAT and IPv6 ULA ranges. It takes the client IP from the connection,
or from the `X-Forwarded-For`, `X-Real-IP`, `Client-IP` and `CF-Connecting-IPv6` headers when a proxy sets them.

* To use another API, point `ip_lookup_url` at it. The server replaces `{ip}` in the URL with the client's IP, and
  the response must be JSON with `ip`, `city`, `country`, `iso_code` and `org` fields.
* To turn lookups off for everyone, set `ip_lookup_url=""` or leave the key out. To turn them off in the web UI
  only, set `getIp_ispInfo: false` in its settings.

## Differences from the PHP backend

* Test IDs are random [ULIDs](https://github.com/ulid/spec) instead of auto-increment numbers, so nobody can guess
  them. `enable_id_obfuscation` isn't needed and isn't supported.
* The server looks up the ISP with the API described above instead of ipinfo.io or an offline GeoIP database, so it
  needs no API key or `.mmdb` file.
* The server doesn't calculate the distance between client and server, so it ignores `server_lat`, `server_lng`,
  `ipinfo_api_key` and `geoip_database_file`.

## License

Copyright (C) 2016-2020 Federico Dossena
Copyright (C) 2020 Maddie Zhan
Copyright (C) 2026 Vladyslav Krylasov

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/lgpl>.
