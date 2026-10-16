(() => {
  if (window.__portalStarted) return;
  window.__portalStarted = true;
  const cfg = window.__PORTAL;
  const say = (m) =>
    fetch("/desktop/probe", { method: "POST", body: JSON.stringify(m) }).catch(
      () => {},
    );
  const quit = () => fetch("/desktop/probe/quit", { method: "POST" });
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const until = async (what, pred, timeout = 60000) => {
    const end = performance.now() + timeout;
    for (;;) {
      const v = await pred();
      if (v) return v;
      if (performance.now() > end)
        throw new Error(`timed out waiting for ${what}`);
      await sleep(50);
    }
  };
  addEventListener("pob:desktop-toast", (e) => say({ toast: e.detail }));
  const config = async () => (await fetch("/desktop/config")).json();
  const listing = async () =>
    (await (await fetch("/desktop/fs/builds/")).json()).map((e) => e.name);

  const pickFolder = async () => {
    await fetch("/desktop/builds-folder", { method: "POST" });
    await sleep(120000);
    throw new Error("the window did not reload onto the chosen folder");
  };

  const afterFolder = async (c) => {
    const put = await fetch("/desktop/fs/builds/portal-build", {
      method: "PUT",
      body: "<PathOfBuilding></PathOfBuilding>",
    });
    await say({
      step: "Builds folder… through the portal",
      source: c.source,
      dir: c.dir,
      put: put.status,
      builds: await listing(),
    });

    const got = new Promise((r) =>
      addEventListener("pob:desktop-import", r, { once: true }),
    );
    await fetch("/desktop/open", { method: "POST" });
    await got;
    const title = await until(
      "the opened build",
      () => /^Imported build \(/.test(document.title) && document.title,
    );
    await say({ step: "File › Open build file through the portal", title });

    const r = await fetch("/desktop/save?name=portal-save.txt", {
      method: "POST",
      body: "saved through the portal",
    });
    await say({
      step: "save dialog through the portal",
      status: r.status,
      path: await r.text(),
    });
  };

  const second = async (c) => {
    await say({
      step: "relaunch keeps the folder",
      source: c.source,
      dir: c.dir,
      builds: await listing(),
    });
  };

  (async () => {
    try {
      await until(
        "pob:data-loaded",
        () => performance.getEntriesByName("pob:data-loaded")[0],
        120000,
      );
      const c = await config();
      if (cfg.phase === "second") await second(c);
      else if (c.source === "setting") await afterFolder(c);
      else return await pickFolder();
      await say({ done: cfg.phase });
    } catch (e) {
      await say({ fail: String(e), title: document.title });
    }
    quit();
  })();
})();
