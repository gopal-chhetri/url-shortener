package url

import (
	"context"
	"testing"

	"github.com/google/uuid"
	dbgen "github.com/gopal-chhetri/url-shortener/internal/db/sqlc"
	"github.com/gopal-chhetri/url-shortener/internal/infra"
	"github.com/gopal-chhetri/url-shortener/internal/response"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// fakeURLRepo stubs the repository methods the service tests exercise; any
// other method panics via the nil embedded interface.
type fakeURLRepo struct {
	UrlRepositoryInterface
	urls    map[uuid.UUID]dbgen.Url
	updated bool
	deleted bool
}

func (f *fakeURLRepo) GetURLByID(_ context.Context, dto GetURLByIDDTO) (dbgen.Url, error) {
	u, ok := f.urls[dto.ID]
	if !ok {
		return dbgen.Url{}, response.NotFoundError{Model: "url"}
	}
	return u, nil
}

func (f *fakeURLRepo) UpdateURL(_ context.Context, dto UpdateURLDTO) (dbgen.Url, error) {
	f.updated = true
	u := f.urls[dto.ID]
	u.OriginalUrl = dto.OriginalURL
	return u, nil
}

func (f *fakeURLRepo) UpdateURLStatus(_ context.Context, dto UpdateURLStatusDTO) (dbgen.Url, error) {
	f.updated = true
	return f.urls[dto.ID], nil
}

func (f *fakeURLRepo) DeleteURL(_ context.Context, _ DeleteURLDTO) error {
	f.deleted = true
	return nil
}

func newTestService(repo UrlRepositoryInterface) *UrlService {
	return &UrlService{repo: repo, env: &infra.Env{}, logger: zap.NewNop()}
}

func TestOwnershipChecks(t *testing.T) {
	owner := uuid.New()
	other := uuid.New()
	ownedID := uuid.New()
	anonID := uuid.New()

	newRepo := func() *fakeURLRepo {
		return &fakeURLRepo{urls: map[uuid.UUID]dbgen.Url{
			ownedID: {ID: ownedID, ShortUrl: "owned", UserID: pgtype.UUID{Bytes: owner, Valid: true}},
			anonID:  {ID: anonID, ShortUrl: "anon"},
		}}
	}

	tests := []struct {
		name    string
		urlID   uuid.UUID
		caller  uuid.UUID
		allowed bool
	}{
		{"owner can modify own URL", ownedID, owner, true},
		{"other user cannot modify owned URL", ownedID, other, false},
		{"no user can modify anonymous URL", anonID, other, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			var notFound response.NotFoundError

			repo := newRepo()
			_, err := newTestService(repo).UpdateURL(ctx, UpdateURLRequest{ID: tt.urlID, OriginalURL: "https://evil.example"}, tt.caller)
			assert.Equal(t, tt.allowed, err == nil, "UpdateURL")
			assert.Equal(t, tt.allowed, repo.updated, "UpdateURL reached repo")
			if !tt.allowed {
				require.ErrorAs(t, err, &notFound)
			}

			repo = newRepo()
			err = newTestService(repo).DeleteURL(ctx, tt.urlID, tt.caller)
			assert.Equal(t, tt.allowed, err == nil, "DeleteURL")
			assert.Equal(t, tt.allowed, repo.deleted, "DeleteURL reached repo")

			repo = newRepo()
			_, err = newTestService(repo).UpdateURLStatus(ctx, tt.urlID, tt.caller, false)
			assert.Equal(t, tt.allowed, err == nil, "UpdateURLStatus")
			assert.Equal(t, tt.allowed, repo.updated, "UpdateURLStatus reached repo")
		})
	}
}

// slugRepo simulates the urls.short_url unique constraint.
type slugRepo struct {
	UrlRepositoryInterface
	taken   map[string]bool
	inserts int
}

func (f *slugRepo) CreateURL(_ context.Context, dto CreateURLDTO) (dbgen.Url, error) {
	f.inserts++
	if f.taken[dto.ShortURL] {
		return dbgen.Url{}, response.DuplicateData{Model: "url"}
	}
	f.taken[dto.ShortURL] = true
	return dbgen.Url{ShortUrl: dto.ShortURL, OriginalUrl: dto.OriginalURL}, nil
}

func TestCreateURL_CustomSlugTakenIsConflict(t *testing.T) {
	repo := &slugRepo{taken: map[string]bool{"mine": true}}
	_, err := newTestService(repo).CreateURL(context.Background(), CreateURLRequest{
		OriginalURL: "https://example.com", CustomSlug: "mine",
	})
	var dup response.DuplicateData
	require.ErrorAs(t, err, &dup)
	assert.Equal(t, 1, repo.inserts, "must not fall back to a random slug")
}

func TestCreateURL_GeneratedSlugRetriesOnCollision(t *testing.T) {
	repo := &slugRepo{taken: map[string]bool{}}
	svc := newTestService(repo)
	ctx := context.Background()

	first, err := svc.CreateURL(ctx, CreateURLRequest{OriginalURL: "https://example.com"})
	require.NoError(t, err)

	// Force every unsalted attempt to collide with the first slug.
	repo.taken = map[string]bool{first.ShortUrl: true}
	repo.inserts = 0
	orig := first.ShortUrl
	collide := &collideFirstRepo{slugRepo: repo, slug: orig}
	second, err := newTestService(collide).CreateURL(ctx, CreateURLRequest{OriginalURL: "https://example.com"})
	require.NoError(t, err)
	assert.NotEqual(t, orig, second.ShortUrl)
	assert.Equal(t, 2, repo.inserts)
}

// collideFirstRepo makes the first insert collide regardless of slug.
type collideFirstRepo struct {
	*slugRepo
	slug string
	done bool
}

func (f *collideFirstRepo) CreateURL(ctx context.Context, dto CreateURLDTO) (dbgen.Url, error) {
	if !f.done {
		f.done = true
		dto.ShortURL = f.slug
	}
	return f.slugRepo.CreateURL(ctx, dto)
}
