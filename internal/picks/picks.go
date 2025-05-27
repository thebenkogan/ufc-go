package picks

import (
	"context"
	"fmt"
	"strings"
	"time"

	pgx "github.com/jackc/pgx/v5"
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

type PicksFilter struct {
	EventIDs []string
	HasScore *bool
}

type EventPicksRepository interface {
	GetUserPicksByEvent(ctx context.Context, user *auth.User, eventId string) (*Picks, error)
	GetAllUserPicks(ctx context.Context, user *auth.User) ([]*Picks, error)
	GetPicksByFilter(ctx context.Context, filter *PicksFilter) ([]*Picks, error)
	SavePicks(ctx context.Context, user *auth.User, eventId string, picks []string) error
	BatchScorePicks(ctx context.Context, picks []*Picks) []error
}

type PostgresEventPicks struct {
	client *pgxpool.Pool
}

func NewPostgresEventPicks(client *pgxpool.Pool) *PostgresEventPicks {
	return &PostgresEventPicks{
		client: client,
	}
}

func (p *PostgresEventPicks) GetUserPicksByEvent(ctx context.Context, user *auth.User, eventId string) (*Picks, error) {
	rows, _ := p.client.Query(ctx, "SELECT * FROM picks WHERE user_id = $1 AND event_id = $2", user.Id, eventId)
	picks, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Picks])
	if err != nil || len(picks) == 0 {
		return nil, err
	}
	return picks[0], nil
}

func (p *PostgresEventPicks) GetAllUserPicks(ctx context.Context, user *auth.User) ([]*Picks, error) {
	rows, _ := p.client.Query(ctx, "SELECT * FROM picks WHERE user_id = $1 ORDER BY created_at DESC", user.Id)
	picks, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[Picks])
	if err != nil {
		return nil, err
	}
	return picks, nil
}

func (p *PostgresEventPicks) GetPicksByFilter(ctx context.Context, filter *PicksFilter) ([]*Picks, error) {
	conds := make([]string, 0)
	args := make([]any, 0)
	argPos := 1

	if len(filter.EventIDs) > 0 {
		conds = append(conds, fmt.Sprintf("event_id = ANY($%d)", argPos))
		args = append(args, filter.EventIDs)
		argPos++
	}

	if filter.HasScore != nil {
		if *filter.HasScore {
			conds = append(conds, "score IS NOT NULL")
		} else {
			conds = append(conds, "score IS NULL")
		}
	}

	query := "SELECT * FROM picks"
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}

	rows, _ := p.client.Query(ctx, query, args...)
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

func (p *PostgresEventPicks) BatchScorePicks(ctx context.Context, picks []*Picks) []error {
	var batch pgx.Batch
	for _, pick := range picks {
		batch.Queue("UPDATE picks SET score = $1 WHERE user_id = $2 AND event_id = $3", pick.Score, pick.UserId, pick.EventId)
	}

	results := p.client.SendBatch(ctx, &batch)
	defer results.Close()

	errors := make([]error, 0)
	for range picks {
		_, err := results.Exec()
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
