/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_URL: string;
  readonly VITE_CHIP_PROJECT_PATH: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
