import type { Config } from 'jest';
import { createDefaultPreset } from 'ts-jest';

// TypeScript 6 turns the previously-inferred common source directory into an error
// (TS5011), so ts-jest needs an explicit rootDir for the test files it compiles.
const tsJestPreset = createDefaultPreset({
  tsconfig: { rootDir: '.' },
});

const config: Config = {
  ...tsJestPreset,
  testEnvironment: 'node',
  verbose: true,
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/$1',
  },
};

export default config;
