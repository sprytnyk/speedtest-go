/**
 * Speed test page: runs the LibreSpeed test from speedtest.js and renders the
 * results, matching the layout of the stability page.
 */

// States the UI can be in
const INITIALIZING = 0;
const READY = 1;
const RUNNING = 2;
const FINISHED = 3;

// Keep some global state here
const testState = {
  state: INITIALIZING,
  speedtest: null,
  servers: [],
  testData: null,
  telemetryEnabled: false,
  noServer: false, // no server answered the reachability check
  aborted: false, // the last test was stopped before it finished
  startedAt: 0, // performance.now() when the test started
  samples: [], // { t: seconds since start, v: Mbps, kind: "dl" | "ul" }
  loaded: { download: null, upload: null }, // latency while each transfer ran, from the stability worker
  loadedDone: { download: false, upload: false }, // the pinger for each transfer finished, with or without a reply
  readyWorkers: {}, // a pinger for each transfer, made before it starts so its script is already loaded
  latencyWorker: null, // pings the server while the download or upload runs
  latencyKind: null, // "download" or "upload" while latencyWorker runs
  customStreams: false, // settings.json sets the download streams itself
  hoverX: null, // pointer position over the chart, in CSS pixels
  dirty: true, // re-render on the next animation frame
};

// Chart colors: blue and amber stay distinct for common color vision
// deficiencies, and download and upload never overlap in time anyway
const CHART_COLORS = {
  download: "#3b82f6",
  downloadFill: "rgba(59, 130, 246, 0.22)",
  downloadFillEnd: "rgba(59, 130, 246, 0)",
  upload: "#f59e0b",
  uploadFill: "rgba(245, 158, 11, 0.2)",
  uploadFillEnd: "rgba(245, 158, 11, 0)",
  grid: "#334155",
  text: "#94a3b8",
  reference: "#94a3b8",
  surface: "#1e293b",
};
const MIN_CHART_SECONDS = 35; // roughly one full run with the default settings
const CHART_PADDING = { right: 8, top: 16, bottom: 26 };

// Server used when no server list is available, matching the stability page.
// URLs are relative so they also work behind a URL prefix.
const LOCAL_SERVER = {
  name: "Local server",
  server: "./",
  dlURL: "backend/garbage.php",
  ulURL: "backend/empty.php",
  pingURL: "backend/empty.php",
  getIpURL: "backend/getIP.php",
};

const $ = (id) => document.getElementById(id);

// Bootstrap the application when the DOM is ready
window.addEventListener("DOMContentLoaded", () => {
  // the translations were just applied, so redo the select's copy of the chosen option
  refreshServerSelect();
  I18N.onChange(() => {
    testState.dirty = true;
    refreshServerSelect();
    try {
      showSponsor(testState.speedtest.getSelectedServer());
    } catch (e) {
      // no server selected yet
    }
  });
  createSpeedtest();
  hookUpControls();
  startRenderingLoop();
  applySettingsJSON();
  applyServerListJSON();
});

/**
 * Create a new Speedtest and hook it into the global state
 */
function createSpeedtest() {
  testState.speedtest = new Speedtest();
  // store the latency while idle, downloading and uploading with the results
  testState.speedtest.setParameter("telemetry_latency", true);
  testState.speedtest.onupdate = (data) => {
    testState.testData = data;
    recordSample(data);
    trackLoadedLatency(data);
    testState.dirty = true;
  };
  testState.speedtest.onend = (aborted) => {
    discardReadyWorkers();
    testState.state = aborted ? READY : FINISHED;
    testState.aborted = aborted;
    testState.dirty = true;
  };
}

/**
 * Make all the buttons, the server list and the chart respond to input
 */
