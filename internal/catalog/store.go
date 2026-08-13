package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Catalog is the storage facade the API is built on.
type Catalog struct {
	DB *pgxpool.Pool
}

// ErrNotFound is returned when a game, set, or card doesn't exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when creating something that already exists.
var ErrConflict = errors.New("already exists")

// ErrInvalid is returned for well-formed requests that reference the wrong
// things (e.g. creating a card in a set that doesn't exist).
var ErrInvalid = errors.New("invalid input")

func New(db *pgxpool.Pool) *Catalog {
	return &Catalog{DB: db}
}

// ---------------------------------------------------------------------------
// Games
// ---------------------------------------------------------------------------

var defaultGames = []Game{
	{Key: "pokemon", Label: "Pokémon"},
	{Key: "magic", Label: "Magic: The Gathering"},
	{Key: "lorcana", Label: "Disney Lorcana"},
	{Key: "sports", Label: "Sports cards"},
}

// Seed inserts the default games when the table is empty.
func (c *Catalog) Seed(ctx context.Context) error {
	var n int
	if err := c.DB.QueryRow(ctx, `SELECT count(*) FROM games`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for _, g := range defaultGames {
		if _, err := c.DB.Exec(ctx,
			`INSERT INTO games (id, key, label) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
			uuid.NewString(), g.Key, g.Label); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) Games(ctx context.Context) ([]Game, error) {
	rows, err := c.DB.Query(ctx, `SELECT id, key, language, label, updated_at FROM games ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	games, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Game])
	if err != nil {
		return nil, err
	}
	if games == nil {
		games = []Game{}
	}
	return games, nil
}

var gameKeyRe = regexp.MustCompile(`^[a-z0-9-]{2,32}$`)

var languageRe = regexp.MustCompile(`^[a-z]{3}$`)

// normLanguage validates an ISO 639 alpha-3 language reference.
// A nil/blank value defaults to "eng".
func normLanguage(raw *string) (string, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return "eng", nil
	}
	lang := strings.ToLower(strings.TrimSpace(*raw))
	if !languageRe.MatchString(lang) {
		return "", fmt.Errorf("%w: language must be an ISO 639 alpha-3 code (e.g. \"eng\", \"jpn\")", ErrInvalid)
	}
	return lang, nil
}

func (c *Catalog) AddGame(ctx context.Context, key, label string, language *string) (Game, error) {
	if !gameKeyRe.MatchString(key) {
		return Game{}, fmt.Errorf("%w: key must be 2–32 characters: lowercase letters, numbers, dashes", ErrInvalid)
	}
	if strings.TrimSpace(label) == "" {
		return Game{}, fmt.Errorf("%w: give the game a name", ErrInvalid)
	}
	lang, err := normLanguage(language)
	if err != nil {
		return Game{}, err
	}
	g := Game{ID: uuid.NewString(), Key: key, Language: lang, Label: strings.TrimSpace(label)}
	err = c.DB.QueryRow(ctx,
		`INSERT INTO games (id, key, language, label) VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING RETURNING updated_at`,
		g.ID, g.Key, g.Language, g.Label).Scan(&g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Game{}, fmt.Errorf("%w: game %q", ErrConflict, key)
	}
	if err != nil {
		return Game{}, err
	}
	return g, nil
}

// ResolveGame turns a game reference (GUID or key) into the full row —
// sets reference games by GUID while cards still use the key.
func (c *Catalog) ResolveGame(ctx context.Context, idOrKey string) (Game, error) {
	var g Game
	err := c.DB.QueryRow(ctx,
		`SELECT id, key, language, label, updated_at FROM games WHERE key = $1 OR id::text = $1`,
		idOrKey).Scan(&g.ID, &g.Key, &g.Language, &g.Label, &g.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Game{}, fmt.Errorf("%w: game %q", ErrNotFound, idOrKey)
	}
	return g, err
}

// ---------------------------------------------------------------------------
// Sets
// ---------------------------------------------------------------------------

// escapeLike neutralizes ILIKE wildcards in user input.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// parseReleaseDate accepts YYYY-MM-DD (or the source-style YYYY/MM/DD) and
// keeps the time at midnight — the column is a plain date.
func parseReleaseDate(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02"} {
		if t, err := time.Parse(layout, strings.TrimSpace(*raw)); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("%w: releaseDate must be YYYY-MM-DD", ErrInvalid)
}

// resolveSetID turns a set reference (GUID or key) into the set's GUID.
func (c *Catalog) resolveSetID(ctx context.Context, gameID, idOrKey string) (string, error) {
	var id string
	err := c.DB.QueryRow(ctx,
		`SELECT id FROM sets WHERE game_id = $1 AND (id::text = $2 OR key = $2)`,
		gameID, idOrKey).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("%w: set %q", ErrNotFound, idOrKey)
	}
	return id, err
}

const setSelect = `
	SELECT s.id, s.key, s.language, s.game_id, s.name, count(c.id)::int,
	       s.release_date, s.card_total, s.logo_url, s.created_at, s.updated_at
	FROM sets s
	LEFT JOIN cards c ON c.set_id = s.id`

