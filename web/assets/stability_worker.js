/*
	LibreSpeed - Stability Test Worker
	https://github.com/librespeed/speedtest/
	GNU LGPLv3 License
*/

// data reported to main thread
let testState = -1; // -1=idle, 0=starting, 1=running, 4=finished, 5=aborted
let currentPing = 0;
let elapsed = 0;
let progress = 0;

// latency over the whole test, and per phase (0=idle, 1=download, 2=upload) when the page runs a load alongside
let overall = newStats();
let phase = 0;
let phases = [newStats()];

let pingData = []; // all ping data points {t: elapsedMs, ping: ms, lost: bool, phase: n}
let lastReportedIndex = 0; // for delta delivery

let settings = {
  url_ping: "backend/empty.php",
  url_ping_external: "", // external URL to ping (uses fetch no-cors, e.g. "https://www.google.com/generate_204")
  duration: 60, // seconds, 0 runs until the page sends "finish"
  ping_interval: 200, // minimum ms between pings to limit sample rate
  ping_timeout: 5000,
  ping_allowPerformanceApi: true,
  mpot: false
};

let xhr = null;
let startTime = 0;
let prevInstspd = 0;
let aborted = false;

function url_sep(url) {
  return url.match(/\?/) ? "&" : "?";
}

function newStats() {
  return { samples: 0, failed: 0, sum: 0, avg: 0, min: -1, max: 0, jitter: 0, jitterSum: 0, jitterCount: 0, jitterMin: -1, jitterMax: 0, loss: 0, prev: 0 };
}

function addPing(s, ms) {
  s.samples++;
  s.sum += ms;
  const received = s.samples - s.failed;
  s.avg = parseFloat((s.sum / received).toFixed(2));
  if (s.min === -1 || ms < s.min) s.min = ms;
  if (ms > s.max) s.max = ms;

  // jitter is the average change between consecutive pings, so a spike counts as much
  // at the end of the test as at the start, with the smallest and largest change as its range
  if (received > 1 && s.prev > 0) {
    const instjitter = Math.abs(ms - s.prev);
    s.jitterSum += instjitter;
    s.jitterCount++;
    s.jitter = parseFloat((s.jitterSum / s.jitterCount).toFixed(2));
    if (s.jitterMin === -1 || instjitter < s.jitterMin) s.jitterMin = instjitter;
    if (instjitter > s.jitterMax) s.jitterMax = instjitter;
  }
  s.prev = ms;
  s.loss = parseFloat(((s.failed / s.samples) * 100).toFixed(2));
}

function addLoss(s) {
  s.samples++;
  s.failed++;
  s.loss = parseFloat(((s.failed / s.samples) * 100).toFixed(2));
}

function phaseSummary(s) {
  return {
    avg: s.avg,
    min: s.min,
    max: s.max,
    jitter: s.jitter,
    jitterMin: s.jitterMin,
    jitterMax: s.jitterMax,
    loss: s.loss,
    samples: s.samples,
    failed: s.failed
  };
}

this.addEventListener("message", function (e) {
  const params = e.data.split(" ");
  if (params[0] === "status") {
    // return current state with delta ping data
    const newData = pingData.slice(lastReportedIndex);
    lastReportedIndex = pingData.length;
    postMessage(
      JSON.stringify({
        testState: testState,
        currentPing: currentPing,
        avgPing: overall.avg,
        minPing: overall.min,
        maxPing: overall.max,
        jitter: overall.jitter,
        jitterMin: overall.jitterMin,
        jitterMax: overall.jitterMax,
        packetLoss: overall.loss,
        elapsed: elapsed,
        duration: settings.duration,
        progress: progress,
        pingData: newData,
        totalSamples: overall.samples,
        failedSamples: overall.failed,
        phase: phase,
        phases: phases.map(phaseSummary)
      })
    );
  }
  if (params[0] === "start" && testState === -1) {
    testState = 0;
    try {
      let s = {};
      try {
        const ss = e.data.substring(6);
        if (ss) s = JSON.parse(ss);
      } catch (e) {
        console.warn("Error parsing settings JSON");
      }
      for (let key in s) {
        if (typeof settings[key] !== "undefined") settings[key] = s[key];
      }
    } catch (e) {
      console.warn("Error applying settings: " + e);
    }
    // start the stability test
    aborted = false;
    startTime = new Date().getTime();
    testState = 1;
    doPing();
  }
  if (params[0] === "phase" && testState === 1) {
    // later pings are recorded under this phase
    phase = parseInt(params[1]) || 0;
    while (phases.length <= phase) phases.push(newStats());
  }
  if (params[0] === "finish") {
    // the page ends a test that has no fixed duration
    if (testState >= 4) return;
    testState = 4;
    progress = 1;
    if (xhr) {
      try {
        xhr.abort();
      } catch (e) {}
    }
  }
  if (params[0] === "abort") {
    if (testState >= 4) return;
    aborted = true;
    testState = 5;
    if (xhr) {
      try {
        xhr.abort();
      } catch (e) {}
    }
  }
});

