let lastCall = performance.now();

export const noteCall = () => {
  lastCall = performance.now();
};

export const whenIdle = (idleMs: number, run: () => unknown) => {
  const check = () => {
    const quiet = performance.now() - lastCall;
    if (quiet >= idleMs) {
      performance.mark('pob:idle-job-start');
      void Promise.resolve(run()).finally(() => performance.mark('pob:idle-job-end'));
    } else setTimeout(check, idleMs - quiet);
  };
  setTimeout(check, idleMs);
};
