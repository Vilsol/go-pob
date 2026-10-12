import { defineConfig } from '@playwright/test';
import base from '../playwright.config';

export default defineConfig({
  ...base,
  testDir: '.',
  testMatch: '*.probe.ts',
  globalSetup: undefined,
  globalTeardown: undefined,
  webServer: undefined,
  outputDir: '../.results/load-output'
});
