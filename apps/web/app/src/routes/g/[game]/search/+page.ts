import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, url }) => ({
  game: params.game,
  q: url.searchParams.get("q") ?? "",
});
