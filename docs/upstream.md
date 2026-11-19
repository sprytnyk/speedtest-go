# Upstream changes

This page tracks which changes we took from [librespeed/speedtest-go](https://github.com/librespeed/speedtest-go).

## Reviewed through `654df71`

These are the six upstream commits after the shared base, `59cff12`.

| Upstream commit | What we did |
| --- | --- |
| [`367cd2e` · Results view and telemetry](https://github.com/librespeed/speedtest-go/commit/367cd2e9cff67074902d04e8b80f2008040537a9) | Limited telemetry requests to 1 MiB, stopped logging expected client disconnects on writes, and added stats pagination and filters. |
| [`9ee94bb` · Run without a server list](https://github.com/librespeed/speedtest-go/commit/9ee94bb361610fbb1e5b79496d2e74b48ab9be04) | Already covered by the fork's local-server fallback. Telemetry remains opt-in. |
| [`2411619` · Exclude failed uploads](https://github.com/librespeed/speedtest-go/commit/24116199bbca2ccd661bb61d187f042f0d8ccf49) | Adapted the worker to exclude failed requests, including HTTP errors, from the amount uploaded. Pings and upload bytes still use the same warm-up boundary. |
| [`f6dc876` · MySQL connection format](https://github.com/librespeed/speedtest-go/commit/f6dc876f1896e92f481616e6338200ffee0626a6) | Already documented in the README and settings. |
| [`84e771d` · Manual GitHub releases](https://github.com/librespeed/speedtest-go/commit/84e771dc7775d55428ba6859b56bd2af69cc2642) | Skipped. The fork already builds release archives from tags and on demand. |
| [`654df71` · Stability page](https://github.com/librespeed/speedtest-go/commit/654df712c6f7ad6902cb2fe86df9c287b7409fdc) | Took the shutdown fix and kept the fork's final sample. The upstream worker matches the fork's October 10 version. |

