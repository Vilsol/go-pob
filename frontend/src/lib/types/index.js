/* eslint-disable */
export let builds;
export let cache;
export let calculator;
export let config;
export let raw;

export const initializeCrystalline = () => {
  builds = {
    ParseBuild: globalThis['go']['go-pob']['builds']['ParseBuild']
  };
  cache = {
    InitializeDiskCache: globalThis['go']['go-pob']['cache']['InitializeDiskCache']
  };
  calculator = {
    NewCalculator: globalThis['go']['go-pob']['calculator']['NewCalculator']
  };
  config = {
    InitLogging: globalThis['go']['go-pob']['config']['InitLogging']
  };
  raw = {
    InitializeAll: globalThis['go']['go-pob']['raw']['InitializeAll']
  };
};
