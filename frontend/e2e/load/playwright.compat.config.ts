import { defineConfig, devices } from '@playwright/test';
import base from '../playwright.config';

export default defineConfig({
  ...base,
  testDir: '.',
  testMatch: '*.compat.ts',
  globalSetup: undefined,
  globalTeardown: undefined,
  webServer: undefined,
  outputDir: '../.results/compat-output',
  projects: [
    { name: 'chromium', use: { ...base.use } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'], browserName: 'firefox', launchOptions: {} } },
    { name: 'webkit', use: { ...devices['Desktop Safari'], browserName: 'webkit', launchOptions: {} } }
  ]
});
