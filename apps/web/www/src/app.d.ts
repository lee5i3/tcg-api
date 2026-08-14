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
     * Absolute URL of the public catalog app, used by every CTA.
     * Defaults to the placeholder https://app.example.com. See .env.example.
     */
    readonly PUBLIC_APP_URL?: string;
  }
}

export {};
