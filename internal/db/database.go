package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"petropavlovsk-budget/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func New() (*Database, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	db := &Database{Pool: pool}
	if err := db.initSchema(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Database) initSchema() error {
	ctx := context.Background()

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS projects (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		category TEXT NOT NULL,
		district TEXT NOT NULL,
		budget INT NOT NULL,
		lat FLOAT NOT NULL,
		lng FLOAT NOT NULL,
		images JSONB DEFAULT '[]',
		status TEXT DEFAULT 'voting',
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS votes (
		id SERIAL PRIMARY KEY,
		project_id INT REFERENCES projects(id) ON DELETE CASCADE,
		user_id INT REFERENCES users(id) ON DELETE CASCADE,
		comment TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(project_id, user_id)
	);

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	CREATE INDEX IF NOT EXISTS idx_votes_project ON votes(project_id);
	`

	_, err := db.Pool.Exec(ctx, schema)
	return err
}

func (db *Database) CreateUser(email, passwordHash string) (*models.User, error) {
	ctx := context.Background()
	var user models.User

	err := db.Pool.QueryRow(ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, email, created_at",
		email, passwordHash,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *Database) GetUserByEmail(email string) (*models.User, error) {
	ctx := context.Background()
	var user models.User

	err := db.Pool.QueryRow(ctx,
		"SELECT id, email, password_hash, created_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *Database) GetUserByID(id int) (*models.User, error) {
	ctx := context.Background()
	var user models.User

	err := db.Pool.QueryRow(ctx,
		"SELECT id, email, created_at FROM users WHERE id = $1",
		id,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (db *Database) CreateProject(p *models.Project) error {
	ctx := context.Background()

	imagesJSON, err := json.Marshal(p.Images)
	if err != nil {
		return err
	}

	return db.Pool.QueryRow(ctx,
		`INSERT INTO projects (title, description, category, district, budget, lat, lng, images, status, user_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id, created_at`,
		p.Title, p.Description, p.Category, p.District, p.Budget, p.Lat, p.Lng, imagesJSON, p.Status, p.UserID,
	).Scan(&p.ID, &p.CreatedAt)
}

func (db *Database) GetAllProjects() ([]models.Project, error) {
	ctx := context.Background()
	rows, err := db.Pool.Query(ctx,
		`SELECT p.id, p.title, p.description, p.category, p.district, p.budget, 
		        p.lat, p.lng, p.images, p.status, p.user_id, p.created_at,
		        COUNT(v.id) as vote_count
		 FROM projects p
		 LEFT JOIN votes v ON p.id = v.project_id
		 GROUP BY p.id
		 ORDER BY p.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		var imagesJSON []byte

		err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.District,
			&p.Budget, &p.Lat, &p.Lng, &imagesJSON, &p.Status, &p.UserID, &p.CreatedAt, &p.VoteCount)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(imagesJSON, &p.Images); err != nil {
			p.Images = []string{}
		}

		projects = append(projects, p)
	}

	return projects, nil
}

func (db *Database) GetProjectByID(id int) (*models.Project, error) {
	ctx := context.Background()
	var p models.Project
	var imagesJSON []byte

	err := db.Pool.QueryRow(ctx,
		`SELECT p.id, p.title, p.description, p.category, p.district, p.budget,
		        p.lat, p.lng, p.images, p.status, p.user_id, p.created_at,
		        COUNT(v.id) as vote_count
		 FROM projects p
		 LEFT JOIN votes v ON p.id = v.project_id
		 WHERE p.id = $1
		 GROUP BY p.id`,
		id,
	).Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.District,
		&p.Budget, &p.Lat, &p.Lng, &imagesJSON, &p.Status, &p.UserID, &p.CreatedAt, &p.VoteCount)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(imagesJSON, &p.Images); err != nil {
		p.Images = []string{}
	}

	return &p, nil
}

func (db *Database) CreateVote(projectID, userID int, comment string) error {
	ctx := context.Background()

	_, err := db.Pool.Exec(ctx,
		"INSERT INTO votes (project_id, user_id, comment) VALUES ($1, $2, $3)",
		projectID, userID, comment,
	)

	return err
}

func (db *Database) HasUserVoted(projectID, userID int) (bool, error) {
	ctx := context.Background()
	var count int

	err := db.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM votes WHERE project_id = $1 AND user_id = $2",
		projectID, userID,
	).Scan(&count)

	return count > 0, err
}

func (db *Database) GetProjectVotes(projectID int) ([]models.Vote, error) {
	ctx := context.Background()
	rows, err := db.Pool.Query(ctx,
		"SELECT id, project_id, user_id, comment, created_at FROM votes WHERE project_id = $1 ORDER BY created_at DESC",
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var votes []models.Vote
	for rows.Next() {
		var v models.Vote
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.UserID, &v.Comment, &v.CreatedAt); err != nil {
			return nil, err
		}
		votes = append(votes, v)
	}

	return votes, nil
}

func (db *Database) Close() {
	db.Pool.Close()
}
