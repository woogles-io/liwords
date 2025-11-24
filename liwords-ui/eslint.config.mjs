import { defineConfig, globalIgnores } from "eslint/config";
import { fixupConfigRules, fixupPluginRules } from "@eslint/compat";
import typescriptEslint from "@typescript-eslint/eslint-plugin";
import react from "eslint-plugin-react";
import tsParser from "@typescript-eslint/parser";
import path from "node:path";
import { fileURLToPath } from "node:url";
import js from "@eslint/js";
import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const compat = new FlatCompat({
  baseDirectory: __dirname,
  recommendedConfig: js.configs.recommended,
  allConfig: js.configs.all,
});

export default defineConfig([
  globalIgnores(["build/*", "public/*", "wolges-wasm-pkg/*", "src/gen/*"]),
  {
    extends: fixupConfigRules(
      compat.extends(
        "plugin:@typescript-eslint/recommended",
        "plugin:react/recommended",
        "plugin:react-hooks/recommended",
        "plugin:prettier/recommended",
      ),
    ),

    plugins: {
      "@typescript-eslint": fixupPluginRules(typescriptEslint),
      react: fixupPluginRules(react),
    },

    languageOptions: {
      parser: tsParser,
      ecmaVersion: 5,
      sourceType: "script",

      parserOptions: {
        project: "tsconfig.json",
      },
    },

    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
        },
      },

      react: {
        version: "18.3.1",
      },
    },

    rules: {
      "object-curly-spacing": ["warn", "always"],

      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          vars: "all",
          args: "none",
        },
      ],

      "@typescript-eslint/no-explicit-any": [
        "error",
        {
          ignoreRestArgs: true,
        },
      ],

      "max-len": [
        "warn",
        {
          code: 80,
          ignoreStrings: true,
          ignoreTemplateLiterals: true,
          ignoreComments: true,
        },
      ],

      "no-restricted-imports": [
        "error",
        {
          paths: [
            {
              name: "dayjs",
              importNames: ["default"],
              message: "Please import dayjs from src/utils/datetime instead",
            },
          ],
        },
      ],

      "no-plusplus": "off",
      "react/react-in-jsx-scope": "off",
      "react/jsx-key": "error",
      "react/jsx-props-no-spreading": "off",
      "import/prefer-default-export": "off",
      "react/jsx-boolean-value": "off",
      "react/prop-types": "off",
      "react/no-unescaped-entities": "off",
      "react/jsx-one-expression-per-line": "off",
      "react/jsx-wrap-multilines": "off",
      "react/destructuring-assignment": "off",
      "@typescript-eslint/explicit-function-return-type": "off",
      "@typescript-eslint/no-empty-function": "off",
      "react/display-name": "off",
      "react/no-children-prop": "off",
      "react-hooks/exhaustive-deps": "error",
      "react-hooks/set-state-in-effect": "warn",
      "react-hooks/purity": "warn",
      "react-hooks/refs": "warn",
      "react-hooks/static-components": "warn",
      "react-hooks/immutability": "warn",
    },
  },
]);
