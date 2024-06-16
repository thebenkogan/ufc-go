package picks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thebenkogan/ufc/internal/auth"
)

type Picks struct {
	Winners []string `db:"picks" json:"winners"`
	Score   *int     `db:"score" json:"score,omitempty"`
}

type EventPicksRepository interface {
	GetPicks(ctx context.Context, user auth.User, eventId string) (*Picks, error)
	SavePicks(ctx context.Context, user auth.User, eventId string, picks []string) error
	ScorePicks(ctx context.Context, user auth.User, eventId string, score int) error
}

type PostgresEventPicks struct {
	client *pgxpool.Pool
}

func NewPostgresEventPicks(client *pgxpool.Pool) *PostgresEventPicks {
	return &PostgresEventPicks{
		client: client,
	}
}

func (p *PostgresEventPicks) GetPicks(ctx context.Context, user auth.User, eventId string) (*Picks, error) {
	var picks []string
	var score *int
	if err := p.client.QueryRow(ctx, "SELECT picks, score FROM picks WHERE user_id = $1 AND event_id = $2", user.Id, eventId).Scan(&picks, &score); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &Picks{
		Winners: picks,
		Score:   score,
	}, nil
}

func (p *PostgresEventPicks) SavePicks(ctx context.Context, user auth.User, eventId string, picks []string) error {
	if _, err := p.client.Exec(ctx, "INSERT INTO picks VALUES ($1, $2, $3) ON CONFLICT (user_id, event_id) DO UPDATE SET picks = EXCLUDED.picks", user.Id, eventId, picks); err != nil {
		return err
	}
	return nil
}

func (p *PostgresEventPicks) ScorePicks(ctx context.Context, user auth.User, eventId string, score int) error {
	if _, err := p.client.Exec(ctx, "UPDATE picks SET score = $1 WHERE user_id = $2 AND event_id = $3", score, user.Id, eventId); err != nil {
		return err
	}
	return nil
}
