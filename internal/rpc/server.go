// Package rpc is the gRPC surface — the trusted service-to-service API with
// full read/write access to the catalog. Public read access goes through
// GraphQL instead (internal/graphql).
package rpc

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lee5i3/pokemon-invest/internal/catalog"
	tcgapiv1 "github.com/lee5i3/pokemon-invest/internal/pb/tcgapi/v1"
)

type Server struct {
	tcgapiv1.UnimplementedCatalogServiceServer
	Catalog *catalog.Catalog
}

// NewGRPCServer builds the gRPC server with auth (when token is non-empty)
// and reflection (for grpcurl and service discovery).
func NewGRPCServer(cat *catalog.Catalog, token string) *grpc.Server {
	srv := grpc.NewServer(grpc.UnaryInterceptor(authInterceptor(token)))
	tcgapiv1.RegisterCatalogServiceServer(srv, &Server{Catalog: cat})
	reflection.Register(srv)
	return srv
}

// authInterceptor requires "authorization: Bearer <token>" metadata on every
// call when a token is configured. Reflection stays open so services can
// discover the contract.
func authInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (any, error) {
		if token == "" || strings.HasPrefix(info.FullMethod, "/grpc.reflection.") {
			return handler(ctx, req)
		}
		md, _ := metadata.FromIncomingContext(ctx)
		for _, v := range md.Get("authorization") {
			got, ok := strings.CutPrefix(v, "Bearer ")
			if ok && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1 {
				return handler(ctx, req)
			}
		}
		return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func int32Ptr(p *int) *int32 {
	if p == nil {
		return nil
	}
	v := int32(*p)
	return &v
}

func intPtr(p *int32) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

// grpcError maps catalog sentinel errors onto gRPC status codes.
func grpcError(err error) error {
	switch {
	case errors.Is(err, catalog.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, catalog.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, catalog.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

// resolveGame turns a game reference (GUID or key) into the games row,
// 404ing unknown games before the handler logic runs.
func (s *Server) resolveGame(ctx context.Context, game string) (catalog.Game, error) {
	g, err := s.Catalog.ResolveGame(ctx, game)
	if errors.Is(err, catalog.ErrNotFound) {
		return catalog.Game{}, status.Errorf(codes.NotFound, "unknown game %q", game)
	}
	if err != nil {
		return catalog.Game{}, grpcError(err)
	}
	return g, nil
}

// ---------------------------------------------------------------------------
// Shape conversions
// ---------------------------------------------------------------------------

func pbVariants(variants []catalog.CardVariant) []*tcgapiv1.CardVariant {
	out := make([]*tcgapiv1.CardVariant, 0, len(variants))
	for _, v := range variants {
		out = append(out, &tcgapiv1.CardVariant{Id: v.ID, Name: v.Name, Price: v.Price})
	}
	return out
}

func pbCardFromSummary(gameID string, c catalog.CardSummary) *tcgapiv1.Card {
	return &tcgapiv1.Card{
		Id:          c.ID,
		TcgplayerId: int32Ptr(c.TCGPlayerID),
		Language:    c.Language,
		GameId:      gameID,
		SetId:       c.SetID,
		Name:        c.Name,
		Number:      c.Number,
		Rarity:      c.Rarity,
		ImageSmall:  c.Image,
		ImageLarge:  c.ImageLarge,
		Variants:    pbVariants(c.Variants),
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}
}

// ---------------------------------------------------------------------------
// Games
// ---------------------------------------------------------------------------

func (s *Server) ListGames(ctx context.Context, _ *tcgapiv1.ListGamesRequest) (*tcgapiv1.ListGamesResponse, error) {
	games, err := s.Catalog.Games(ctx)
	if err != nil {
		return nil, grpcError(err)
	}
	out := make([]*tcgapiv1.Game, 0, len(games))
	for _, g := range games {
		out = append(out, &tcgapiv1.Game{Id: g.ID, Key: g.Key, Language: g.Language, Label: g.Label, UpdatedAt: timestamppb.New(g.UpdatedAt)})
	}
	return &tcgapiv1.ListGamesResponse{Games: out}, nil
}

func (s *Server) CreateGame(ctx context.Context, req *tcgapiv1.CreateGameRequest) (*tcgapiv1.CreateGameResponse, error) {
	g, err := s.Catalog.AddGame(ctx, req.GetKey(), req.GetLabel(), req.Language)
	if err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.CreateGameResponse{
		Game: &tcgapiv1.Game{Id: g.ID, Key: g.Key, Language: g.Language, Label: g.Label, UpdatedAt: timestamppb.New(g.UpdatedAt)},
	}, nil
}

// ---------------------------------------------------------------------------
// Sets
// ---------------------------------------------------------------------------

func (s *Server) ListSets(ctx context.Context, req *tcgapiv1.ListSetsRequest) (*tcgapiv1.ListSetsResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	sets, err := s.Catalog.ListSets(ctx, game.ID, req.GetQuery())
	if err != nil {
		return nil, grpcError(err)
	}
	out := make([]*tcgapiv1.Set, 0, len(sets))
	for _, st := range sets {
		out = append(out, &tcgapiv1.Set{
			Id:          st.ID,
			Key:         st.Key,
			Language:    st.Language,
			GameId:      st.GameID,
			Name:        st.Name,
			CardCount:   int32(st.CardCount),
			ReleaseDate: st.ReleaseDate,
			CardTotal:   int32Ptr(st.CardTotal),
			LogoUrl:     st.LogoURL,
			CreatedAt:   timestamppb.New(st.CreatedAt),
			UpdatedAt:   timestamppb.New(st.UpdatedAt),
		})
	}
	return &tcgapiv1.ListSetsResponse{Sets: out}, nil
}

func (s *Server) CreateSet(ctx context.Context, req *tcgapiv1.CreateSetRequest) (*tcgapiv1.CreateSetResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	if req.GetKey() == "" || strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "set key and name are required")
	}
	in := catalog.SetInput{
		Key: req.GetKey(), Language: req.Language, Name: req.GetName(),
		ReleaseDate: req.ReleaseDate, CardTotal: intPtr(req.CardTotal), LogoURL: req.LogoUrl,
	}
	id, err := s.Catalog.CreateSet(ctx, game.ID, in)
	if err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.CreateSetResponse{Id: id, Key: req.GetKey()}, nil
}

func (s *Server) UpdateSet(ctx context.Context, req *tcgapiv1.UpdateSetRequest) (*tcgapiv1.UpdateSetResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "set name is required")
	}
	in := catalog.SetInput{
		Name: req.GetName(),
		ReleaseDate: req.ReleaseDate, CardTotal: intPtr(req.CardTotal), LogoURL: req.LogoUrl,
	}
	if err := s.Catalog.UpdateSet(ctx, game.ID, req.GetId(), in); err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.UpdateSetResponse{}, nil
}

