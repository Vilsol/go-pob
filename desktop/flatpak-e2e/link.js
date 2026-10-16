(() => {
  if (window.__linkStarted) return;
  window.__linkStarted = true;
  const say = (m) =>
    fetch("/desktop/probe", { method: "POST", body: JSON.stringify(m) }).catch(
      () => {},
    );
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const until = async (what, pred, timeout) => {
    const end = performance.now() + timeout;
    for (;;) {
      const v = await pred();
      if (v) return v;
      if (performance.now() > end)
        throw new Error(`timed out waiting for ${what}`);
      await sleep(50);
    }
  };
  (async () => {
    try {
      await until(
        "pob:data-loaded",
        () => performance.getEntriesByName("pob:data-loaded")[0],
        120000,
      );
      const update = await (await fetch("/desktop/update/check")).json();
      if (update.enabled || update.manager !== "Flathub")
        throw new Error(
          `Check for update in a Flatpak: ${JSON.stringify(update)}, want {enabled: false, manager: "Flathub"}`,
        );
      await say({
        step: "ready for the link; Check for update defers to Flathub",
        update,
      });
      const title = await until(
        "the linked build",
        () => /^Imported build \(/.test(document.title) && document.title,
        60000,
      );
      await until(
        "its stats",
        () => /Life/.test(document.body.innerText),
        60000,
      );
      await say({
        step: "pob:// through the desktop entry → first instance → /api proxy → stats",
        title,
      });
      await say({ done: "link" });
    } catch (e) {
      await say({ fail: String(e), title: document.title });
    }
    fetch("/desktop/probe/quit", { method: "POST" });
  })();
})();
