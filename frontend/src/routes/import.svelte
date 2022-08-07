<script lang="ts">
  import Input from '../lib/components/Input.svelte';
  import { syncWrap } from '../lib/go/worker';

  // TODO Remove from prod
  const sampleBuildCode = `eJzsW21v2zgS_pz-CkFfD41jp0nchd2FYzuJcXHXF6XtHQ6HgpbGNjcUqZKUU--vP5DUqyNZtLJ3V-zeYrGwh_PMcB4OyRk6O_j5e0icLXCBGR263dMz1wHqswDT9dD99Hjztu_-_OHNYIHk5pfVdYyJGvnw5mSgPzs-QUJ8RCEMXc_HjLqORHwN8nNq8fzrmessEQ2wHLofGQXXiRCVG2B0jn5l_JYFL-SYluQEtkCGbtd1QoSpx_wnkLecxZGWbTE8z1mgJvDX2f295zpI-ECDcT4zbebDm5OTwYKgHXBPIulsEYlBRdzvX1x13_X0f89dR0gkh-5oCxytYYJCtAa3UwvupQAvAghqFc9StQWH6WoFvsRbGHMsxxtE_XoHGc5Ct3t6UdSex0TiiGDgtYj371PAXaP13mnvspczlQIfmURksvCaIzCaTB7r4QuWm2sCEBzycgA7W1MsoSV4wbBg9Kj4rJTHMSGYrlvMaszCJaaH2cjczBFFYybqOc807_EK7DSnnp3eA1pbWlSzXAD3gUr7yR4FSDx44DMaHOXjGEghjnbOWiCnXhuAvaNeBvRk_UGSa03ge_0B1c2y-AG-HdLM7c1o_dy6FxcFe4c0C9PbMqmuqTrNy_PSfp7eLepV80N9sxPYR2SOvuMwDu-wfERPUO_k_Co_qu_xeiMppus24BvMoQ1uzEjQCrdBTNgCL0vp3bw2XgT-T0p1Rn1Lo58oBwF8e-DmLQMewGfqZl-SA_foWY2PZOPYbecHWB_gpqR6D-BvbhGmD0jWT-uidM5Y0qlUD9FZNmpBZ4GcMqKRnNP3RaAlPUrVjp4M8gVxizpsSoGvd94GAzlSO0mh3RhFlotQRB9ajBp3VkQVIXaEdS8z7BaJQ2fiXjRG2yqQORCA6RYFYFvnLjj7VVXH5DjYiIcsrr-fyhEYZasA0pPd9AEPEMT-wesjA14T5j_Zzt6LgJCjECMpkf80YcHamiLt5HiEF0cRB6GWvAH3Nj9q1b30AALbFCS57i9b4Id2VcGBusBsHeS6RzjIrmVbL3uAY2JRl6p1MLlyk4us2MoazjnbQghU6l51ziwOvpuY2HV3C_YMfLxRvb-oLyWqtOeovg7M58GB_raztl9St3IwpUHMVXpb-9hH5G4ecQgEhJggiRwBiPubeyzk0NXjg45-OVGfZmHEuNTCMaMrvNbwGY1i6QjJ9ROMeQKh-h3DPKMkXurV9t9X7AHJw0sevA8bRgLgDo3DJfChe5FigEK4q3qVeAE5P3uB2X9WeYG5OiuD8teVQSejavDIARykU1sd7CrltUX1pfgKNAv0SlMWgBi6F_3--bnrhEhI4DuzOcTQdR3JAQrvVt2-a563DFxZPhl8erjXH042Ukbip07n-fn5NEJyw1bwHRM49VnYiZAQeAtvxRMm5K0y2xmNRqPr9Uj_ow11UksD86AlDBkdNXcdpgpOZwYivigsYEpRNyVIe_lqxJVLPb25mY4fZ5-nKSLEwv-6jFerOQvSVfBA32wODoaupyx6QMCXriPipTBjQ_czhmc9OAGJMBGu4zNCUCQgGLorRESVtbtsvUu2tJ3CmIWlL4j7HIMoG5p-By4xXeejFqYmTFZMx8xFlXdqD1sZMplZMmUu5zESUhdgNlbUpigbURK7UGZhhMgeuanMJgAV-OMuAnXKiQpS5C4CEYGPV9hPdOxWPTkZytz4fsyRv7OyoZ_9yvhEZAE2T3dldCqzYVU_G-6xmshs8gt8tCujE5EFOCvgywYyMaN3Kl0sLE0JjDBRd_7eyn5kVCc7putcwcLgHAs_OTLLBn-RG-DZiIWlkZQcL2O5v52Lchuu8GpvlYzEJhbVR5egRmLDa7F9LJ9GpRG7Q23PhJHYUGi6njJ9icwmiKTvK88_FVoYSIrWEt7I7NZOtzz7-1uL2oHN3nhpQvK4LpcnsALq7-egTuZ8yHJfVNsyd8poy3BgjsNXWdMzqzbWLspk91cGe7RF3V6-MsL9rtPuuNUkm_fICvaTgVcbwnR9Z3st11rK2rE7QERuFoyR1xncf5B9XZxMCkSDCayB1gU66KRF6eAjk6CrUyVNvwx03SCcAFYoJvIWwvvCT7jS36SSR6aaJ-RL4ImGceKIDXseEfm3GBEsd7cQimwoN5qM6to-6QJM4SpN88u40hLXu8nCS5K5LLxRh3T5JzXtWWWf0VN1kRi6o_v7pLNIHGgOu0lPoIUOQUsVwDSM5M78Uj3K5zR0KSauAxQtSbaz9pU0qYkmpj6JA5jRpAM34k6Fw392_-XcYA5LREiF366l126VT0O5aXgGtxAmpm4JWyLSSw0Whd1UqDoN05Tlk9OdygJxsz4-i6msmqBWm2OKGZ1JCNP1zP8M4JtZ-Jmus3Q2FGzn8RTsDF0vDkNGIVhsEJVIhMmwspFPcA2hEsxBogBJ1FHuRUflS0dzdQthrvwtT79kXYxS1Rr1zv5oi5SsRu_swHL8GNwX9ofzF0d9HQUBBM6YkcBJu88_0JIc2iD_xRVpH3jF8rwmZiVJTnRtWRlO7R4KMbsEXqDa7vw_Rfr9KCfC75p___Gba5-YmQ-3jED4Q-aw-v7_JP5zJvFRzPxPszj56IHULYvpSdKfDT5jeE5-m_EkH7pJ4S-RnODVCrhuTFNGfmMs7WD6RvEOkJyjKGtKlMbfh2632z_tvT_r9_u9q-6lEf9j6F5dnr7rnr-7enem_zU9lA4w6VuKPMcCzN9lfQEUMarF-V4YJKpJA9KgfTLwCJPJq_81Y1I4o-VOCEQc86uD8851sDTUZxQ2YS4OY8xMnK73jKJ9aLfBHRD5O0B6x0POj4c0UVcBeQVzDd4OQRu8XrNg56TPhm19XNpBj1zcOljDAtfBGha5DtZA_R2QEGQ703a09VpspUNQO_oqoXYUVkLtMrgS2pDBozAmlivQa0ljSwpb0teSunbHzQtYQ1KafD-Sx2pQA4vVoAYOq0FWe_hI_qpBDewVTttjb7p6ZNOFV49suvfqkY2VQy3S_j46kttbwrbwolxpoLUa1MBoNaiBzGpQA4_VoAYKq0GNmVlR6zXmZAWmMRsrMJZXs92BdljrAdN1kyWt02DnhiDxZKfUEJ1RakgEo2RZQTrqFrU8_ZtVjesWhXhTxgGxrJvaVzEvSq7G04dJ0b5SabPHLGvnFuWYOQde6Aw6SRepG2PdiH54M-js_z-e_w4AAP__S2Sp7g==`;

  // TODO Set to empty in prod
  let importCode = sampleBuildCode;

  const importBuildFromCode = () => {
    syncWrap
      .ImportCode(importCode)
      .catch((err) => {
        // TODO Error notification
        console.error(err);
      })
      .then(() => {
        syncWrap.Tick();
      });
  };