function recordPing(instspd) {
  // guard against 0ms pings
  if (instspd < 1) instspd = prevInstspd;
  if (instspd < 1) instspd = 1;
  prevInstspd = instspd;

  currentPing = instspd;
  addPing(overall, instspd);
  addPing(phases[phase], instspd);

  // record data point
  const now = new Date().getTime();
  elapsed = (now - startTime) / 1000;
  pingData.push({ t: elapsed, ping: parseFloat(instspd.toFixed(2)), lost: false, phase: phase });
}

function recordLoss() {
  const now = new Date().getTime();
  addLoss(overall);
  addLoss(phases[phase]);
  elapsed = (now - startTime) / 1000;
  pingData.push({ t: elapsed, ping: 0, lost: true, phase: phase });
}

// pace pings to avoid excessive sample rates on low-latency links
function schedulePing(rtt) {
  const delay = Math.max(0, settings.ping_interval - rtt);
  if (delay > 0) {
    setTimeout(doPing, delay);
  } else {
    doPing();
  }
}

function doPing() {
  if (aborted || testState >= 4) return;

  // check if duration exceeded
  const now = new Date().getTime();
  elapsed = (now - startTime) / 1000;
  if (settings.duration > 0) {
    progress = Math.min(1, elapsed / settings.duration);
    if (elapsed >= settings.duration) {
      testState = 4;
      progress = 1;
      return;
    }
  }

  // external ping mode: use fetch with no-cors
  if (settings.url_ping_external) {
    doPingExternal();
    return;
  }

  const prevT = new Date().getTime();
  xhr = new XMLHttpRequest();
  xhr.onload = function () {
    if (aborted || testState >= 4) return;
    const now = new Date().getTime();
    let instspd = now - prevT;

    if (settings.ping_allowPerformanceApi) {
      try {
        let p = performance.getEntries();
        p = p[p.length - 1];
        let d = p.responseStart - p.requestStart;
        if (d <= 0) d = p.duration;
        if (d > 0 && d < instspd) instspd = d;
      } catch (e) {
        // Performance API not available, use estimate
      } finally {
        try {
          performance.clearResourceTimings();
        } catch (e) {}
      }
    }

    recordPing(instspd);
    schedulePing(instspd);
  };
  xhr.onerror = function () {
    if (aborted || testState >= 4) return;
    recordLoss();
    schedulePing(0);
  };
  xhr.ontimeout = xhr.onerror;
  xhr.open(
    "GET",
    settings.url_ping + url_sep(settings.url_ping) + (settings.mpot ? "cors=true&" : "") + "r=" + Math.random(),
    true
  );
  try {
    xhr.timeout = settings.ping_timeout;
  } catch (e) {}
  xhr.send();
}

// ping an external host using fetch with no-cors (opaque response, but timing still works)
function doPingExternal() {
  const prevT = new Date().getTime();
  const remainingMs =
    settings.duration > 0 ? Math.max(1, settings.duration * 1000 - (prevT - startTime)) : settings.ping_timeout;
  const timeoutMs = Math.min(settings.ping_timeout, remainingMs);
  // no cache-busting parameter: "no-store" already skips the browser cache, and a unique URL would push
  // every ping past the target's CDN edge to its origin, measuring their backend instead of the connection
  const url = settings.url_ping_external;

  let timeoutId = null;
  const controller = typeof AbortController !== "undefined" ? new AbortController() : null;
  const fetchOptions = { mode: "no-cors", cache: "no-store" };
  if (controller) fetchOptions.signal = controller.signal;

  const timeout = new Promise(function (_, reject) {
    timeoutId = setTimeout(function () {
      if (controller) controller.abort();
      reject(new Error("timeout"));
    }, timeoutMs);
  });

  Promise.race([fetch(url, fetchOptions), timeout])
    .then(function () {
      if (timeoutId) clearTimeout(timeoutId);
      if (aborted || testState >= 4) return;
      const instspd = new Date().getTime() - prevT;
      recordPing(instspd);
      schedulePing(instspd);
    })
    .catch(function () {
      if (timeoutId) clearTimeout(timeoutId);
      if (aborted || testState >= 4) return;
      recordLoss();
      schedulePing(0);
    });
}
