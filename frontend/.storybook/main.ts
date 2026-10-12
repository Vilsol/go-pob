import type { StorybookConfig } from '@storybook/sveltekit';

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(ts|svelte)'],
  addons: ['@storybook/addon-svelte-csf', '@storybook/addon-docs', '@storybook/addon-a11y', '@storybook/addon-vitest'],
  framework: { name: '@storybook/sveltekit', options: {} },
  core: { disableTelemetry: true }
};

export default config;