function hookUpControls() {
  $("start-button").addEventListener("click", startButtonClickHandler);
  $("choose-privacy").addEventListener("click", () => $("privacy").showModal());
  $("share-results").addEventListener("click", () => $("share").showModal());
  $("copy-link").addEventListener("click", copyLinkButtonClickHandler);
  document.querySelectorAll(".close-dialog").forEach((element) => {
    element.addEventListener("click", () => element.closest("dialog").close());
  });
  // close a dialog by clicking its backdrop
  document.querySelectorAll("dialog").forEach((dialog) => {
    dialog.addEventListener("click", (e) => {
      if (e.target === dialog) dialog.close();
    });
  });

  $("server").addEventListener("change", (e) => {
    const server = testState.servers[e.target.value];
    if (server) selectServer(server);
  });

  const chart = $("speedChart");
  function trackPointer(e) {
    const x = e.clientX - chart.getBoundingClientRect().left;
    testState.hoverX = x >= 0 && x <= chart.clientWidth - CHART_PADDING.right ? x : null;
    testState.dirty = true;
  }
  chart.addEventListener("pointermove", trackPointer);
  chart.addEventListener("pointerdown", trackPointer);
  chart.addEventListener("pointerleave", (e) => {
    // on touch screens, keep the tooltip after the finger lifts
    if (e.pointerType !== "mouse") return;
    testState.hoverX = null;
    testState.dirty = true;
  });
  document.addEventListener("pointerdown", (e) => {
    if (testState.hoverX !== null && e.target !== chart) {
      testState.hoverX = null;
      testState.dirty = true;
    }
  });

  // the chart's height follows the window in the wide layout, so watch the canvas itself
  if (window.ResizeObserver) {
    new ResizeObserver(() => (testState.dirty = true)).observe(chart);
  } else {
    window.addEventListener("resize", () => (testState.dirty = true));
  }
}

/**
 * Event listener for clicks on the main start button
 */
function startButtonClickHandler() {
  switch (testState.state) {
    case READY:
    case FINISHED:
      testState.testData = null;
      testState.samples = [];
      testState.loaded = { download: null, upload: null };
      testState.loadedDone = { download: false, upload: false };
      testState.aborted = false;
      testState.hoverX = null;
      if (!testState.customStreams) {
        // over HTTP/1.1 browsers open at most 6 connections to a server, so leave room for the latency pings
        testState.speedtest.setParameter("xhr_dlMultistream", 4);
      }
      // the first ping goes out as soon as a transfer starts, instead of waiting for a script over a busy line
      discardReadyWorkers();
      testState.readyWorkers = { download: newLatencyWorker(), upload: newLatencyWorker() };
      testState.startedAt = performance.now();
      testState.speedtest.start();
      testState.state = RUNNING;
      testState.dirty = true;
      return;
    case RUNNING:
      testState.speedtest.abort();
      // testState.state is updated by `onend` handler of speedtest
      return;
    default:
      return;
  }
}

/**
 * Event listener for clicks on the "Copy link" button in the share dialog
 */
async function copyLinkButtonClickHandler() {
  const button = $("copy-link");
  try {
    await navigator.clipboard.writeText($("share-url").value);
    button.textContent = I18N.t("index.share.copied");
  } catch (error) {
    // clipboard access can be refused; leave the link selected for a manual copy
    $("share-url").select();
    button.textContent = I18N.t("index.share.manual");
  }
  setTimeout(() => (button.textContent = I18N.t("index.share.copy")), 3000);
}

/**
 * Load settings from settings.json on the server and apply them
 */
async function applySettingsJSON() {
  try {
    const response = await fetch("settings.json");
    if (!response.ok) return;
    const settings = await response.json();
    if (!settings || typeof settings !== "object") {
      return console.error("Settings are empty or malformed");
    }
    for (let setting in settings) {
      testState.speedtest.setParameter(setting, settings[setting]);
      if (setting == "xhr_dlMultistream") testState.customStreams = true;
      if (
        setting == "telemetry_level" &&
        settings[setting] &&
        settings[setting] != "off" &&
        settings[setting] != "disabled" &&
        settings[setting] != "false"
      ) {
        testState.telemetryEnabled = true;
        $("privacy-warning").hidden = false;
      }
    }
  } catch (error) {
    console.error("Failed to fetch settings:", error);
  }
}

