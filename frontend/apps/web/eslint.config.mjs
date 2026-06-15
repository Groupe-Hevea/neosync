import next from 'eslint-config-next';

// eslint-config-next 16 ships a native flat config (core-web-vitals + typescript,
// with @typescript-eslint already registered), so we spread it directly instead of
// bridging the legacy config through FlatCompat.
export default [
  ...next,
  {
    files: ['**/*.{ts,tsx}'],
    rules: {
      'no-unused-vars': 'off',

      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_',
          caughtErrorsIgnorePattern: '^_',
        },
      ],

      'no-console': [
        'error',
        {
          allow: ['error', 'warn'],
        },
      ],

      '@typescript-eslint/no-explicit-any': 'error',

      // eslint-config-next 16 ships a React-Compiler-era react-hooks plugin that
      // promotes these opinionated rules to errors. Keep them at warning level
      // (like react-hooks/exhaustive-deps) so they surface for future cleanup
      // without blocking on pre-existing patterns.
      'react-hooks/set-state-in-effect': 'warn',
      'react-hooks/globals': 'warn',
    },
  },
  {
    ignores: ['.next/**'],
  },
];
