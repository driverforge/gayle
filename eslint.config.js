const { FlatCompat } = require('@eslint/eslintrc');
const js = require('@eslint/js');
const prettierConfig = require('eslint-config-prettier');
const prettierPlugin = require('eslint-plugin-prettier');
const jestPlugin = require('eslint-plugin-jest');
const globals = require('globals');

const compat = new FlatCompat({
  baseDirectory: __dirname,
});

module.exports = [
  {
    ignores: ['node_modules/', 'coverage/'],
  },
  js.configs.recommended,
  ...compat.extends('airbnb-base'),
  prettierConfig,
  {
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'commonjs',
      globals: {
        ...globals.node,
        ...globals.es2021,
        Atomics: 'readonly',
        SharedArrayBuffer: 'readonly',
      },
    },
    plugins: {
      prettier: prettierPlugin,
      jest: jestPlugin,
    },
    rules: {
      'prettier/prettier': 'error',
      'import/prefer-default-export': 'off',
      'no-restricted-imports': ['error'],
    },
  },
  {
    files: ['**/*.test.js', '**/*.spec.js', '__mocks__/**/*.js'],
    languageOptions: {
      globals: globals.jest,
    },
    ...jestPlugin.configs['flat/recommended'],
  },
  {
    files: ['eslint.config.js', 'jest.config.js'],
    rules: {
      'import/no-extraneous-dependencies': ['error', { devDependencies: true }],
    },
  },
];