func scanSets(rows pgx.Rows) ([]SetSummary, error) {
	defer rows.Close()
	sets := []SetSummary{}
	for rows.Next() {
		var st SetSummary
		var releaseDate *time.Time
		if err := rows.Scan(&st.ID, &st.Key, &st.Language, &st.GameID, &st.Name,
			&st.CardCount, &releaseDate, &st.CardTotal, &st.LogoURL,
			&st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, err
		}
		if releaseDate != nil {
			formatted := releaseDate.Format("2006-01-02")
			st.ReleaseDate = &formatted
		}
		sets = append(sets, st)
	}
	return sets, rows.Err()
}

// ListSets returns a game's sets with live card counts, newest release first.
func (c *Catalog) ListSets(ctx context.Context, gameID, query string) ([]SetSummary, error) {
	rows, err := c.DB.Query(ctx, setSelect+`
		WHERE s.game_id = $1
		  AND ($2 = '' OR s.name ILIKE '%' || $2 || '%')
		GROUP BY s.id
		ORDER BY s.release_date DESC NULLS LAST, s.name`,
		gameID, escapeLike(strings.TrimSpace(query)))
	if err != nil {
		return nil, err
	}
	return scanSets(rows)
}

// CreateSet adds a set and returns its GUID.
func (c *Catalog) CreateSet(ctx context.Context, gameID string, in SetInput) (string, error) {
	releaseDate, err := parseReleaseDate(in.ReleaseDate)
	if err != nil {
		return "", err
	}
	lang, err := normLanguage(in.Language)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	tag, err := c.DB.Exec(ctx, `
		INSERT INTO sets (id, key, language, game_id, name, release_date, card_total, logo_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (game_id, key) DO NOTHING`,
		id, in.Key, lang, gameID, in.Name, releaseDate, in.CardTotal, in.LogoURL)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", fmt.Errorf("%w: set %q", ErrConflict, in.Key)
	}
	return id, nil
}

// UpdateSet rewrites a set's fields. The key is immutable; idOrKey accepts
// the GUID or the key.
func (c *Catalog) UpdateSet(ctx context.Context, gameID, idOrKey string, in SetInput) error {
	releaseDate, err := parseReleaseDate(in.ReleaseDate)
	if err != nil {
		return err
	}
	tag, err := c.DB.Exec(ctx, `
		UPDATE sets
		SET name = $3, release_date = $4, card_total = $5, logo_url = $6
		WHERE game_id = $1 AND (id::text = $2 OR key = $2)`,
		gameID, idOrKey, in.Name, releaseDate, in.CardTotal, in.LogoURL)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: set %q", ErrNotFound, idOrKey)
	}
	return nil
}

// DeleteSet removes a set; its cards and their variants cascade away.
// Returns the number of cards removed.
func (c *Catalog) DeleteSet(ctx context.Context, gameID, idOrKey string) (int, error) {
	setID, err := c.resolveSetID(ctx, gameID, idOrKey)
	if err != nil {
		return 0, err
	}
	var n int
	if err := c.DB.QueryRow(ctx,
		`SELECT count(*) FROM cards WHERE set_id = $1`, setID).Scan(&n); err != nil {
		return 0, err
	}
	if _, err := c.DB.Exec(ctx, `DELETE FROM sets WHERE id = $1`, setID); err != nil {
		return 0, err
	}
	return n, nil
}

// ---------------------------------------------------------------------------
// Cards (reads)
// ---------------------------------------------------------------------------

// A card's printings and their current prices, aggregated to one JSON value
// per card.
const variantsJoin = `
	LEFT JOIN LATERAL (
		SELECT coalesce(jsonb_agg(jsonb_build_object(
			'id', v.id, 'name', v.name, 'price', v.price::float8)
			ORDER BY v.name), '[]'::jsonb) AS list
		FROM card_variants v
		WHERE v.card_id = c.id
	) cv ON true`

const cardSummarySelect = `
	SELECT c.id, c.tcgplayer_id, c.language, c.name, coalesce(c.number, ''), c.rarity, c.set_id,
	       c.image_small, c.image_large, c.created_at, c.updated_at, cv.list
	FROM cards c` + variantsJoin

func scanCardSummaries(rows pgx.Rows) ([]CardSummary, error) {
	defer rows.Close()
	cards := []CardSummary{}
	for rows.Next() {
		var card CardSummary
		var variantsJSON []byte
		if err := rows.Scan(&card.ID, &card.TCGPlayerID, &card.Language, &card.Name, &card.Number, &card.Rarity,
			&card.SetID, &card.Image, &card.ImageLarge, &card.CreatedAt, &card.UpdatedAt,
			&variantsJSON); err != nil {
			return nil, err
		}
		card.Variants = decodeVariants(variantsJSON)
		cards = append(cards, card)
	}
	return cards, rows.Err()
}