/**
 * Load server list from the configured source and populate the dropdown.
 * Falls back to the local server if the list is missing, malformed or empty.
 */
async function applyServerListJSON() {
  try {
    const serverSource =
      typeof globalThis.SPEEDTEST_SERVERS !== "undefined"
        ? globalThis.SPEEDTEST_SERVERS
        : "server-list.json";
    let servers = serverSource;
    if (!Array.isArray(serverSource)) {
      try {
        servers = await fetch(serverSource).then((response) => response.json());
      } catch (error) {
        servers = null;
      }
    }
    if (!servers || !Array.isArray(servers) || servers.length === 0) {
      console.info("No server list available, using the local server");
      servers = [LOCAL_SERVER];
    }

    testState.servers = servers;

    // If there's only one server, just show it. No reachability checks needed.
    if (servers.length === 1) {
      populateDropdown(servers, servers[0]);
      selectServer(servers[0]);
      return;
    }

    // For multiple servers: first run the built-in selection (which pings servers
    // and annotates them with pingT). Only then populate the dropdown so that
    // dead servers don't appear.
    testState.speedtest.addTestPoints(servers);
    testState.speedtest.selectServer((bestServer) => {
      const aliveServers = testState.servers.filter((s) => {
        // Keep servers that responded to ping (pingT !== -1).
        if (s.pingT !== -1) return true;
        // Also keep protocol-relative servers ("//...") as a defensive fallback.
        // LibreSpeed normalizes them to the page protocol before pinging, so they
        // are normally treated like any other server and get a real pingT value.
        return typeof s.server === "string" && s.server.startsWith("//");
      });

      // Prefer to show only reachable servers, but if none are reachable,
      // fall back to the full list so users can still pick a server manually.
      if (aliveServers.length > 0) {
        testState.servers = aliveServers;
      }
      populateDropdown(testState.servers, bestServer);

      if (bestServer) {
        selectServer(bestServer);
      } else {
        testState.noServer = true;
        testState.dirty = true;
      }
    });
  } catch (error) {
    console.error("Failed to load server list:", error);
  }
}

/**
 * Redo the closed select's copy of the chosen option, which doesn't follow a change to the option's text
 */
function refreshServerSelect() {
  const select = $("server");
  const index = select.selectedIndex;
  select.selectedIndex = -1;
  select.selectedIndex = index;
}

/**
 * Fill the server dropdown
 * @param {Array} servers - an array of server objects
 * @param {Object|null} selected - the server to show as selected
 */
function populateDropdown(servers, selected) {
  const select = $("server");
  // remove the options but keep the select's button
  while (select.options.length > 0) select.remove(0);

  if (!selected) {
    const placeholder = document.createElement("option");
    placeholder.value = "";
    placeholder.setAttribute("data-i18n", "index.chooseServer");
    placeholder.textContent = I18N.t("index.chooseServer");
    select.appendChild(placeholder);
  }
  servers.forEach((server, index) => {
    const option = document.createElement("option");
    option.value = index;
    if (server === LOCAL_SERVER) {
      // the built-in fallback gets a translated name; names from a server list are shown as given
      option.setAttribute("data-i18n", "index.localServer");
      option.textContent = I18N.t("index.localServer");
    } else {
      option.textContent = server.sponsorName ? `${server.name} (${server.sponsorName})` : server.name;
    }
    option.selected = server === selected;
    select.appendChild(option);
  });

  // a single server is shown, not chosen
  select.classList.toggle("single", servers.length === 1);
}

/**
 * Set the given server as the selected server for the speedtest
 * @param {Object} server - a server object
 */
