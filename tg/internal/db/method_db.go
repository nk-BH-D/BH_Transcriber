package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresUs struct {
	DB_us *sql.DB
}

func NewUsPostgres(db_us_url string, smoc, smic int) (*PostgresUs, error) {
	db, err := sql.Open("pgx", db_us_url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(smoc)
	db.SetMaxIdleConns(smic)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	p := &PostgresUs{DB_us: db}
	if err := p.ensureSchrema(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *PostgresUs) CloseUs() error {
	return p.DB_us.Close()
}

func (p *PostgresUs) ensureSchrema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	schrema := `
	CREATE TABLE IF NOT EXISTS users (
		user_id BIGINT PRIMARY KEY,
		status TEXT NOT NULL DEFAULT 'user'
			CHECK (status in ('user', 'admin', 'root')),
		accept BOOLEAN NOT NULL DEFAULT FALSE,		
		num_transcrip INTEGER NOT NULL DEFAULT 0,
		like_count INTEGER NOT NULL DEFAULT 0,
		dis_like INTEGER NOT NULL DEFAULT 0,
		creared_ad TIMESTAMP WITH TIME ZONE DEFAULT now()
	);`

	_, err := p.DB_us.ExecContext(ctx, schrema)
	return err
}

func (p *PostgresUs) InsertUser(ctx context.Context, user_id int64, status string) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"INSERT INTO users (user_id, status) VALUES($1, $2)",
		user_id,
		status,
	)
	return err
}

func (p *PostgresUs) GetUserData(ctx context.Context, user_id int64) (bool, int, int, int, time.Time, error) {
	var (
		accept   bool
		num_t    int
		like     int
		dis_like int
		reg_time time.Time
	)
	row := p.DB_us.QueryRowContext(
		ctx,
		"SELECT accept, num_transcrip, like_count, dis_like, creared_ad FROM users WHERE user_id=$1",
		user_id,
	)
	if err := row.Scan(&accept, &num_t, &like, &dis_like, &reg_time); err != nil {
		return false, 0, 0, 0, time.Time{}, err
	}
	return accept, num_t, like, dis_like, reg_time, nil
}

func (p *PostgresUs) UpUserAccept(ctx context.Context, user_id int64, accept bool) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"UPDATE users SET accept=$2 WHERE user_id=$1",
		user_id,
		accept,
	)
	return err
}

func (p *PostgresUs) UpUserData(ctx context.Context, user_id int64, num_tr, like, dis_like int) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"UPDATE users SET num_transcrip=num_transcrip+$2, like_count=like_count+$3, dis_like=dis_like+$4 WHERE user_id=$1",
		user_id,
		num_tr,
		like,
		dis_like,
	)
	return err
}

func (p *PostgresUs) DeleteUserData(ctx context.Context, user_id int64) error {
	_, err := p.DB_us.ExecContext(
		ctx,
		"DELETE FROM users WHERE user_id=$1",
		user_id,
	)
	return err
}

type PostgresErrors struct {
	DB_err *sql.DB
}

func NewErrPostgres(db_err_url string, smoc, smic int) (*PostgresErrors, error) {
	db, err := sql.Open("pgx", db_err_url)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(smoc)
	db.SetMaxIdleConns(smic)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	pe := &PostgresErrors{DB_err: db}
	if err := pe.ensureSchremaErr(); err != nil {
		return nil, err
	}

	return pe, nil
}

func (pe *PostgresErrors) CloseErr() error {
	return pe.DB_err.Close()
}

func (pe *PostgresErrors) ensureSchremaErr() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	schrema := `
	CREATE TABLE IF NOT EXISTS errors (
    	id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    	user_id BIGINT NOT NULL,
    	comment TEXT NOT NULL,
    	status TEXT NOT NULL DEFAULT 'new'
        	CHECK (status IN ('new', 'in_progress', 'resolved', 'no_valid_comment')),
    	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    	resolved_at TIMESTAMPTZ
	);`

	_, err := pe.DB_err.ExecContext(ctx, schrema)
	return err
}

func (pe *PostgresErrors) InsertError(ctx context.Context, userID int64, comment string) (int64, error) {
	var id int64

	err := pe.DB_err.QueryRowContext(
		ctx,
		`INSERT INTO errors (user_id, comment)
         VALUES ($1, $2)
         RETURNING id`,
		userID,
		comment,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (pe *PostgresErrors) GetUnresolvedErrors(ctx context.Context) (map[int64]map[int64]string, error) {
	result := make(map[int64]map[int64]string)

	rows, err := pe.DB_err.QueryContext(
		ctx,
		`SELECT id, user_id, comment
         FROM errors
         WHERE status IN ('new', 'in_progress')
         ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user_id int64
		var errorID int64
		var comment string

		if err := rows.Scan(&errorID, &user_id, &comment); err != nil {
			return nil, err
		}

		if result[user_id] == nil {
			result[user_id] = make(map[int64]string)
		}

		result[user_id][errorID] = comment
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (pe *PostgresErrors) GetUserError(ctx context.Context, userID int64) (map[int64]string, error) {
	res := make(map[int64]string)

	rows, err := pe.DB_err.QueryContext(
		ctx,
		`SELECT id, status
         FROM errors
         WHERE user_id=$1
         ORDER BY created_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			errID  int64
			status string
		)

		if err := rows.Scan(&errID, &status); err != nil {
			return nil, err
		}

		if res[errID] == "" {
			res[errID] = status
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func (pe *PostgresErrors) ResolveError(ctx context.Context, errorID int64, set_status string) (string, error) {
	var status string

	err := pe.DB_err.QueryRowContext(
		ctx,
		`UPDATE errors
         SET status = $2,
             resolved_at = NOW()
         WHERE id = $1 AND status = 'new'
		 RETURNING status`,
		errorID,
		set_status,
	).Scan(&status)
	return status, err
}

func (pe *PostgresErrors) FindError(ctx context.Context, errorID int64) (string, error) {
	var status string

	err := pe.DB_err.QueryRowContext(
		ctx,
		`SELECT status
		FROM errors
		WHERE id=$1`,
		errorID,
	).Scan(&status)
	if err != nil {
		return "", err
	}

	return status, nil
}