</script>

<div class="p-2 px-4 min-w-[650px]">
  <fieldset class="border border-white bg-neutral-900 p-2">
    <legend class="container">Character Import</legend>
    <div class="flex flex-col gap-3">
      <span>Character import status: Idle</span>

      <div class="flex flex-col gap-1">
        <span>To start importing a character, enter the character's account name:</span>

        <div class="flex flex-row gap-1">
          <div class="container select-wrapper">
            <select class="input">
              <option>PC</option>
              <option>Xbox</option>
              <option>PS4</option>
              <option>Garena</option>
              <option>Tencent</option>
            </select>
          </div>

          <Input />

          <button class="container">Start</button>

          <div class="flex flex-row gap-2 flex-1">
            <div class="container select-wrapper min-w-full">
              <select class="input" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </fieldset>

  <fieldset class="border border-white bg-neutral-900 p-2 mt-4">
    <legend class="container">Build Sharing</legend>
    <div class="flex flex-col gap-3">
      <div class="flex flex-col gap-1">
        <div class="flex flex-row gap-2 items-center">
          <span>Generate a code to share this build with other Path of Building users:</span>
          <button class="container">Generate</button>
        </div>

        <div class="flex flex-row gap-2 items-center">
          <Input placeholder="Code" disabled={true} />

          <button class="container" disabled>Copy</button>

          <div class="container select-wrapper disabled">
            <select class="input" disabled>
              <option>Pastebin.com</option>
              <option>PoeNinja</option>
              <option>pobb.in</option>
            </select>
          </div>

          <button class="container" disabled>Share</button>
        </div>

        <span>Note: this code can be very long; you can 'Share' to shrink it.</span>
      </div>

      <div class="flex flex-col gap-1">
        <span>To import a build, enter URL or code here:</span>

        <Input fullWidth={true} bind:value={importCode} />

        <div class="flex flex-row gap-1">
          <div class="container select-wrapper disabled">
            <select class="input" disabled>
              <option>Import to a new build</option>
            </select>
          </div>

          <button class="container" disabled={importCode === ''} on:click={importBuildFromCode}>Import</button>
        </div>
      </div>
    </div>
  </fieldset>
</div>