function selectServer(server) {
  testState.speedtest.setSelectedServer(server);
  testState.noServer = false;
  testState.state = READY;
  testState.dirty = true;
  showSponsor(server);
}

/**
 * Show the sponsor of the selected server, linked only to http(s) URLs
 * @param {Object} server - a server object
 */
function showSponsor(server) {
  const sponsor = $("sponsor");
  sponsor.textContent = "";
  sponsor.hidden = !server.sponsorName;
  if (!server.sponsorName) return;

  sponsor.append(I18N.t("index.sponsoredBy"));
  if (server.sponsorURL && /^https?:\/\//i.test(server.sponsorURL)) {
    const link = document.createElement("a");
    link.href = server.sponsorURL;
    link.rel = "noopener";
    link.textContent = server.sponsorName;
    sponsor.appendChild(link);
  } else {
    sponsor.append(server.sponsorName);
  }
}

/**
 * Remember the current speed for the chart while download or upload runs
 * @param {Object} data - status update from the speedtest worker
 */
function recordSample(data) {
  const t = (performance.now() - testState.startedAt) / 1000;
  let v = NaN;
  let kind = null;
  if (data.testState === 1) {
    v = Number(data.dlStatus);
    kind = "dl";
  } else if (data.testState === 3) {
    v = Number(data.ulStatus);
    kind = "ul";
  }
  if (kind && data[kind + "Status"] !== "" && !isNaN(v)) {
    testState.samples.push({ t, v, kind });
  }
}

/**
 * Ping the server while the download or upload runs, with a new stability worker for each
 * @param {Object} data - status update from the speedtest worker
 */
function trackLoadedLatency(data) {
  // the worker keeps its state through the pause after a transfer, but the transfer's progress is 1 by then
  let kind = null;
  if (data.testState === 1 && Number(data.dlProgress) < 1) kind = "download";
  else if (data.testState === 3 && Number(data.ulProgress) < 1) kind = "upload";
  if (kind === testState.latencyKind) return;

  stopLatencyWorker();
  if (!kind) return;
  const server = testState.speedtest.getSelectedServer();
  // pings run in their own worker, so the download's resource timings don't mix with theirs
  const worker = testState.readyWorkers[kind] || newLatencyWorker();
  delete testState.readyWorkers[kind];
  const updater = setInterval(() => worker.postMessage("status"), 200);
  const run = testState.startedAt;
  worker.onmessage = (e) => {
    const status = JSON.parse(e.data);
    // the last ping's grace can outlast its test, and a new test has its own pingers
    if (testState.startedAt !== run) {
      clearInterval(updater);
      worker.terminate();
      return;
    }
    const stats = status.phases[0];
    if (stats.samples > stats.failed) testState.loaded[kind] = stats;
    if (status.testState >= 4) {
      clearInterval(updater);
      worker.terminate();
      testState.loadedDone[kind] = true;
      testState.speedtest.setLoadedLatency(loadedLatencySummary());
    }
    testState.dirty = true;
  };
  // a congested line can hold a ping for seconds, which is the latency under load rather than a lost ping,
  // so pings wait as long as they take, and the last one gets 2 s after the transfer ends to come back
  const settings = { url_ping: server.server + server.pingURL, duration: 0, mpot: true, ping_timeout: 0, finish_grace: 2000 };
  worker.postMessage("start " + JSON.stringify(settings));
  testState.latencyWorker = worker;
  testState.latencyKind = kind;
}

/**
 * The latency under load as stored with the results, rounded to hundredths of a millisecond
 */
function loadedLatencySummary() {
  const round = (v) => Math.round(v * 100) / 100;
  const summary = {};
  for (const kind of ["download", "upload"]) {
    const stats = testState.loaded[kind];
    if (!stats) continue;
    summary[kind] = { avg: round(stats.avg), jitter: round(stats.jitter), low: round(stats.min), high: round(stats.max) };
    // the jitter has a range once two pings came back
    if (stats.jitterMin >= 0) {
      summary[kind].jitterLow = round(stats.jitterMin);
      summary[kind].jitterHigh = round(stats.jitterMax);
    }
  }
  return summary;
}

