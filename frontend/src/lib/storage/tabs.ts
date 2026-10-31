// zenfs keeps its file table and next inode id per instance, so two tabs writing one IndexedDB database overwrite each
// other's files. The holder of a Web Lock is the only writer; every other tab mounts the databases read-only, forwards
// its writes to the writer over a BroadcastChannel and remounts a bucket the writer says it changed. The lock passes to
// the next waiting tab when the writer's worker ends.

export type StorageWrite =
  | { kind: 'set'; bucket: string; key: string; value: Uint8Array }
  | { kind: 'mkdir' | 'delete'; bucket: string; key: string }
  | { kind: 'clear'; bucket: string };

type Message =
  | { type: 'write'; id: string; from: string; write: StorageWrite }
  | { type: 'ack'; id: string; error?: string }
  | { type: 'changed'; bucket: string; from: string }
  | { type: 'writer' };

export type TabStorageOptions = {
  buckets: string[];
  // performs a write on this tab's mount; only ever called while this tab holds the lock
  apply: (w: StorageWrite) => Promise<void>;
  // replaces a bucket's mount with one that reads the database as it is now
  remount: (bucket: string) => Promise<void>;
  // another tab changed a bucket, and this tab now reads the change
  changed?: (bucket: string) => void;
  // buckets whose changes may wait this many ms, so a burst of writes remounts them once; a set in one does not wait
  // for the writer
  lazy?: Record<string, number>;
  locks?: LockManager;
};

type Pending = { write: StorageWrite; done: (error?: string) => void };

export class TabStorage {
  readonly id = crypto.randomUUID();
  private readonly channel: BroadcastChannel;
  private readonly pending = new Map<string, Pending>();
  private writer = false;
  private applying: Promise<unknown> = Promise.resolve();
  private readonly views = new Map<string, Promise<void>>();
  private readonly queued = new Map<string, Promise<void>>();
  private readonly timers = new Set<string>();
  private release?: () => void;
  private readonly closing = new AbortController();

  constructor(
    private readonly name: string,
    private readonly o: TabStorageOptions
  ) {
    this.channel = new BroadcastChannel(name);
    this.channel.onmessage = (e: MessageEvent<Message>) => void this.receive(e.data);
    // Node keeps a process alive while a channel is open; browsers have no unref
    (this.channel as { unref?: () => void }).unref?.();
  }

  get isWriter() {
    return this.writer;
  }

  // resolves with whether this tab writes; a tab that does not keeps waiting for the lock and takes over when it frees
  start(): Promise<boolean> {
    const locks = this.o.locks ?? (globalThis.navigator as Navigator | undefined)?.locks;
    if (!locks) {
      // no Web Locks outside a secure context: nothing can coordinate, so this tab writes as a single tab does
      this.writer = true;
      return Promise.resolve(true);
    }
    const hold = () => new Promise<void>((r) => (this.release = r));
    return new Promise((started) => {
      void locks.request(this.name, { ifAvailable: true }, (lock) => {
        if (lock) {
          this.writer = true;
          started(true);
          return hold();
        }
        started(false);
        locks.request(this.name, { signal: this.closing.signal }, () => this.promote().then(hold)).catch(() => {});
        return undefined;
      });
    });
  }

  close() {
    this.closing.abort();
    this.release?.();
    this.channel.close();
  }

  async write(w: StorageWrite): Promise<void> {
    if (this.writer) {
      const error = await this.applied(w, this.id);
      if (error) throw new Error(error);
      return;
    }
    const id = crypto.randomUUID();
    const acked = new Promise<string | undefined>((done) => this.pending.set(id, { write: w, done }));
    this.post({ type: 'write', id, from: this.id, write: w });
    if (!this.waits(w)) return;
    const error = await acked;
    if (error) throw new Error(error);
  }

  private waits(w: StorageWrite) {
    return w.kind !== 'set' || this.o.lazy?.[w.bucket] === undefined;
  }

  private post(m: Message) {
    this.channel.postMessage(m);
  }

  // writes run one at a time: two concurrent creates in one directory each rewrite the listing they read
  private applied(w: StorageWrite, from: string): Promise<string | undefined> {
    const run = this.applying.then(async () => {
      try {
        await this.o.apply(w);
      } catch (e) {
        return String(e);
      }
      this.post({ type: 'changed', bucket: w.bucket, from });
      return undefined;
    });
    this.applying = run;
    return run;
  }

  private async promote() {
    await Promise.all(this.o.buckets.map((b) => this.refresh(b)));
    for (const [id, p] of this.pending) {
      this.pending.delete(id);
      p.done(await this.applied(p.write, this.id));
    }
    this.writer = true;
    this.post({ type: 'writer' });
  }

  private refresh(bucket: string): Promise<void> {
    const queued = this.queued.get(bucket);
    if (queued) return queued;
    const next = (this.views.get(bucket) ?? Promise.resolve())
      .then(() => {
        this.queued.delete(bucket);
        return this.o.remount(bucket);
      })
      .catch((e: unknown) => console.error(`remounting ${bucket}`, e));
    this.queued.set(bucket, next);
    this.views.set(bucket, next);
    return next;
  }

  private async receive(m: Message) {
    switch (m.type) {
      case 'write': {
        if (!this.writer) return;
        const error = await this.applied(m.write, m.from);
        this.post({ type: 'ack', id: m.id, error });
        if (!error) this.o.changed?.(m.write.bucket);
        return;
      }
      case 'ack': {
        const p = this.pending.get(m.id);
        if (!p) return;
        this.pending.delete(m.id);
        if (!m.error && this.waits(p.write)) await this.refresh(p.write.bucket);
        p.done(m.error);
        return;
      }
      case 'changed': {
        if (this.writer) return;
        const delay = this.o.lazy?.[m.bucket];
        // a write this tab waited for was remounted on its ack; a lazy one was not
        if (delay === undefined) {
          if (m.from === this.id) return;
          await this.refresh(m.bucket);
          this.o.changed?.(m.bucket);
        } else if (!this.timers.has(m.bucket)) {
          this.timers.add(m.bucket);
          setTimeout(() => {
            this.timers.delete(m.bucket);
            void this.refresh(m.bucket).then(() => this.o.changed?.(m.bucket));
          }, delay);
        }
        return;
      }
      case 'writer':
        if (this.writer) return;
        for (const [id, p] of this.pending) this.post({ type: 'write', id, from: this.id, write: p.write });
        return;
    }
  }
}

// localStorage is shared by every tab, and the browser tells the others of each change; store re-reads key with load
// returns the unsubscribe, for a component that follows a key while mounted
export const followOtherTabs = <T>(key: string, store: { set: (v: T) => void }, load: () => T) => {
  if (typeof window === 'undefined') return () => {};
  const on = (e: StorageEvent) => {
    if (e.key === key || e.key === null) store.set(load());
  };
  window.addEventListener('storage', on);
  return () => window.removeEventListener('storage', on);
};
