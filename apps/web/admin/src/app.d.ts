declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }

  interface ImportMetaEnv {
    /**
     * Base URL of the TCG catalog API. Leave unset for same-origin
     * requests (relative /v1/... URLs). See .env.example.
     */
    readonly VITE_API_URL?: string;
  }
}

export {};
