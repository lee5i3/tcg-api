import type { PageLoad } from "./$types";

export const load: PageLoad = ({ params }) => ({
  game: params.game,
  card: params.card,
});
