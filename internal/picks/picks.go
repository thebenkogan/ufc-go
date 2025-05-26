package picks

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thebenkogan/ufc/internal/auth"
)

type Picks struct {
	UserId    string    `db:"user_id" json:"user_id"`
	EventId   string    `db:"event_id" json:"event_id"`
	Winners   []string  `db:"picks" json:"winners"`
	Score     *int      `db:"score" json:"score,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type EventPicksRepository interface {
	GetPicks(ctx context.Context, user *auth.User, eventId string) (*Picks, error)
	GetAllPicks(ctx context.Context, user *auth.User) ([]*Picks, error)
	SavePicks(ctx context.Context, user *auth.User, eventId string, picks []string) error
	ScorePicks(ctx context.Context, user *auth.User, eventId string, score int) error
}

type PostgresEventPicks struct {
	client *pgxpool.Pool
}

func NewPostgresEventPicks(client *pgxpool.Pool) *PostgresEventPicks {
	return &PostgresEventPicks{
		client: client,
	}
}

func (p *PostgresEventPicks) GetPicks(ctx context.Context, user *auth.User, eventId string) (*Picks, error) {
	rows, _ := p.client.Query(ctx, "SELECT * FROM picks WHERE user_id = $1 AND event_id = $2", user.Id, eventId)
	picks, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Picks])
	if err != nil || len(picks) == 0 {
		return nil, err
	}
	return picks[0], nil
}

func (p *PostgresEventPicks) GetAllPicks(ctx context.Context, user *auth.User) ([]*Picks, error) {
	rows, _ := p.client.Query(ctx, "SELECT * FROM picks WHERE user_id = $1 ORDER BY created_at DESC", user.Id)
	picks, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Picks])
	if err != nil {
		return nil, err
	}
	return picks, nil
}

func (p *PostgresEventPicks) SavePicks(ctx context.Context, user *auth.User, eventId string, picks []string) error {
	if _, err := p.client.Exec(ctx, "INSERT INTO picks VALUES ($1, $2, $3) ON CONFLICT (user_id, event_id) DO UPDATE SET picks = EXCLUDED.picks, created_at = CURRENT_TIMESTAMP", user.Id, eventId, picks); err != nil {
		return err
	}
	return nil
}

func (p *PostgresEventPicks) ScorePicks(ctx context.Context, user *auth.User, eventId string, score int) error {
	if _, err := p.client.Exec(ctx, "UPDATE picks SET score = $1 WHERE user_id = $2 AND event_id = $3", score, user.Id, eventId); err != nil {
		return err
	}
	return nil
}
