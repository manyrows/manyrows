package repo

import (
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
)

func (r *Repo) SetAccountLockedUntil(ctx context.Context, accountID uuid.UUID, lockedUntil time.Time) error {
	const q = `UPDATE accounts SET locked_until = $2 WHERE id = $1;`
	ct, err := r.db.Pool().Exec(ctx, q, accountID, lockedUntil)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ClearAccountLockedUntil(ctx context.Context, accountID uuid.UUID) error {
	const q = `UPDATE accounts SET locked_until = NULL WHERE id = $1;`
	ct, err := r.db.Pool().Exec(ctx, q, accountID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) SetUserLockedUntil(ctx context.Context, userID uuid.UUID, lockedUntil time.Time) error {
	const q = `UPDATE users SET locked_until = $2 WHERE id = $1;`
	ct, err := r.db.Pool().Exec(ctx, q, userID, lockedUntil)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ClearUserLockedUntil(ctx context.Context, userID uuid.UUID) error {
	const q = `UPDATE users SET locked_until = NULL WHERE id = $1;`
	ct, err := r.db.Pool().Exec(ctx, q, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
