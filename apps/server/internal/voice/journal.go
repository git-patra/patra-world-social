package voice

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PGJournal struct{ Pool *pgxpool.Pool }

func (j PGJournal) Save(ctx context.Context, r Record) error {
	_, err := j.Pool.Exec(ctx, `INSERT INTO voice_revocations(identity,room) VALUES($1,$2) ON CONFLICT(identity) DO NOTHING`, r.ID, r.Room)
	return err
}
func (j PGJournal) Delete(ctx context.Context, id string) error {
	_, err := j.Pool.Exec(ctx, `DELETE FROM voice_revocations WHERE identity=$1`, id)
	return err
}
func (j PGJournal) Load(ctx context.Context) ([]Record, error) {
	rows, err := j.Pool.Query(ctx, `SELECT identity,room FROM voice_revocations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		var r Record
		if err = rows.Scan(&r.ID, &r.Room); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