func decodeVariants(raw []byte) []CardVariant {
	variants := []CardVariant{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &variants)
	}
	return variants
}

// SearchCards searches a game's catalog by card name, newest set first.
func (c *Catalog) SearchCards(ctx context.Context, gameID, query string) ([]CardSummary, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []CardSummary{}, nil
	}
	rows, err := c.DB.Query(ctx, cardSummarySelect+`
		LEFT JOIN sets s ON s.id = c.set_id
		WHERE c.game_id = $1 AND c.name ILIKE '%' || $2 || '%'
		ORDER BY s.release_date DESC NULLS LAST,
		         coalesce((substring(c.number FROM '^[0-9]+'))::int, 0),
		         c.number
		LIMIT 48`,
		gameID, escapeLike(q))
	if err != nil {
		return nil, err
	}
	return scanCardSummaries(rows)
}

// SetCards returns every card of a set (GUID or key), in collector-number
// order.
func (c *Catalog) SetCards(ctx context.Context, gameID, setIDOrKey string) ([]CardSummary, error) {
	setID, err := c.resolveSetID(ctx, gameID, setIDOrKey)
	if err != nil {
		return nil, err
	}
	rows, err := c.DB.Query(ctx, cardSummarySelect+`
		WHERE c.set_id = $1
		ORDER BY coalesce((substring(c.number FROM '^[0-9]+'))::int, 0), c.number`,
		setID)
	if err != nil {
		return nil, err
	}
	return scanCardSummaries(rows)
}

// GetCard fetches one card by GUID or TCGplayer id, with its variants and
// their current prices.
func (c *Catalog) GetCard(ctx context.Context, gameID, idOrKey string) (*CardDetail, error) {
	var d CardDetail
	var variantsJSON []byte
	err := c.DB.QueryRow(ctx, `
		SELECT c.id, c.tcgplayer_id, c.language, c.game_id, c.name, c.number, c.rarity, c.set_id,
		       c.image_small, c.image_large, c.created_at, c.updated_at, cv.list
		FROM cards c`+variantsJoin+`
		WHERE c.game_id = $1 AND (c.id::text = $2 OR c.tcgplayer_id::text = $2)`,
		gameID, idOrKey).Scan(
		&d.ID, &d.TCGPlayerID, &d.Language, &d.GameID, &d.Name, &d.Number, &d.Rarity, &d.Set,
		&d.Images.Small, &d.Images.Large, &d.CreatedAt, &d.UpdatedAt, &variantsJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: card %q", ErrNotFound, idOrKey)
	}
	if err != nil {
		return nil, err
	}
	d.Variants = decodeVariants(variantsJSON)
	return &d, nil
}

// ---------------------------------------------------------------------------
// Cards (writes)
// ---------------------------------------------------------------------------

// CreateCard adds a card to a set (GUID or key) and returns the card's GUID.
func (c *Catalog) CreateCard(ctx context.Context, game Game, in CardInput) (string, error) {
	setID, err := c.resolveSetID(ctx, game.ID, in.SetID)
	if err != nil {
		return "", fmt.Errorf("%w: set %q does not exist in game %q", ErrInvalid, in.SetID, game.Key)
	}
	lang, err := normLanguage(in.Language)
	if err != nil {
		return "", err
	}
	id := uuid.NewString()
	_, err = c.DB.Exec(ctx, `
		INSERT INTO cards (id, tcgplayer_id, language, game_id, set_id, name, number, rarity, image_small, image_large)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, in.TCGPlayerID, lang, game.ID, setID, in.Name, in.Number, in.Rarity, in.ImageSmall, in.ImageLarge)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateCard rewrites a card's fields; idOrKey accepts the GUID or the
// TCGplayer id.
func (c *Catalog) UpdateCard(ctx context.Context, game Game, idOrKey string, in CardInput) error {
	setID, err := c.resolveSetID(ctx, game.ID, in.SetID)
	if err != nil {
		return fmt.Errorf("%w: set %q does not exist in game %q", ErrInvalid, in.SetID, game.Key)
	}
	tag, err := c.DB.Exec(ctx, `
		UPDATE cards
		SET tcgplayer_id = $3, set_id = $4, name = $5, number = $6, rarity = $7,
		    image_small = $8, image_large = $9
		WHERE game_id = $1 AND (id::text = $2 OR tcgplayer_id::text = $2)`,
		game.ID, idOrKey, in.TCGPlayerID, setID, in.Name, in.Number, in.Rarity, in.ImageSmall, in.ImageLarge)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: card %q", ErrNotFound, idOrKey)
	}
	return nil
}

// DeleteCard removes a card; its variants cascade away.
func (c *Catalog) DeleteCard(ctx context.Context, gameID, idOrKey string) error {
	tag, err := c.DB.Exec(ctx,
		`DELETE FROM cards WHERE game_id = $1 AND (id::text = $2 OR tcgplayer_id::text = $2)`, gameID, idOrKey)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: card %q", ErrNotFound, idOrKey)
	}
	return nil
}
