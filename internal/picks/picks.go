package picks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thebenkogan/ufc/internal/auth"
)

type EventPicksRepository interface {
	GetPicks(ctx context.Context, user auth.User, eventId string) ([]string, error)
	SavePicks(ctx context.Context, user auth.User, eventId string, picks []string) error
}

type PostgresEventPicks struct {
	client *pgxpool.Pool
}

func NewPostgresEventPicks(client *pgxpool.Pool) *PostgresEventPicks {
	return &PostgresEventPicks{
		client: client,
	}
}

func (p *PostgresEventPicks) GetPicks(ctx context.Context, user auth.User, eventId string) ([]string, error) {
	var picks []string
	if err := p.client.QueryRow(ctx, "SELECT picks FROM picks WHERE user_id = $1 AND event_id = $2", user.Id, eventId).Scan(&picks); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return picks, nil
}

func (p *PostgresEventPicks) SavePicks(ctx context.Context, user auth.User, eventId string, picks []string) error {
	if _, err := p.client.Exec(ctx, "INSERT INTO picks VALUES ($1, $2, $3) ON CONFLICT (user_id, event_id) DO UPDATE SET picks = EXCLUDED.picks", user.Id, eventId, picks); err != nil {
		return err
	}
	return nil
}