function newLatencyWorker() {
  return new Worker("stability_worker.js?r=" + Math.random());
}

// pingers made for a transfer the test never ran, such as after it was stopped
function discardReadyWorkers() {
  for (const kind in testState.readyWorkers) testState.readyWorkers[kind].terminate();
  testState.readyWorkers = {};
}

function stopLatencyWorker() {
  const worker = testState.latencyWorker;
  testState.latencyWorker = null;
  testState.latencyKind = null;
  if (!worker) return;
  // the status after "finish" carries the last pings and ends the worker
  worker.postMessage("finish");
  worker.postMessage("status");
}

/**
 * Start the requestAnimationFrame UI rendering loop
 */
function startRenderingLoop() {
  const buttonTexts = {
    [INITIALIZING]: "btn.loading",
    [READY]: "btn.start",
    [RUNNING]: "btn.stop",
    [FINISHED]: "btn.start",
  };

  function renderUI() {
    // the elapsed time ticks while the test runs, everything else only changes on events
    if (testState.state === RUNNING) {
      $("elapsed").textContent = formatTime((performance.now() - testState.startedAt) / 1000);
    }
    if (testState.dirty) {
      testState.dirty = false;
      renderControls(buttonTexts);
      renderResults();
      drawChart();
    }
    requestAnimationFrame(renderUI);
  }

  renderUI();
}

function renderControls(buttonTexts) {
  const state = testState.state;
  const data = testState.testData;

  const startButton = $("start-button");
  startButton.textContent = I18N.t(testState.noServer ? "btn.noServer" : buttonTexts[state]);
  startButton.disabled = state === INITIALIZING;
  startButton.classList.toggle("running", state === RUNNING);

  // the server can't change during a test, and a single server is shown, not chosen
  const select = $("server");
  select.disabled = state === RUNNING || select.options.length <= 1 || (state === INITIALIZING && !testState.noServer);

  // overall progress across the phases the worker reports
  let progress = 0;
  if (state === FINISHED) progress = 1;
  else if (data) progress = (Number(data.dlProgress) + Number(data.ulProgress) + Number(data.pingProgress)) / 3;
  const pct = Math.round(Math.min(1, Math.max(0, progress)) * 100);
  $("progress-bar").style.width = pct + "%";
  $("progress-track").setAttribute("aria-valuenow", pct);
  $("progress-percent").textContent = pct + "%";
  if (state !== RUNNING && !data) $("elapsed").textContent = "00:00";

  renderStatus();

  // Show share button after test if server supports it
  const testId = data && data.testId;
  $("share-results").hidden = !(state === FINISHED && testState.telemetryEnabled && testId);
  if (testId) {
    // the image is drawn in the page's language
    const lang = I18N.lang() === "en" ? "" : "&lang=" + I18N.lang();
    const url =
      window.location.href.substring(0, window.location.href.lastIndexOf("/")) + "/results/?id=" + testId + lang;
    if ($("share-url").value !== url) {
      $("share-url").value = url;
      $("results").src = url;
    }
  }
}

