import { get } from 'svelte/store';
import { currentBuild, outputs } from '$lib/global.js';
import { syncWrap } from '$lib/go/worker';
import { logError } from '$lib/utils';

export class MainGroup {
  groups = $state<string[]>([]);
  index = $state(-1);
  #skillSet = $state(0);
  #seq = { set: 0, groups: 0, index: 0 };
  #selects = 0;

  constructor() {
    $effect(() =>
      currentBuild.subscribe((b) => {
        const read = ++this.#seq.set;
        b
          ?.ActiveSkillSetIndex()
          .then((i) => {
            if (read === this.#seq.set) this.#skillSet = Math.max(0, i ?? 0);
          })
          .catch(logError);
      })
    );
    $effect(() => {
      const set = this.#skillSet;
      return outputs.subscribe(() => {
        if (!get(currentBuild)) {
          this.groups = [];
          return;
        }
        const read = ++this.#seq.groups;
        syncWrap
          ?.GetSocketGroupSummaries(set)
          .then((rows) => {
            if (read === this.#seq.groups) this.groups = rows.map((r) => r.label);
          })
          .catch(logError);
      });
    });
    $effect(() =>
      currentBuild.subscribe((b) => {
        if (!b) return;
        const read = ++this.#seq.index;
        const selects = this.#selects;
        syncWrap
          ?.GetBuildInfo()
          .then((info) => {
            if (info && read === this.#seq.index && selects === this.#selects) this.index = info.mainSocketGroup - 1;
          })
          .catch(logError);
      })
    );
  }

  select(i: number) {
    this.#selects++;
    this.index = i;
    syncWrap
      ?.SetMainSocketGroup(i + 1)
      .then(() => currentBuild.set(get(currentBuild)))
      .catch(logError);
  }
}
