(() => {
  const root = document.documentElement;
  const storageKey = "speedtest-theme";
  const systemTheme = window.matchMedia("(prefers-color-scheme: dark)");
  const themeColors = document.querySelectorAll("meta[data-theme-color]");
  let currentTheme;
  let toggle;

  function oppositeTheme() {
    return currentTheme === "dark" ? "light" : "dark";
  }

  function renderToggle() {
    if (!toggle) return;
    const next = oppositeTheme();
    const key = next === "light" ? "theme.toLight" : "theme.toDark";
    const translation = window.I18N?.t(key);
    const label = translation && translation !== key ? translation : `Switch to the ${next} theme`;
    toggle.dataset.nextTheme = next;
    toggle.setAttribute("aria-label", label);
    toggle.title = label;
  }

  function applyTheme() {
    const selected = root.dataset.theme;
    for (const meta of themeColors) {
      const theme = meta.dataset.themeColor;
      const media = selected
        ? (selected === theme ? "all" : "not all")
        : `(prefers-color-scheme: ${theme})`;
      meta.setAttribute("media", media);
    }
    const theme = selected || (systemTheme.matches ? "dark" : "light");
    if (theme === currentTheme) return;
    currentTheme = theme;
    renderToggle();
    window.dispatchEvent(new Event("speedtest-theme-change"));
  }

  function selectTheme(theme) {
    if (theme === "light" || theme === "dark") root.dataset.theme = theme;
    else delete root.dataset.theme;
    applyTheme();
  }

  let savedTheme;
  try {
    savedTheme = localStorage.getItem(storageKey);
  } catch (e) {}
  selectTheme(savedTheme);
  systemTheme.addEventListener("change", applyTheme);
  window.addEventListener("storage", (event) => {
    if (event.key === storageKey || event.key === null) selectTheme(event.newValue);
  });

  document.addEventListener("DOMContentLoaded", () => {
    toggle = document.querySelector(".theme-toggle");
    if (!toggle) return;
    toggle.addEventListener("click", () => {
      const theme = oppositeTheme();
      selectTheme(theme);
      try {
        localStorage.setItem(storageKey, theme);
      } catch (e) {}
    });
    window.I18N?.onChange(renderToggle);
    renderToggle();
    toggle.hidden = false;
  });
})();
