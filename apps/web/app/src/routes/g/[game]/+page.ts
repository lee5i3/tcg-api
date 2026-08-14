import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params, url }) => ({
  game: params.game,
  query: url.searchParams.get("query") ?? "",
});
