import type { Preview } from '@storybook/sveltekit';
import { themes } from 'storybook/theming';
import '../src/lib/design/styles/spec.css';

const preview: Preview = {
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: { theme: { ...themes.dark, appContentBg: '#0e0f11', appPreviewBg: '#0e0f11' } },
    a11y: { test: 'error' },
    viewport: {
      options: {
        half: { name: '960×1080 (half a 1920 screen)', styles: { width: '960px', height: '1080px' }, type: 'desktop' },
        min: { name: '1280×720 (minimum)', styles: { width: '1280px', height: '720px' }, type: 'desktop' },
        mid: { name: '1600×900', styles: { width: '1600px', height: '900px' }, type: 'desktop' },
        target: { name: '1920×1080 (target)', styles: { width: '1920px', height: '1080px' }, type: 'desktop' }
      }
    }
  }
};

export default preview;
