package graphql

import (
	"context"
	"errors"
	"strings"
	"time"

	graphqlgo "github.com/graph-gophers/graphql-go"

	"github.com/lee5i3/pokemon-invest/internal/catalog"
)

// Resolver is the Query root.
type Resolver struct {
	Catalog *catalog.Catalog
}

// NewSchema parses the SDL against the resolver. Panics on mismatch, which
// is a compile-time-style failure caught at boot.
func NewSchema(cat *catalog.Catalog) *graphqlgo.Schema {
	return graphqlgo.MustParseSchema(Schema, &Resolver{Catalog: cat},
		graphqlgo.UseFieldResolvers())
}

type gameResolver struct {
	ID        graphqlgo.ID
	Key       string
	Language  string
	Label     string
	UpdatedAt string
}

type setResolver struct {
	ID          graphqlgo.ID
	Key         string
	Language    string
	GameID      graphqlgo.ID
	Name        string
	CardCount   int32
	ReleaseDate *string
	CardTotal   *int32
	LogoURL     *string
	CreatedAt   string
	UpdatedAt   string
}

type cardVariantResolver struct {
	ID    graphqlgo.ID
	Name  string
	Price *float64
}

type cardResolver struct {
	ID             graphqlgo.ID
	TCGPlayerID    *int32
	Language       string
	Name           string
	Number         string
	Rarity     *string
	SetID      graphqlgo.ID
	Image      *string
	ImageLarge *string
	Variants   []cardVariantResolver
	CreatedAt  string
	UpdatedAt  string
}

func int32Ptr(p *int) *int32 {
	if p == nil {
		return nil
	}
	v := int32(*p)
	return &v
}

func variants(vs []catalog.CardVariant) []cardVariantResolver {
	out := make([]cardVariantResolver, 0, len(vs))
	for _, v := range vs {
		out = append(out, cardVariantResolver{ID: graphqlgo.ID(v.ID), Name: v.Name, Price: v.Price})
	}
	return out
}

func card(c catalog.CardSummary) cardResolver {
	return cardResolver{
		ID:          graphqlgo.ID(c.ID),
		TCGPlayerID: int32Ptr(c.TCGPlayerID),
		Language:    c.Language,
		Name:        c.Name,
		Number:      c.Number,
		Rarity:      c.Rarity,
		SetID:       graphqlgo.ID(c.SetID),
		Image:       c.Image,
		ImageLarge:  c.ImageLarge,
		Variants:    variants(c.Variants),
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   c.UpdatedAt.Format(time.RFC3339),
	}
}

func cards(list []catalog.CardSummary) []cardResolver {
	out := make([]cardResolver, 0, len(list))
	for _, c := range list {
		out = append(out, card(c))
	}
	return out
}

// resolveGame turns a game reference (GUID or key) into the games row,
// surfacing unknown games as a query error up front.
func (r *Resolver) resolveGame(ctx context.Context, game string) (catalog.Game, error) {
	g, err := r.Catalog.ResolveGame(ctx, game)
	if errors.Is(err, catalog.ErrNotFound) {
		return catalog.Game{}, errors.New("unknown game " + game)
	}
	return g, err
}

// ---------------------------------------------------------------------------
// Query fields
// ---------------------------------------------------------------------------

func (r *Resolver) Games(ctx context.Context) ([]gameResolver, error) {
	games, err := r.Catalog.Games(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]gameResolver, 0, len(games))
	for _, g := range games {
		out = append(out, gameResolver{ID: graphqlgo.ID(g.ID), Key: g.Key, Language: g.Language, Label: g.Label, UpdatedAt: g.UpdatedAt.Format(time.RFC3339)})
	}
	return out, nil
}

func (r *Resolver) Sets(ctx context.Context, args struct {
	Game  string
	Query *string
}) ([]setResolver, error) {
	game, err := r.resolveGame(ctx, args.Game)
	if err != nil {
		return nil, err
	}
	q := ""
	if args.Query != nil {
		q = *args.Query
	}
	sets, err := r.Catalog.ListSets(ctx, game.ID, q)
	if err != nil {
		return nil, err
	}
	out := make([]setResolver, 0, len(sets))
	for _, s := range sets {
		out = append(out, setResolver{
			ID:          graphqlgo.ID(s.ID),
			Key:         s.Key,
			Language:    s.Language,
			GameID:      graphqlgo.ID(s.GameID),
			Name:        s.Name,
			CardCount:   int32(s.CardCount),
			ReleaseDate: s.ReleaseDate,
			CardTotal:   int32Ptr(s.CardTotal),
			LogoURL:     s.LogoURL,
			CreatedAt:   s.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   s.UpdatedAt.Format(time.RFC3339),
		})
	}
	return out, nil
}

func (r *Resolver) SetCards(ctx context.Context, args struct {
	Game  string
	SetID string
}) ([]cardResolver, error) {
	game, err := r.resolveGame(ctx, args.Game)
	if err != nil {
		return nil, err
	}
	list, err := r.Catalog.SetCards(ctx, game.ID, args.SetID)
	if err != nil {
		return nil, err
	}
	return cards(list), nil
}

func (r *Resolver) SearchCards(ctx context.Context, args struct {
	Game  string
	Query string
}) ([]cardResolver, error) {
	game, err := r.resolveGame(ctx, args.Game)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(args.Query)) < 2 {
		return []cardResolver{}, nil
	}
	list, err := r.Catalog.SearchCards(ctx, game.ID, args.Query)
	if err != nil {
		return nil, err
	}
	return cards(list), nil
}

func (r *Resolver) Card(ctx context.Context, args struct {
	Game string
	ID   string
}) (*cardResolver, error) {
	game, err := r.resolveGame(ctx, args.Game)
	if err != nil {
		return nil, err
	}
	d, err := r.Catalog.GetCard(ctx, game.ID, args.ID)
	if errors.Is(err, catalog.ErrNotFound) {
		return nil, nil // nullable in the schema — absence isn't an error
	}
	if err != nil {
		return nil, err
	}
	out := cardResolver{
		ID:          graphqlgo.ID(d.ID),
		TCGPlayerID: int32Ptr(d.TCGPlayerID),
		Language:    d.Language,
		Name:        d.Name,
		Rarity:      d.Rarity,
		SetID:       graphqlgo.ID(d.Set),
		Image:       d.Images.Small,
		ImageLarge:  d.Images.Large,
		Variants:    variants(d.Variants),
		CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}
	if d.Number != nil {
		out.Number = *d.Number
	}
	return &out, nil
}
