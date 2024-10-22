import eslint from '@eslint/js';
import prettier from 'eslint-config-prettier';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  eslint.configs.recommended,

  ...tseslint.configs.recommendedTypeChecked,
  {
    rules: {
      'array-callback-return': 'error',
      'no-constant-binary-expression': 'error',
      'no-self-compare': 'error',
      'no-template-curly-in-string': 'error',
      'no-unmodified-loop-condition': 'error',
      'no-unreachable-loop': 'error',
      'arrow-body-style': ['error', 'as-needed'],
      'block-scoped-var': 'error',
      curly: ['error', 'all'],
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-var': 'error',
      'one-var': ['error', 'never'],
      'prefer-arrow-callback': 'error',
      'prefer-const': 'error',
      yoda: 'error',
      'array-bracket-newline': ['error', { multiline: true }],
      'brace-style': 'error',
      'no-shadow': 'error',
      'no-use-before-define': 'error',
      'dot-notation': 'error',
      'no-async-promise-executor': 'off',
      // Maybe review in future
      '@typescript-eslint/no-misused-promises': 'off'
    }
  },
  {
    languageOptions: {
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname
      }
    }
  },

  ...svelte.configs['flat/recommended'],
  prettier,
  ...svelte.configs['flat/prettier'],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      }
    }
  },
  {
    files: ['**/*.svelte'],
    languageOptions: {
      parserOptions: {
        extraFileExtensions: ['.svelte'],
        parser: tseslint.parser
      }
    },
    rules: {
      // These are broken due to $props() right now
      '@typescript-eslint/no-unsafe-assignment': 'off',
      'prefer-const': 'off'
    }
  },
  {
    ignores: [
      '**/*.js',
      '**/*.cjs',
      '.env.*',
      '.env',
      '.svelte-kit/',
      'build/',
      'dist/',
      'node_modules',
      'package-lock.json',
      'package/',
      'pnpm-lock.yaml',
      'static/tinymce/'
    ]
  }
);
