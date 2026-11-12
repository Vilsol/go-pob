// Every pass here must leave the rendered pixels unchanged; go run ./scripts/logo proves it in Chromium on each run.
// The disabled preset passes are the ones that moved pixels on this logo.
export default {
  multipass: true,
  js2svg: { indent: 0, pretty: false },
  plugins: [
    {
      name: 'preset-default',
      params: {
        overrides: {
          // the source has two id="tile" elements; renaming them changes which one a url(#tile) finds
          cleanupIds: false,
          convertTransform: false,
          convertShapeToPath: false,
          mergePaths: false,
          // relative coordinates alone move pixels at 2048 px, and absolute ones are larger than the source
          convertPathData: false,
          removeDesc: { removeAny: true }
        }
      }
    },
    'removeDimensions',
    'removeXlink',
    'cleanupListOfValues',
    'removeOffCanvasPaths'
  ]
};