function renderStatus() {
  const data = testState.testData;
  let cls = "idle";
  let title;
  let text;

  if (testState.noServer) {
    cls = "bad";
    title = I18N.t("index.status.noServer");
    text = I18N.t("index.status.noServerText");
  } else if (testState.state === INITIALIZING) {
    title = I18N.t("index.status.loading");
    text = I18N.t("server.findingBest");
  } else if (testState.state === RUNNING) {
    const phase = data ? data.testState : 0;
    const phases = { 1: "index.status.download", 2: "index.status.ping", 3: "index.status.upload" };
    title = I18N.t(phases[phase] || "index.status.starting");
    text = I18N.t("index.status.running");
  } else if (testState.state === FINISHED && data) {
    cls = "ok";
    title = I18N.t("index.status.complete");
    // one sentence for the speeds and one for the latency of each phase; jitter is left to its card
    const sentences = [];
    const speeds = summaryParts([
      ["index.summary.download", data.dlStatus],
      ["index.summary.upload", data.ulStatus],
    ]);
    if (speeds.length) sentences.push(capitalize(speeds.join(", ")) + ".");
    const latency = summaryParts([
      ["index.summary.idle", idleLatency(data)],
      ["index.summary.downloading", testState.loaded.download && testState.loaded.download.avg],
      ["index.summary.uploading", testState.loaded.upload && testState.loaded.upload.avg],
    ]);
    if (latency.length) sentences.push(I18N.t("index.summary.latency", { v: latency.join(", ") }));
    text = sentences.length ? sentences.join(" ") : I18N.t("index.status.none");
  } else if (testState.aborted) {
    title = I18N.t("status.stopped");
    text = I18N.t("index.status.stoppedText");
  } else {
    title = I18N.t("status.ready");
    text = I18N.t("index.status.readyText");
  }

  $("status").className = "verdict " + cls;
  $("status-title").textContent = title;
  $("status-text").textContent = text;
}

function renderResults() {
  const data = testState.testData || {};
  setStat("download-speed", formatNumber(data.dlStatus));
  setStat("upload-speed", formatNumber(data.ulStatus));
  setStat("ping", formatNumber(idleLatency(data)));
  setStat("jitter", formatNumber(data.jitterStatus));
  // the lowest ping reads 0 before the first pong, so its range waits for the average
  const pinged = formatNumber(idleLatency(data)) !== "--";
  setRange("ping-range", pinged ? data.pingStatus : null, data.pingHigh);
  setRange("jitter-range", data.jitterLow, data.jitterHigh);
  setLoaded("download-latency", "download");
  setLoaded("upload-latency", "upload");
  $("download-progress").style.width = Math.round((Number(data.dlProgress) || 0) * 100) + "%";
  $("upload-progress").style.width = Math.round((Number(data.ulProgress) || 0) * 100) + "%";

  const clientIp = $("client-ip");
  clientIp.textContent = data.clientIp || I18N.t("index.connectionEmpty");
  clientIp.classList.toggle("empty", !data.clientIp);
}

// units are hidden until there is a value to go with them
function setStat(id, text) {
  const el = $(id);
  el.textContent = text;
  el.parentNode.classList.toggle("empty", text === "--");
}

// the lowest and highest under the average, hidden until both are measured
function setRange(id, low, high) {
  const el = $(id);
  const shown = formatNumber(low) !== "--" && formatNumber(high) !== "--";
  el.textContent = shown ? formatNumber(low) + "–" + formatNumber(high) : "";
  el.hidden = !shown;
}

// the average of the ping test, like the latency under load; workers without it report the lowest ping
function idleLatency(data) {
  return data.pingAvgStatus !== undefined ? data.pingAvgStatus : data.pingStatus;
}

// the numbers stand out from the words around them, like the label before them; the text
// carries its own separator from the label, so "no reply" can follow it with a colon
function setLoaded(id, kind) {
  const el = $(id);
  const stats = testState.loaded[kind];
  if (!stats) {
    // the pinger finished without a single reply, which says more than "--"
    const noReply = testState.loadedDone[kind] && !testState.aborted;
    el.textContent = noReply ? ": " + I18N.t("index.loaded.noReply") : " --";
    return;
  }
  const value = (v) => {
    const span = document.createElement("span");
    span.className = "stat-sub-value";
    span.textContent = formatNumber(v) + " " + I18N.t("unit.ms");
    return span;
  };
  el.replaceChildren(" ", value(stats.avg), ", " + I18N.t("index.loaded.jitter") + " ", value(stats.jitter));
}

