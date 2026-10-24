/**
 * Translations for the speed test, stability and statistics pages.
 *
 * Static text is marked in the HTML with data-i18n="key" (text content) and
 * data-i18n-attr="attribute:key;attribute:key". Text set from scripts goes
 * through I18N.t(). Elements whose text a script sets must not carry
 * data-i18n, or the two would overwrite each other.
 *
 * The chosen language is remembered in localStorage; otherwise it follows the
 * browser's preferred languages.
 */
(function () {
  var STRINGS = {
    en: {
      "lang.label": "Language",
      "nav.stability": "Stability test",
      "footer.source": "Source code",
      "unit.ms": "ms",
      "unit.mbps": "Mbps",
      "btn.start": "Start",
      "btn.stop": "Stop",
      "btn.reset": "Reset",
      "btn.loading": "Loading…",
      "btn.noServer": "No server available",
      "btn.close": "Close",
      "field.server": "Server",
      "server.findingBest": "Finding the best server…",
      "status.ready": "Ready",
      "status.stopped": "Stopped",
      "aria.testSettings": "Test settings",
      "aria.results": "Results",
      "aria.progress": "Test progress",

      // speed test page
      "index.title": "LibreSpeed - Free and Open Source Speedtest",
      "index.tagline": "No Flash, No Java, No Websockets, No Bullsh*t.",
      "index.chooseServer": "Choose a server",
      "index.localServer": "Local server",
      "index.sponsoredBy": "Sponsored by ",
      "index.share": "Share",
      "index.status.noServer": "No server reachable",
      "index.status.noServerText": "None of the servers answered. Choose one from the list to try anyway.",
      "index.status.loading": "Loading",
      "index.status.download": "Testing download…",
      "index.status.ping": "Measuring ping…",
      "index.status.upload": "Testing upload…",
      "index.status.starting": "Starting…",
      "index.status.running": "Keep this tab open until the test finishes.",
      "index.status.complete": "Test complete",
      "index.status.none": "No results were measured.",
      "index.status.stoppedText": "The test was stopped before it finished.",
      "index.status.readyText": "Press Start to measure download, upload and latency.",
      "index.summary.download": "download {v} Mbps",
      "index.summary.upload": "upload {v} Mbps",
      "index.summary.latency": "Latency: {v}.",
      "index.summary.idle": "idle {v} ms",
      "index.summary.downloading": "downloading {v} ms",
      "index.summary.uploading": "uploading {v} ms",
      "index.connection": "Your connection",
      "index.connectionEmpty": "Shown when the test starts",
      "index.privacyNote": "Results are recorded when the test finishes. By starting the test you agree to the",
      "index.privacyLink": "privacy policy",
      "index.download": "Download",
      "index.upload": "Upload",
      "index.ping": "Ping",
      "index.jitter": "Jitter",
      "index.idleLatency": "Idle latency",
      "index.idleJitter": "Idle jitter",
      "index.loaded": "Latency",
      "index.loaded.jitter": "jitter",
      "index.chart.title": "Speed",
      "index.chart.note": "Megabits per second over the test",
      "index.chart.aria": "Download and upload speed over the test",
      "index.aria.chart": "Speed chart",
      "index.share.title": "Share results",
      "index.share.alt": "Test results in graphical form",
      "index.share.url": "Link to the results image",
      "index.share.copy": "Copy link",
      "index.share.copied": "Copied!",
      "index.share.manual": "Press Ctrl+C",
      "privacy.title": "Privacy Policy",
      "privacy.intro": "This HTML5 speed test server is configured with telemetry enabled.",
      "privacy.collect": "What data we collect",
      "privacy.collectIntro": "At the end of the test, the following data is collected and stored:",
      "privacy.item.id": "Test ID",
      "privacy.item.time": "Time of testing",
      "privacy.item.results": "Test results (download and upload speed, ping and jitter)",
      "privacy.item.ip": "IP address",
      "privacy.item.isp": "ISP information",
      "privacy.item.location": "Approximate location (inferred from IP address, not GPS)",
      "privacy.item.agent": "User agent and browser locale",
      "privacy.item.log": "Test log (contains no personal information)",
      "privacy.use": "How we use the data",
      "privacy.useIntro": "Data collected through this service is used to:",
      "privacy.use.share": "Allow sharing of test results (sharable image for forums, etc.)",
      "privacy.use.improve": "To improve the service offered to you (for instance, to detect problems on our side)",
      "privacy.thirdParties": "No personal information is disclosed to third parties.",
      "privacy.consent": "Your consent",
      "privacy.consentText": "By starting the test, you consent to the terms of this privacy policy.",
      "privacy.removal": "Data removal",
      "privacy.removalText":
        "If you want to have your information deleted, you need to provide either the ID of the test or your IP address. This is the only way to identify your data, without this information we won't be able to comply with your request.",
      "privacy.contact": "Contact this email address for all deletion requests:",

      // stability page
      "stab.title": "LibreSpeed - Stability Test",
      "stab.heading": "Stability Test",
      "stab.subtitle": "Measure latency, jitter and failed requests over time, idle or under load.",
      "stab.duration": "Duration",
      "stab.duration.60": "1 minute",
      "stab.duration.90": "90 seconds",
      "stab.duration.120": "2 minutes",
      "stab.duration.180": "3 minutes",
      "stab.duration.300": "5 minutes",
      "stab.load": "Load",
      "stab.load.none": "Idle",
      "stab.load.updown": "Download and upload",
      "stab.load.title": "Latency under load",
      "stab.load.note": "Idle, then downloading, then uploading, a third of the test each.",
      "stab.phase": "Phase",
      "stab.phase.idle": "Idle",
      "stab.phase.download": "Download",
      "stab.phase.upload": "Upload",
      "stab.speed": "Speed",
      "stab.range": "Low–high",
      "stab.loadSummary": "Compared to idle, latency changes by {dl} ms while downloading and {ul} ms while uploading.",
      "stab.loadSummary.download": "Compared to idle, latency changes by {dl} ms while downloading.",
      "stab.target": "Ping target",
      "stab.thisServer": "This server",
      "stab.findingServer": "Finding server…",
      "stab.readyText": "Press Start to measure stability.",
      "stab.measuring": "Measuring…",
      "stab.measuringText": "Waiting for the first response.",
      "stab.stoppedText": "No responses were recorded.",
      "stab.noResponses": "No responses",
      "stab.noResponsesText": "Every request so far has failed.",
      "stab.rating.great": "Great stability",
      "stab.rating.good": "Good stability",
      "stab.rating.poor": "Poor stability",
      "stab.rating.bad": "Bad stability",
      "stab.summary": "Average {avg} ms, jitter {jitter} ms, {loss}% failed requests.",
      "stab.current": "Current",
      "stab.average": "Average",
      "stab.jitter": "Jitter",
      "stab.min": "Min",
      "stab.max": "Max",
      "stab.failed": "Failed",
      "stab.requests": "Requests",
      "stab.threshold": "Alert threshold",
      "stab.thresholdOff": "Off",
      "stab.thresholdNote": "Beep when latency goes above this value.",
      "stab.latency": "Latency",
      "stab.chart.note": "Milliseconds, last 60 seconds",
      "stab.chart.aria": "Latency over the last 60 seconds",
      "stab.key.failed": "Failed request",
      "stab.csv": "Download CSV",
      "stab.csvTitle": "Download all samples as CSV",
      "stab.tooltipFailed": "Failed",
      "stab.aria.chart": "Latency chart",
      "stab.aria.alert": "Alert",

      // statistics page
      "stats.title": "LibreSpeed - Statistics",
      "stats.heading": "Statistics",
      "stats.subtitle": "Results of the speed tests run on this server.",
      "stats.signIn": "Sign in",
      "stats.signOut": "Sign out",
      "stats.signInNote": "Enter the statistics password to see the results.",
      "stats.password": "Password",
      "stats.wrongPassword": "Wrong password",
      "stats.wrongPasswordText": "Check the password and try again.",
      "stats.locked": "Statistics are locked",
      "stats.lockedText": "To enable access, set a password in",
      "stats.disabled": "Statistics are disabled",
      "stats.disabledText": "Results are not stored, because the database type is set to",
      "stats.search": "Find a test by its ID",
      "stats.idPlaceholder": "Test ID",
      "stats.find": "Find",
      "stats.allTests": "All tests",
      "stats.notFound": "Test not found",
      "stats.notFoundText": "No test has this ID:",
      "stats.loadFailed": "Could not load the tests",
      "stats.loadFailedText": "The database returned an error. The details are in the server log.",
      "stats.latest": "Latest tests",
      "stats.latestNote": "Up to 100, newest first. Open a test to see all its details.",
      "stats.empty": "No tests have been recorded yet.",
      "stats.time": "Time",
      "stats.testId": "Test ID",
      "stats.col.download": "Download, Mbps",
      "stats.col.upload": "Upload, Mbps",
      "stats.col.ping": "Ping, ms",
      "stats.col.jitter": "Jitter, ms",
      "stats.result": "Test result",
      "stats.client": "Connection",
      "stats.ip": "IP address",
      "stats.isp": "Provider",
      "stats.location": "Location",
      "stats.lookup": "ISP lookup",
      "stats.agent": "Browser",
      "stats.languages": "Languages",
      "stats.latency": "Latency",
      "stats.latencyNote": "Average while idle, downloading and uploading",
      "stats.latencyJitter": "jitter",
      "stats.latencyRange": "low–high",
      "stats.extra": "Extra data",
      "stats.extraText": "Text",
      "stats.image": "Result image",
      "stats.openImage": "Open",
      "stats.imageAlt": "Result image of this test",
      "stats.log": "Test log",
      "stats.logNote": "Seconds from the start of the test",
      "stats.rawLog": "Show the raw log",
      "stats.noLog": "No log was recorded for this test.",
      "stats.delete": "Delete",
      "stats.deleteSelected": "Delete selected",
      "stats.deleteAll": "Delete all",
      "stats.select": "Select this test",
      "stats.selectAll": "Select all tests",
      "stats.confirmDelete": "Delete this test?",
      "stats.confirmDeleteSelected": "Delete the selected tests?",
      "stats.confirmDeleteAll": "Delete all tests?",
      "stats.confirmUndone": "This cannot be undone.",
      "stats.confirmDeleteAllText": "This deletes every stored test, not only the ones shown here. It cannot be undone.",
      "stats.cancel": "Cancel",
      "stats.deleted": "Deleted tests:",
      "stats.deleteFailed": "Could not delete the tests",
    },

    uk: {
      "lang.label": "Мова",
      "nav.stability": "Тест стабільності",
      "footer.source": "Вихідний код",
      "unit.ms": "мс",
      "unit.mbps": "Мбіт/с",
      "btn.start": "Почати",
      "btn.stop": "Зупинити",
      "btn.reset": "Скинути",
      "btn.loading": "Завантаження…",
      "btn.noServer": "Немає доступного сервера",
      "btn.close": "Закрити",
      "field.server": "Сервер",
      "server.findingBest": "Шукаємо найкращий сервер…",
      "status.ready": "Готово",
      "status.stopped": "Зупинено",
      "aria.testSettings": "Налаштування тесту",
      "aria.results": "Результати",
      "aria.progress": "Прогрес тесту",

      "index.title": "LibreSpeed — вільний тест швидкості з відкритим кодом",
      "index.tagline": "Без Flash, без Java, без WebSocket, без дурниць.",
      "index.chooseServer": "Виберіть сервер",
      "index.localServer": "Локальний сервер",
      "index.sponsoredBy": "Спонсор: ",
      "index.share": "Поділитися",
      "index.status.noServer": "Жоден сервер не відповідає",
      "index.status.noServerText": "Жоден із серверів не відповів. Виберіть сервер зі списку, щоб усе одно спробувати.",
      "index.status.loading": "Завантаження",
      "index.status.download": "Тестуємо завантаження…",
      "index.status.ping": "Вимірюємо пінг…",
      "index.status.upload": "Тестуємо вивантаження…",
      "index.status.starting": "Запуск…",
      "index.status.running": "Не закривайте цю вкладку до завершення тесту.",
      "index.status.complete": "Тест завершено",
      "index.status.none": "Не вдалося нічого виміряти.",
      "index.status.stoppedText": "Тест зупинено до завершення.",
      "index.status.readyText": "Натисніть «Почати», щоб виміряти завантаження, вивантаження і затримку.",
      "index.summary.download": "завантаження {v} Мбіт/с",
      "index.summary.upload": "вивантаження {v} Мбіт/с",
      "index.summary.latency": "Затримка: {v}.",
      "index.summary.idle": "у простої {v} мс",
      "index.summary.downloading": "під час завантаження {v} мс",
      "index.summary.uploading": "під час вивантаження {v} мс",
      "index.connection": "Ваше з’єднання",
      "index.connectionEmpty": "З’явиться після початку тесту",
      "index.privacyNote": "Результати зберігаються після завершення тесту. Починаючи тест, ви погоджуєтеся з",
      "index.privacyLink": "політикою конфіденційності",
      "index.download": "Завантаження",
      "index.upload": "Вивантаження",
      "index.ping": "Пінг",
      "index.jitter": "Джитер",
      "index.idleLatency": "Затримка в простої",
      "index.idleJitter": "Джитер у простої",
      "index.loaded": "Затримка",
      "index.loaded.jitter": "джитер",
      "index.chart.title": "Швидкість",
      "index.chart.note": "Мегабіти за секунду протягом тесту",
      "index.chart.aria": "Швидкість завантаження й вивантаження протягом тесту",
      "index.aria.chart": "Графік швидкості",
      "index.share.title": "Поділитися результатами",
      "index.share.alt": "Результати тесту у вигляді зображення",
      "index.share.url": "Посилання на зображення з результатами",
      "index.share.copy": "Копіювати посилання",
      "index.share.copied": "Скопійовано!",
      "index.share.manual": "Натисніть Ctrl+C",
      "privacy.title": "Політика конфіденційності",
      "privacy.intro": "На цьому сервері тесту швидкості HTML5 увімкнено телеметрію.",
      "privacy.collect": "Які дані ми збираємо",
      "privacy.collectIntro": "Після завершення тесту збираються й зберігаються такі дані:",
      "privacy.item.id": "Ідентифікатор тесту",
      "privacy.item.time": "Час тестування",
      "privacy.item.results": "Результати тесту (швидкість завантаження й вивантаження, пінг і джитер)",
      "privacy.item.ip": "IP-адреса",
      "privacy.item.isp": "Відомості про інтернет-провайдера",
      "privacy.item.location": "Приблизне місцезнаходження (визначене за IP-адресою, не за GPS)",
      "privacy.item.agent": "User agent і мова браузера",
      "privacy.item.log": "Журнал тесту (не містить персональних даних)",
      "privacy.use": "Як ми використовуємо дані",
      "privacy.useIntro": "Дані, зібрані цим сервісом, використовуються, щоб:",
      "privacy.use.share": "Дозволити ділитися результатами тесту (зображення для форумів тощо)",
      "privacy.use.improve": "Покращувати сервіс для вас (наприклад, виявляти проблеми на нашому боці)",
      "privacy.thirdParties": "Персональні дані не передаються третім особам.",
      "privacy.consent": "Ваша згода",
      "privacy.consentText": "Починаючи тест, ви погоджуєтеся з умовами цієї політики конфіденційності.",
      "privacy.removal": "Видалення даних",
      "privacy.removalText":
        "Щоб видалити ваші дані, надайте ідентифікатор тесту або вашу IP-адресу. Лише так можна знайти ваші дані; без цієї інформації ми не зможемо виконати ваш запит.",
      "privacy.contact": "З усіма запитами на видалення звертайтеся на цю адресу:",

      "stab.title": "LibreSpeed — тест стабільності",
      "stab.heading": "Тест стабільності",
      "stab.subtitle": "Вимірювання затримки, джитера й невдалих запитів протягом часу, у простої чи під навантаженням.",
      "stab.duration": "Тривалість",
      "stab.duration.60": "1 хвилина",
      "stab.duration.90": "90 секунд",
      "stab.duration.120": "2 хвилини",
      "stab.duration.180": "3 хвилини",
      "stab.duration.300": "5 хвилин",
      "stab.load": "Навантаження",
      "stab.load.none": "Простій",
      "stab.load.updown": "Завантаження й вивантаження",
      "stab.load.title": "Затримка під навантаженням",
      "stab.load.note": "Простій, потім завантаження, потім вивантаження, по третині тесту.",
      "stab.phase": "Етап",
      "stab.phase.idle": "Простій",
      "stab.phase.download": "Завантаження",
      "stab.phase.upload": "Вивантаження",
      "stab.speed": "Швидкість",
      "stab.range": "Мін.–макс.",
      "stab.loadSummary": "Порівняно з затримкою в простої, під час завантаження вона змінюється на {dl} мс, а під час вивантаження — на {ul} мс.",
      "stab.loadSummary.download": "Порівняно з затримкою в простої, під час завантаження вона змінюється на {dl} мс.",
      "stab.target": "Ціль пінгу",
      "stab.thisServer": "Цей сервер",
      "stab.findingServer": "Шукаємо сервер…",
      "stab.readyText": "Натисніть «Почати», щоб виміряти стабільність.",
      "stab.measuring": "Вимірювання…",
      "stab.measuringText": "Очікуємо першу відповідь.",
      "stab.stoppedText": "Жодної відповіді не отримано.",
      "stab.noResponses": "Немає відповідей",
      "stab.noResponsesText": "Усі запити досі невдалі.",
      "stab.rating.great": "Відмінна стабільність",
      "stab.rating.good": "Добра стабільність",
      "stab.rating.poor": "Слабка стабільність",
      "stab.rating.bad": "Погана стабільність",
      "stab.summary": "Середня {avg} мс, джитер {jitter} мс, {loss}% невдалих запитів.",
      "stab.current": "Поточна",
      "stab.average": "Середня",
      "stab.jitter": "Джитер",
      "stab.min": "Мін.",
      "stab.max": "Макс.",
      "stab.failed": "Невдалі",
      "stab.requests": "Запити",
      "stab.threshold": "Поріг сповіщення",
      "stab.thresholdOff": "Вимк.",
      "stab.thresholdNote": "Звуковий сигнал, коли затримка перевищує це значення.",
      "stab.latency": "Затримка",
      "stab.chart.note": "Мілісекунди, останні 60 секунд",
      "stab.chart.aria": "Затримка за останні 60 секунд",
      "stab.key.failed": "Невдалий запит",
      "stab.csv": "Завантажити CSV",
      "stab.csvTitle": "Завантажити всі вимірювання у CSV",
      "stab.tooltipFailed": "Невдалий",
      "stab.aria.chart": "Графік затримки",
      "stab.aria.alert": "Сповіщення",

      // statistics page
      "stats.title": "LibreSpeed - Статистика",
      "stats.heading": "Статистика",
      "stats.subtitle": "Результати тестів швидкості на цьому сервері.",
      "stats.signIn": "Увійти",
      "stats.signOut": "Вийти",
      "stats.signInNote": "Введіть пароль статистики, щоб побачити результати.",
      "stats.password": "Пароль",
      "stats.wrongPassword": "Неправильний пароль",
      "stats.wrongPasswordText": "Перевірте пароль і спробуйте ще раз.",
      "stats.locked": "Статистику заблоковано",
      "stats.lockedText": "Щоб відкрити доступ, задайте пароль у",
      "stats.disabled": "Статистику вимкнено",
      "stats.disabledText": "Результати не зберігаються, бо тип бази даних —",
      "stats.search": "Знайти тест за ID",
      "stats.idPlaceholder": "ID тесту",
      "stats.find": "Знайти",
      "stats.allTests": "Усі тести",
      "stats.notFound": "Тест не знайдено",
      "stats.notFoundText": "Немає тесту з таким ID:",
      "stats.loadFailed": "Не вдалося завантажити тести",
      "stats.loadFailedText": "База даних повернула помилку. Подробиці — в журналі сервера.",
      "stats.latest": "Останні тести",
      "stats.latestNote": "До 100, спершу найновіші. Відкрийте тест, щоб побачити всі подробиці.",
      "stats.empty": "Ще не збережено жодного тесту.",
      "stats.time": "Час",
      "stats.testId": "ID тесту",
      "stats.col.download": "Завантаження, Мбіт/с",
      "stats.col.upload": "Вивантаження, Мбіт/с",
      "stats.col.ping": "Пінг, мс",
      "stats.col.jitter": "Джитер, мс",
      "stats.result": "Результат тесту",
      "stats.client": "З'єднання",
      "stats.ip": "IP-адреса",
      "stats.isp": "Провайдер",
      "stats.location": "Розташування",
      "stats.lookup": "Дані провайдера",
      "stats.agent": "Браузер",
      "stats.languages": "Мови",
      "stats.latency": "Затримка",
      "stats.latencyNote": "Середня в простої, під час завантаження і вивантаження",
      "stats.latencyJitter": "джитер",
      "stats.latencyRange": "мін.–макс.",
      "stats.extra": "Додаткові дані",
      "stats.extraText": "Текст",
      "stats.image": "Зображення результату",
      "stats.openImage": "Відкрити",
      "stats.imageAlt": "Зображення з результатом цього тесту",
      "stats.log": "Журнал тесту",
      "stats.logNote": "Секунди від початку тесту",
      "stats.rawLog": "Показати необроблений журнал",
      "stats.noLog": "Для цього тесту журнал не збережено.",
      "stats.delete": "Видалити",
      "stats.deleteSelected": "Видалити вибрані",
      "stats.deleteAll": "Видалити всі",
      "stats.select": "Вибрати цей тест",
      "stats.selectAll": "Вибрати всі тести",
      "stats.confirmDelete": "Видалити цей тест?",
      "stats.confirmDeleteSelected": "Видалити вибрані тести?",
      "stats.confirmDeleteAll": "Видалити всі тести?",
      "stats.confirmUndone": "Цю дію не можна скасувати.",
      "stats.confirmDeleteAllText": "Буде видалено всі збережені тести, а не лише показані тут. Цю дію не можна скасувати.",
      "stats.cancel": "Скасувати",
      "stats.deleted": "Видалено тестів:",
      "stats.deleteFailed": "Не вдалося видалити тести",
    },
  };

  var STORAGE_KEY = "librespeed-lang";
  var listeners = [];

  function detect() {
    try {
      var saved = localStorage.getItem(STORAGE_KEY);
      if (saved && STRINGS[saved]) return saved;
    } catch (e) {}
    var preferred = navigator.languages || [navigator.language || ""];
    for (var i = 0; i < preferred.length; i++) {
      var code = String(preferred[i]).toLowerCase().split("-")[0];
      if (STRINGS[code]) return code;
    }
    return "en";
  }

  var lang = detect();

  // look up a string, falling back to English, and fill in {name} placeholders
  function t(key, params) {
    var text = STRINGS[lang][key];
    if (text === undefined) text = STRINGS.en[key];
    if (text === undefined) return key;
    if (params) {
      text = text.replace(/\{(\w+)\}/g, function (match, name) {
        return params[name] !== undefined ? params[name] : match;
      });
    }
    return text;
  }

  // format a number with a fixed number of decimals in the current language
  function num(value, digits) {
    return Number(value).toLocaleString(lang, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
      useGrouping: false,
    });
  }

  function apply() {
    document.documentElement.lang = lang;
    var nodes = document.querySelectorAll("[data-i18n]");
    for (var i = 0; i < nodes.length; i++) {
      nodes[i].textContent = t(nodes[i].getAttribute("data-i18n"));
    }
    nodes = document.querySelectorAll("[data-i18n-attr]");
    for (var j = 0; j < nodes.length; j++) {
      var pairs = nodes[j].getAttribute("data-i18n-attr").split(";");
      for (var k = 0; k < pairs.length; k++) {
        var pair = pairs[k].split(":");
        nodes[j].setAttribute(pair[0].trim(), t(pair[1].trim()));
      }
    }
    var toggles = document.querySelectorAll("[data-lang]");
    for (var m = 0; m < toggles.length; m++) {
      toggles[m].setAttribute("aria-pressed", toggles[m].getAttribute("data-lang") === lang ? "true" : "false");
    }
  }

  function setLang(code) {
    if (!STRINGS[code] || code === lang) return;
    lang = code;
    try {
      localStorage.setItem(STORAGE_KEY, code);
    } catch (e) {}
    apply();
    for (var i = 0; i < listeners.length; i++) listeners[i](lang);
  }

  window.I18N = {
    t: t,
    num: num,
    setLang: setLang,
    lang: function () {
      return lang;
    },
    // call fn after the language changes, to redo text that scripts set
    onChange: function (fn) {
      listeners.push(fn);
    },
  };

  document.addEventListener("DOMContentLoaded", function () {
    apply();
    var toggles = document.querySelectorAll("[data-lang]");
    for (var i = 0; i < toggles.length; i++) {
      toggles[i].addEventListener("click", function () {
        setLang(this.getAttribute("data-lang"));
      });
    }
  });
})();
