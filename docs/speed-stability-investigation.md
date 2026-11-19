# Speed and stability release check

On 2026-11-19, I ran the speed test and `/stability/?load=all` in Chromium on this build and on v5.1.2. The numbers
match, the upload no longer counts rejected bytes, and the stability test still takes about a minute. I found no
measurement bug that would block the release.

## Live comparison with v5.1.2

Both builds ran on localhost against the same test server, one test at a time, over the VPN with CAKE on. The
baseline was v5.1.2 (`70b7178`). No run lost a packet under load.

### Speed test

The page, the saved results, and the latency recalculated from the raw pings all agreed. Three runs on the current
build turned on the multi-server flag (`mpot`) against the same server.

| Build | Run | Flag | Down / up (Mbps) | Idle ping avg / jitter (ms) | Loaded p95 down / up (ms) | Upload measurement (s) |
| --- | ---: | --- | --- | --- | --- | ---: |
| v5.1.2 | 1 | Off | 87.31 / 74.12 | 47.74 / 0.67 | 51.7 / 50.5 | 5.20 |
| Current | 1 | Off | 86.25 / 82.38 | 46.91 / 0.30 | 53.4 / 51.6 | 8.80 |
| v5.1.2 | 2 | Off | 86.02 / 74.41 | 51.20 / 2.38 | 51.3 / 60.8 | 5.20 |
| Current | 2 | Off | 86.52 / 79.02 | 47.71 / 0.96 | 57.8 / 54.6 | 9.00 |
| v5.1.2 | 3 | Off | 85.69 / 82.35 | 47.70 / 1.04 | 145.8 / 66.2 | 5.20 |
| Current | 3 | Off | 85.78 / 75.04 | 60.07 / 26.75 | 57.2 / 96.2 | 8.40 |
| Current | 4 | On | 74.61 / 80.97 | 46.14 / 0.70 | 356.0 / 48.5 | 8.80 |
| Current | 5 | Off | 86.30 / 79.80 | 47.13 / 0.80 | 54.2 / 50.9 | 9.00 |
| Current | 6 | On | 87.14 / 79.98 | 47.21 / 0.50 | 49.7 / 53.0 | 8.80 |
| Current | 7 | On | 86.67 / 81.58 | 47.20 / 1.07 | 50.0 / 50.3 | 8.60 |
| Current | 8 | Off | 79.77 / 82.49 | 53.24 / 0.35 | 52.5 / 48.6 | 8.60 |

The 356 ms download p95 in current run 4 was real. The raw pings show it, and the two repeats with the flag on came
back near 50 ms. The download took about 5.2 seconds on both builds. The upload times leave out the 3.2 second
warm-up.

### Stability with idle, download, and upload

The latency, jitter, p50, p95, and packet loss recalculated from the raw pings matched what the page showed, both
overall and per phase. In the current build's third run, the upload phase had one 297.5 ms ping, with a p95 of
50.5 ms.

| Build | Run | Down / up (Mbps) | p95 idle / down / up (ms) | Pings | Browser elapsed (s) |
| --- | ---: | --- | --- | ---: | ---: |
| v5.1.2 | 1 | 84.35 / 78.01 | 49.9 / 49.3 / 49.5 | 273 | 60.16 |
| Current | 1 | 78.51 / 82.81 | 59.1 / 50.1 / 49.9 | 274 | 61.16 |
| v5.1.2 | 2 | 84.21 / 75.80 | 49.8 / 51.0 / 49.3 | 273 | 61.16 |
| Current | 2 | 84.06 / 81.19 | 50.0 / 50.4 / 49.0 | 273 | 61.16 |
| v5.1.2 | 3 | 62.14 / 77.10 | 50.6 / 79.3 / 52.6 | 273 | 61.16 |
| Current | 3 | 83.71 / 83.83 | 51.7 / 50.5 / 50.5 | 272 | 61.15 |

## Why the speed test uploads for longer

With `time_auto` on, v5.1.2 shortened the upload by the bytes sent so far. The current worker shortens it only by
bytes the server has confirmed with a successful reply. The speed shown still counts the bytes in flight. The
stability test has `time_auto` off and fixed phase lengths, so this doesn't change it.

On the same simulated connection, the measurement took 5.2 seconds on v5.1.2 and 8.0 seconds with the new worker.
Feeding the v5.1.2 byte count into the new timer brought it back to 5.2 seconds, with both builds reporting about
84.7 Mbps. With `time_auto` off, both measured for 15.2 seconds.

The current worker also waits for the server's reply before it sends the next request on that stream. With 1 MiB
requests and a server that waits 200 ms to reply, upload on localhost dropped from 240.19 to 110.10 Mbps. The real
test sends 20 MB at a time, or 4 MB on Chrome mobile, and the server replies at once. In 10 upload-only pairs in
Firefox against the same server, both builds had a median upload of 84 Mbps.

## Rejected uploads

Before the fix, the worker dropped the bytes of a rejected request only once the whole error reply had arrived. A
server that sent HTTP 413 and kept the reply open could leave 83.23 Mbps on the screen. The worker now drops those
bytes as soon as the error status arrives.

| Loopback browser check | Final reading (Mbps) |
| --- | ---: |
| HTTP 413 with immediate response body | 0.00 |
| HTTP 413 with delayed response body | 0.00 |
| HTTP 200 with delayed response body | 84.62 |

Nine more upload checks passed, among them a rejected stream next to a working one, network errors, retries, and
the user pressing Stop. `go test -count=1 ./...` passed.
