(() => {
  if (window.__e2eStarted) return;
  window.__e2eStarted = true;
  const cfg = window.__E2E;
  const say = (m) => fetch('/desktop/probe', { method: 'POST', body: JSON.stringify(m) }).catch(() => {});
  const quit = () => fetch('/desktop/probe/quit', { method: 'POST' });
  const errors = [];
  addEventListener('error', (e) => errors.push(String(e.message)));
  addEventListener('unhandledrejection', (e) => errors.push(String(e.reason)));
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
  const until = async (what, pred, timeout = 60000) => {
    const end = performance.now() + timeout;
    for (;;) {
      const v = await pred();
      if (v) return v;
      if (performance.now() > end) throw new Error(`timed out waiting for ${what}`);
      await sleep(25);
    }
  };
  const mark = (n) => performance.getEntriesByName(n)[0]?.startTime;
  const step = async (name, fn) => {
    const t = performance.now();
    const detail = await fn();
    await say({ step: name, ms: Math.round(performance.now() - t), ...(detail ?? {}) });
  };
  const buttonByText = (root, text) => [...root.querySelectorAll('button')].find((b) => b.textContent.trim() === text);
  const listing = async () => (await (await fetch('/desktop/fs/builds/')).json()).map((e) => e.name);

  const importAndSave = async (name) => {
    let opened;
    const got = new Promise((r) => addEventListener('pob:desktop-import', r, { once: true }));
    await step(`import ${name} (open file → stats)`, async () => {
      await fetch('/desktop/open', { method: 'POST' });
      opened = await got;
      const title = await until('the imported build', () => /^Imported build \(/.test(document.title) && document.title);
      await until('its stats', () => /Life/.test(document.body.innerText));
      return { title };
    });
    await step(`save ${name} to the builds folder`, async () => {
      const save = await until('the Save button', () => document.querySelector('button[aria-label^="Save ("]'));
      save.click();
      const dialog = await until('Save As', () => [...document.querySelectorAll('dialog[open]')].find((d) => /Build name/.test(d.textContent)));
      const input = dialog.querySelector('input');
      input.value = name;
      input.dispatchEvent(new Event('input', { bubbles: true }));
      await sleep(50);
      buttonByText(dialog, 'Save').click();
      await until(`${name} on disk`, async () => (await listing()).includes(name));
      await until('the saved title', () => document.title.startsWith(name));
      return { opened: String(opened.detail).slice(0, 24) };
    });
  };

  const hintText = /To use Path of Building's Builds folder, choose it with File › Builds folder/;
  const config = async () => {
    await step('config', async () => {
      const config = await (await fetch('/desktop/config')).json();
      if (config.source !== cfg.source) throw new Error(`builds folder source ${config.source}, want ${cfg.source}`);
      return { config };
    });
    if (cfg.hint)
      await step('the Builds folder hint, dismissed', async () => {
        const toast = await until('the hint', () => [...document.querySelectorAll('[role=status]')].find((t) => hintText.test(t.textContent)));
        buttonByText(toast, 'Dismiss').click();
        await until('the hint gone', () => !hintText.test(document.body.innerText));
        return { hint: toast.innerText.replace(/\s*Dismiss$/, '') };
      });
    else if (hintText.test(document.body.innerText)) throw new Error('the Flatpak hint showed');
  };

  const first = async () => {
    await config();
    await step('New build → tree', async () => {
      buttonByText(document, 'New build').click();
      const title = await until('the new build', () => /^Unnamed build/.test(document.title) && document.title);
      await until('its stats', () => /Life/.test(document.body.innerText));
      return { title, path: location.pathname };
    });
    for (const name of cfg.builds) await importAndSave(name);
    await step('builds folder listing', async () => ({ builds: await listing() }));
    if (cfg.link)
      await step(`import ${cfg.link} (deep link → /api proxy → stats)`, async () => {
        dispatchEvent(new CustomEvent('pob:desktop-import', { detail: cfg.link }));
        const title = await until('the linked build', () => /^Imported build \(/.test(document.title) && document.title);
        await until('its stats', () => /Life/.test(document.body.innerText));
        return { title };
      });
    // Build.lua:1421-1448
    if (cfg.link) {
      const askOpen = async () => {
        const got = new Promise((r) => addEventListener('pob:desktop-import', r, { once: true }));
        await fetch('/desktop/open', { method: 'POST' });
        await got;
        return until('Save Changes', () => [...document.querySelectorAll('dialog[open]')].find((d) => /This build has unsaved changes/.test(d.textContent)));
      };
      await step('open a file over unsaved changes → Save Changes, Cancel keeps the build', async () => {
        await until('the unsaved state', () => document.querySelector('button[aria-label="Save (unsaved changes)"]'));
        const before = document.body.innerText;
        const prompt = await askOpen();
        buttonByText(prompt, 'Cancel').click();
        await until('the prompt closed', () => !prompt.open);
        await sleep(2000);
        if (document.body.innerText !== before) throw new Error('Cancel replaced the build');
        if (!document.querySelector('button[aria-label="Save (unsaved changes)"]')) throw new Error('Cancel dropped the unsaved state');
        return { title: document.title };
      });
      await step("open it again → Save Changes, Don't Save opens the file", async () => {
        const before = document.title;
        buttonByText(await askOpen(), "Don't Save").click();
        await until('the opened file', () => document.title !== before && /^Imported build \(/.test(document.title));
        await until('its stats', () => /Life/.test(document.body.innerText));
        return { title: document.title };
      });
    }
    if (cfg.folder) {
      await step("Builds folder… over unsaved changes → Save Changes, Don't Save", async () => {
        await until('the unsaved state', () => document.querySelector('button[aria-label="Save (unsaved changes)"]'));
        dispatchEvent(new CustomEvent('pob:desktop-builds-folder', { detail: '' }));
        const prompt = await until('Save Changes', () =>
          [...document.querySelectorAll('dialog[open]')].find((d) => /This build has unsaved changes/.test(d.textContent))
        );
        sessionStorage.setItem('e2eResume', 'folder');
        buttonByText(prompt, "Don't Save").click();
      });
      await sleep(60000);
      throw new Error('the window did not reload onto the chosen folder');
    }
  };

  const resumeFolder = async () => {
    await step('reloaded onto the chosen folder', async () => {
      const config = await (await fetch('/desktop/config')).json();
      if (config.source !== 'setting') throw new Error(`builds folder source ${config.source}, want setting`);
      await until('the list', () => document.querySelector(`[role=row][data-build="${cfg.builds[0]}"]`));
      return { dir: config.dir, builds: await listing() };
    });
  };

  const second = async () => {
    await config();
    await step('builds folder listing', async () => ({ builds: await listing() }));
    await step('Settings › Export all builds (save dialog)', async () => {
      document.querySelector('button[aria-label="Settings"]').click();
      (await until('Data and storage', () => [...document.querySelectorAll('[role=tab]')].find((t) => t.textContent.startsWith('Data and storage')))).click();
      (await until('Export .zip', () => buttonByText(document, 'Export .zip'))).click();
      const toast = await until('the export toast', () => /Exported \d+ builds?\s+\S+\.zip/.exec(document.body.innerText)?.[0]);
      buttonByText(document, 'Done')?.click();
      return { toast };
    });
    await step(`reopen ${cfg.builds[0]} from the list`, async () => {
      const row = await until('its row', () => document.querySelector(`[role=row][data-build="${cfg.builds[0]}"]`));
      row.dispatchEvent(new MouseEvent('dblclick', { bubbles: true }));
      const title = await until('the reopened build', () => document.title.startsWith(cfg.builds[0]) && document.title);
      await until('its stats', () => /Life/.test(document.body.innerText));
      return { title };
    });
    // Build.lua:89-100
    await step('the reopened build is saved; Back closes it without asking', async () => {
      await until('the saved state', () => document.querySelector('button[aria-label="Save (no unsaved changes)"]'));
      document.querySelector('a[aria-label="Back to builds"]').click();
      await until('the list', () => document.querySelector(`[role=row][data-build="${cfg.builds[0]}"]`));
    });
    await step('open a file with no build open → unsaved; another file → Save Changes', async () => {
      const opened = () => {
        const got = new Promise((r) => addEventListener('pob:desktop-import', r, { once: true }));
        return fetch('/desktop/open', { method: 'POST' }).then(() => got);
      };
      await opened();
      await until('the imported build', () => /^Imported build \(/.test(document.title));
      await until('its stats', () => /Life/.test(document.body.innerText));
      await until('the unsaved state', () => document.querySelector('button[aria-label="Save (unsaved changes)"]'));
      const before = document.title;
      await opened();
      const prompt = await until('Save Changes', () =>
        [...document.querySelectorAll('dialog[open]')].find((d) => /This build has unsaved changes/.test(d.textContent))
      );
      buttonByText(prompt, "Don't Save").click();
      await until('the prompt closed', () => !prompt.open);
      await until('the other file', () => document.title !== before && /^Imported build \(/.test(document.title));
      return { title: document.title };
    });
  };

  (async () => {
    try {
      await until('pob:data-loaded', () => mark('pob:data-loaded'), 120000);
      await say({ step: 'boot', booted: Math.round(mark('pob:wasm-booted')), dataLoaded: Math.round(mark('pob:data-loaded')) });
      if (sessionStorage.getItem('e2eResume') === 'folder') {
        sessionStorage.removeItem('e2eResume');
        await resumeFolder();
      } else await (cfg.phase === 'first' ? first() : second());
      await say({ done: cfg.phase, errors });
    } catch (e) {
      await say({ fail: String(e), errors, title: document.title, path: location.pathname });
    }
    quit();
  })();
})();
