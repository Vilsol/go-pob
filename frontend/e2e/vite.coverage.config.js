// vite.config.js with every frontend/src function marked when it runs; see coverage.mjs
import base from '../vite.config.js';
import { coveragePlugin } from './coverage.mjs';

export default {
  ...base,
  plugins: [...base.plugins, coveragePlugin()],
  worker: { plugins: () => [coveragePlugin()] }
};
