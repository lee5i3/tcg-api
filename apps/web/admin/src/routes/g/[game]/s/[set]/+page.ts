import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => ({
  game: params.game,
  set: params.set,
});