/**
 * Draw download and upload speed over time, in CSS pixels
 */
function drawChart() {
  const canvas = $("speedChart");
  const ctx = canvas.getContext("2d");
  const dp = window.devicePixelRatio || 1;
  const w = canvas.clientWidth;
  const h = canvas.clientHeight;
  if (canvas.width !== Math.round(w * dp) || canvas.height !== Math.round(h * dp)) {
    canvas.width = Math.round(w * dp);
    canvas.height = Math.round(h * dp);
  }
  ctx.setTransform(dp, 0, 0, dp, 0, 0);
  ctx.clearRect(0, 0, w, h);
  ctx.font = "11px ui-sans-serif, system-ui, sans-serif";
  ctx.lineWidth = 1;

  const samples = testState.samples;
  let maxV = 0;
  for (const s of samples) if (s.v > maxV) maxV = s.v;
  const scale = niceScale(maxV);
  const lastT = samples.length ? samples[samples.length - 1].t : 0;
  const timeEnd = Math.max(MIN_CHART_SECONDS, Math.ceil(lastT / 5) * 5);

  // leave room on the left for the widest value label
  const padL = Math.ceil(ctx.measureText(formatAxis(scale.max)).width) + 14;
  const padT = CHART_PADDING.top;
  const plotW = w - padL - CHART_PADDING.right;
  const plotH = h - padT - CHART_PADDING.bottom;
  const baseY = padT + plotH;
  const xFor = (t) => padL + (t / timeEnd) * plotW;
  const yFor = (v) => baseY - (v / scale.max) * plotH;

  // horizontal grid lines with value labels
  ctx.textAlign = "right";
  ctx.textBaseline = "middle";
  for (let v = 0; v <= scale.max + scale.step / 2; v += scale.step) {
    const y = Math.round(yFor(v)) + 0.5;
    ctx.strokeStyle = CHART_COLORS.grid;
    ctx.beginPath();
    ctx.moveTo(padL, y);
    ctx.lineTo(padL + plotW, y);
    ctx.stroke();
    ctx.fillStyle = CHART_COLORS.text;
    ctx.fillText(formatAxis(v), padL - 8, y);
  }

  // time labels, kept inside the canvas at both ends
  ctx.textAlign = "center";
  ctx.textBaseline = "top";
  const maxLabels = Math.floor(plotW / (ctx.measureText("00:00").width + 24)) + 1;
  const intervals = [5, 10, 15, 20, 30, 60];
  let tickInterval = intervals[intervals.length - 1];
  for (const interval of intervals) {
    if (timeEnd / interval + 1 <= maxLabels) {
      tickInterval = interval;
      break;
    }
  }
  for (let t = 0; t <= timeEnd; t += tickInterval) {
    const label = formatTime(t);
    const half = ctx.measureText(label).width / 2;
    ctx.fillText(label, Math.min(Math.max(xFor(t), padL + half), padL + plotW - half), baseY + 8);
  }

  const tooltip = $("chart-tooltip");
  if (samples.length === 0) {
    tooltip.hidden = true;
    return;
  }

  // one filled line per phase
  for (const kind of ["dl", "ul"]) {
    const points = samples.filter((s) => s.kind === kind);
    if (points.length === 0) continue;
    const color = kind === "dl" ? CHART_COLORS.download : CHART_COLORS.upload;
    const fill = kind === "dl" ? CHART_COLORS.downloadFill : CHART_COLORS.uploadFill;
    const fillEnd = kind === "dl" ? CHART_COLORS.downloadFillEnd : CHART_COLORS.uploadFillEnd;

    const gradient = ctx.createLinearGradient(0, padT, 0, baseY);
    gradient.addColorStop(0, fill);
    gradient.addColorStop(1, fillEnd);
    ctx.beginPath();
    ctx.moveTo(xFor(points[0].t), baseY);
    for (const p of points) ctx.lineTo(xFor(p.t), yFor(p.v));
    ctx.lineTo(xFor(points[points.length - 1].t), baseY);
    ctx.closePath();
    ctx.fillStyle = gradient;
    ctx.fill();

    ctx.beginPath();
    points.forEach((p, i) => (i === 0 ? ctx.moveTo(xFor(p.t), yFor(p.v)) : ctx.lineTo(xFor(p.t), yFor(p.v))));
    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.lineJoin = "round";
    ctx.lineCap = "round";
    ctx.stroke();
  }

  // hover crosshair and tooltip on the nearest sample
  if (testState.hoverX === null || testState.hoverX < padL) {
    tooltip.hidden = true;
    return;
  }
  const hoverT = ((testState.hoverX - padL) / plotW) * timeEnd;
  let nearest = samples[0];
  for (const s of samples) {
    if (Math.abs(s.t - hoverT) < Math.abs(nearest.t - hoverT)) nearest = s;
  }
  const nx = xFor(nearest.t);
  const ny = yFor(nearest.v);
  ctx.strokeStyle = CHART_COLORS.reference;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(Math.round(nx) + 0.5, padT);
  ctx.lineTo(Math.round(nx) + 0.5, baseY);
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(nx, ny, 4, 0, Math.PI * 2);
  ctx.fillStyle = nearest.kind === "dl" ? CHART_COLORS.download : CHART_COLORS.upload;
  ctx.fill();
  ctx.strokeStyle = CHART_COLORS.surface;
  ctx.lineWidth = 2;
  ctx.stroke();

  $("tooltip-label").textContent = formatTime(nearest.t) + " " + I18N.t(nearest.kind === "dl" ? "index.download" : "index.upload");
  $("tooltip-value").textContent = formatNumber(nearest.v) + " " + I18N.t("unit.mbps");
  tooltip.hidden = false;
  const half = tooltip.offsetWidth / 2;
  tooltip.style.left = Math.min(Math.max(nx, half), w - half) + "px";
  tooltip.style.top = Math.max(ny, tooltip.offsetHeight + 12) + "px";
}

