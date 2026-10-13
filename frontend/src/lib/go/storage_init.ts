import { fs } from '@zenfs/core';
import { builds, storage } from '../types';

export const initializeStorage = async () => {
  await storage.InitializeStorage(
    async (bucket: string, dir: string) =>
      new Promise((res, rej) => {
        fs.readdir(`/${bucket}/${dir}`, (err, files) => {
          if (err) {
            console.error(err);
            return rej(err);
          }

          res(
            files?.map((file) => {
              const fullPath = `/${bucket}/${dir}/${file}`;
              const stats = fs.statSync(fullPath);

              let summary: builds.SavedBuild | undefined;
              if (stats.isFile()) {
                const buildFile = fs.readFileSync(fullPath);
                if (buildFile) {
                  const parsed = builds.SavedBuildSummary(buildFile.toString());
                  summary = parsed.ok ? parsed.value : undefined;
                }
              }

              return {
                Name: file,
                Type: stats.isDirectory() ? 'dir' : 'file',
                Class: summary?.ClassName || '',
                LastEdit: stats.mtime.toISOString(),
                Level: summary?.Level || 0,
                Ascendancy: summary?.Ascendancy ?? '',
                MainSkill: summary?.MainSkill ?? ''
              };
            })
          );
        });
      }),
    async (bucket: string, key: string) =>
      new Promise((res, rej) => {
        fs.readFile(`/${bucket}/${key}`, (err, file) => {
          if (err) {
            console.error(err);
            return rej(err);
          }

          if (file) {
            return res(new Uint8Array(file));
          }

          return res(new Uint8Array(0));
        });
      }),
    async (bucket: string, key: string, value: Uint8Array | undefined) => {
      if (!value || !key) {
        return;
      }

      return new Promise((res, rej) => {
        console.log('Writing to storage', `/${bucket}/${key}`);
        fs.writeFile(`/${bucket}/${key}`, value, (err) => {
          if (err) {
            return rej(err);
          }
          res();
        });
      });
    },
    async (bucket: string, key: string) =>
      new Promise((res) => {
        fs.access(`/${bucket}/${key}`, (err) => {
          if (err) {
            if (err.code !== 'ENOENT') {
              console.error(err);
            }
            return res(false);
          }

          return res(true);
        });
      }),
    async (bucket: string, dir: string) =>
      new Promise((res) => {
        fs.mkdir(`/${bucket}/${dir}`, 0o777, (err) => {
          if (err) {
            console.error(err);
            return res();
          }

          return res();
        });
      }),
    async (bucket: string, path: string) =>
      new Promise((res) => {
        fs.unlink(`/${bucket}/${path}`, (err) => {
          if (err) {
            console.error(err);
            return res();
          }

          return res();
        });
      })
  );
};