func (s *Server) DeleteSet(ctx context.Context, req *tcgapiv1.DeleteSetRequest) (*tcgapiv1.DeleteSetResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	removed, err := s.Catalog.DeleteSet(ctx, game.ID, req.GetId())
	if err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.DeleteSetResponse{RemovedCards: int32(removed)}, nil
}

// ---------------------------------------------------------------------------
// Cards
// ---------------------------------------------------------------------------

func (s *Server) ListSetCards(ctx context.Context, req *tcgapiv1.ListSetCardsRequest) (*tcgapiv1.ListSetCardsResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	cards, err := s.Catalog.SetCards(ctx, game.ID, req.GetSetId())
	if err != nil {
		return nil, grpcError(err)
	}
	out := make([]*tcgapiv1.Card, 0, len(cards))
	for _, c := range cards {
		out = append(out, pbCardFromSummary(game.ID, c))
	}
	return &tcgapiv1.ListSetCardsResponse{Cards: out}, nil
}

func (s *Server) SearchCards(ctx context.Context, req *tcgapiv1.SearchCardsRequest) (*tcgapiv1.SearchCardsResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(req.GetQuery())) < 2 {
		return &tcgapiv1.SearchCardsResponse{Cards: []*tcgapiv1.Card{}}, nil
	}
	cards, err := s.Catalog.SearchCards(ctx, game.ID, req.GetQuery())
	if err != nil {
		return nil, grpcError(err)
	}
	out := make([]*tcgapiv1.Card, 0, len(cards))
	for _, c := range cards {
		out = append(out, pbCardFromSummary(game.ID, c))
	}
	return &tcgapiv1.SearchCardsResponse{Cards: out}, nil
}

func (s *Server) GetCard(ctx context.Context, req *tcgapiv1.GetCardRequest) (*tcgapiv1.GetCardResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	d, err := s.Catalog.GetCard(ctx, game.ID, req.GetId())
	if err != nil {
		return nil, grpcError(err)
	}
	card := &tcgapiv1.Card{
		Id:          d.ID,
		TcgplayerId: int32Ptr(d.TCGPlayerID),
		Language:    d.Language,
		GameId:      d.GameID,
		SetId:       d.Set,
		Name:        d.Name,
		Number:      deref(d.Number),
		Rarity:      d.Rarity,
		ImageSmall:  d.Images.Small,
		ImageLarge:  d.Images.Large,
		Variants:    pbVariants(d.Variants),
		CreatedAt:   timestamppb.New(d.CreatedAt),
		UpdatedAt:   timestamppb.New(d.UpdatedAt),
	}
	return &tcgapiv1.GetCardResponse{Card: card}, nil
}

func cardInput(tcgplayerID *int32, setID, name string, number, rarity, imageSmall, imageLarge *string) catalog.CardInput {
	return catalog.CardInput{
		TCGPlayerID: intPtr(tcgplayerID), SetID: setID, Name: name,
		Number: number, Rarity: rarity, ImageSmall: imageSmall, ImageLarge: imageLarge,
	}
}

func (s *Server) CreateCard(ctx context.Context, req *tcgapiv1.CreateCardRequest) (*tcgapiv1.CreateCardResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	if req.GetSetId() == "" || strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "set_id and name are required")
	}
	in := cardInput(req.TcgplayerId, req.GetSetId(), req.GetName(),
		req.Number, req.Rarity, req.ImageSmall, req.ImageLarge)
	in.Language = req.Language
	id, err := s.Catalog.CreateCard(ctx, game, in)
	if err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.CreateCardResponse{Id: id}, nil
}

func (s *Server) UpdateCard(ctx context.Context, req *tcgapiv1.UpdateCardRequest) (*tcgapiv1.UpdateCardResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	if req.GetSetId() == "" || strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "set_id and name are required")
	}
	in := cardInput(req.TcgplayerId, req.GetSetId(), req.GetName(),
		req.Number, req.Rarity, req.ImageSmall, req.ImageLarge)
	if err := s.Catalog.UpdateCard(ctx, game, req.GetId(), in); err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.UpdateCardResponse{}, nil
}

func (s *Server) DeleteCard(ctx context.Context, req *tcgapiv1.DeleteCardRequest) (*tcgapiv1.DeleteCardResponse, error) {
	game, err := s.resolveGame(ctx, req.GetGame())
	if err != nil {
		return nil, err
	}
	if err := s.Catalog.DeleteCard(ctx, game.ID, req.GetId()); err != nil {
		return nil, grpcError(err)
	}
	return &tcgapiv1.DeleteCardResponse{}, nil
}
