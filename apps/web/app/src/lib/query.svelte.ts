export type QueryState<T> =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "ready"; data: T };

export interface Query<T> {
  readonly state: QueryState<T>;
}

/**
 * Client-side data fetching helper (these are static sites — all data comes
 * from the REST API in the browser).
 *
 * Must be called during component initialisation. The loader is re-run
 * whenever any reactive state it reads synchronously (e.g. the `data` prop
 * from a `+page.ts` load) changes; stale responses are discarded.
 */
export function createQuery<T>(load: () => Promise<T>): Query<T> {
  let state = $state.raw<QueryState<T>>({ status: "loading" });

  $effect(() => {
    let cancelled = false;
    state = { status: "loading" };
    load().then(
      (data) => {
        if (!cancelled) {
          state = { status: "ready", data };
        }
      },
      (error: unknown) => {
        if (!cancelled) {
          state = {
            status: "error",
            message: error instanceof Error ? error.message : String(error),
          };
        }
      },
    );
    return () => {
      cancelled = true;
    };
  });

  return {
    get state() {
      return state;
    },
  };
}