/**
 * Pick a round axis maximum and step with about four grid lines
 * @param {number} max - the largest value to show
 */
function niceScale(max) {
  max = Math.max(max * 1.15, 10);
  const raw = max / 4;
  const magnitude = Math.pow(10, Math.floor(Math.log10(raw)));
  const norm = raw / magnitude;
  const step = (norm <= 1 ? 1 : norm <= 2 ? 2 : norm <= 2.5 ? 2.5 : norm <= 5 ? 5 : 10) * magnitude;
  return { step, max: Math.ceil(max / step) * step };
}

function formatAxis(v) {
  v = Math.round(v * 10) / 10;
  return I18N.num(v, Number.isInteger(v) ? 0 : 1);
}

function formatTime(sec) {
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return (m < 10 ? "0" : "") + m + ":" + (s < 10 ? "0" : "") + s;
}

/**
 * Convert a number to a user friendly version
 * @param {string|number} value Speed, ping or jitter
 * @returns {string} A text version with proper decimals, or "--" when there is none
 */
function formatNumber(value) {
  // "" means not measured yet; 0 is a real result (jitter on a quiet link)
  if (value === "" || value === null || value === undefined) return "--";
  value = Number(value);
  if (isNaN(value) || value < 0) return "--";
  return I18N.num(value, value < 10 ? 2 : value < 100 ? 1 : 0);
}

/**
 * Fill in each measured value's text, skipping the ones not measured
 * @param {Array} results - pairs of a translation key and a value
 * @returns {string[]} The texts
 */
function summaryParts(results) {
  const parts = [];
  for (const [key, value] of results) {
    if (formatNumber(value) !== "--") parts.push(I18N.t(key, { v: formatNumber(value) }));
  }
  return parts;
}

function capitalize(text) {
  return text.charAt(0).toUpperCase() + text.slice(1);
}
