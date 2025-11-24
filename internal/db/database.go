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
                role TEXT DEFAULT 'citizen',
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
                status TEXT DEFAULT 'moderation',
                ai_analysis TEXT,
                vote_start TIMESTAMP,
                vote_end TIMESTAMP,
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

        CREATE TABLE IF NOT EXISTS comments (
                id SERIAL PRIMARY KEY,
                project_id INT REFERENCES projects(id) ON DELETE CASCADE,
                user_id INT REFERENCES users(id) ON DELETE CASCADE,
                content TEXT NOT NULL,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        CREATE TABLE IF NOT EXISTS project_status_history (
                id SERIAL PRIMARY KEY,
                project_id INT REFERENCES projects(id) ON DELETE CASCADE,
                status TEXT NOT NULL,
                comment TEXT,
                admin_id INT REFERENCES users(id),
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );

        CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
        CREATE INDEX IF NOT EXISTS idx_votes_project ON votes(project_id);
        CREATE INDEX IF NOT EXISTS idx_comments_project ON comments(project_id);
        CREATE INDEX IF NOT EXISTS idx_status_history_project ON project_status_history(project_id);
        `

        _, err := db.Pool.Exec(ctx, schema)
        if err != nil {
                return err
        }

        _, err = db.Pool.Exec(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'citizen'")
        if err != nil {
                return err
        }

        _, err = db.Pool.Exec(ctx, "ALTER TABLE projects ADD COLUMN IF NOT EXISTS vote_start TIMESTAMP")
        if err != nil {
                return err
        }

        _, err = db.Pool.Exec(ctx, "ALTER TABLE projects ADD COLUMN IF NOT EXISTS vote_end TIMESTAMP")
        if err != nil {
                return err
        }

        _, err = db.Pool.Exec(ctx, "ALTER TABLE projects ADD COLUMN IF NOT EXISTS ai_analysis TEXT")
        if err != nil {
                return err
        }

        _, err = db.Pool.Exec(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS nickname TEXT")
        if err != nil {
                return err
        }

        _, err = db.Pool.Exec(ctx, "UPDATE projects SET status = 'moderation' WHERE status = 'voting' AND id NOT IN (SELECT DISTINCT project_id FROM votes)")
        
        return nil
}

func (db *Database) CreateUser(email, nickname, passwordHash string) (*models.User, error) {
        ctx := context.Background()
        var user models.User

        err := db.Pool.QueryRow(ctx,
                "INSERT INTO users (email, nickname, password_hash, role) VALUES ($1, $2, $3, 'citizen') RETURNING id, email, nickname, role, created_at",
                email, nickname, passwordHash,
        ).Scan(&user.ID, &user.Email, &user.Nickname, &user.Role, &user.CreatedAt)

        if err != nil {
                return nil, err
        }

        return &user, nil
}

func (db *Database) CreateAdmin(email, nickname, passwordHash string) (*models.User, error) {
        ctx := context.Background()
        var user models.User

        err := db.Pool.QueryRow(ctx,
                "INSERT INTO users (email, nickname, password_hash, role) VALUES ($1, $2, $3, 'admin') RETURNING id, email, nickname, role, created_at",
                email, nickname, passwordHash,
        ).Scan(&user.ID, &user.Email, &user.Nickname, &user.Role, &user.CreatedAt)

        if err != nil {
                return nil, err
        }

        return &user, nil
}

func (db *Database) GetUserByEmail(email string) (*models.User, error) {
        ctx := context.Background()
        var user models.User
        var nickname *string

        err := db.Pool.QueryRow(ctx,
                "SELECT id, email, nickname, password_hash, role, created_at FROM users WHERE email = $1",
                email,
        ).Scan(&user.ID, &user.Email, &nickname, &user.PasswordHash, &user.Role, &user.CreatedAt)

        if err != nil {
                return nil, err
        }

        if nickname != nil {
                user.Nickname = *nickname
        }

        return &user, nil
}

func (db *Database) GetUserByID(id int) (*models.User, error) {
        ctx := context.Background()
        var user models.User
        var nickname *string

        err := db.Pool.QueryRow(ctx,
                "SELECT id, email, nickname, role, created_at FROM users WHERE id = $1",
                id,
        ).Scan(&user.ID, &user.Email, &nickname, &user.Role, &user.CreatedAt)

        if err != nil {
                return nil, err
        }

        if nickname != nil {
                user.Nickname = *nickname
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
                `INSERT INTO projects (title, description, category, district, budget, lat, lng, images, status, ai_analysis, user_id)
                 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id, created_at`,
                p.Title, p.Description, p.Category, p.District, p.Budget, p.Lat, p.Lng, imagesJSON, p.Status, p.AIAnalysis, p.UserID,
        ).Scan(&p.ID, &p.CreatedAt)
}

func (db *Database) GetAllProjects() ([]models.Project, error) {
        ctx := context.Background()
        rows, err := db.Pool.Query(ctx,
                `SELECT p.id, p.title, p.description, p.category, p.district, p.budget, 
                        p.lat, p.lng, p.images, p.status, p.ai_analysis, p.user_id, p.created_at,
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
                var aiAnalysis *string

                err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.District,
                        &p.Budget, &p.Lat, &p.Lng, &imagesJSON, &p.Status, &aiAnalysis, &p.UserID, &p.CreatedAt, &p.VoteCount)
                if err != nil {
                        return nil, err
                }

                if err := json.Unmarshal(imagesJSON, &p.Images); err != nil {
                        p.Images = []string{}
                }

                if aiAnalysis != nil {
                        p.AIAnalysis = *aiAnalysis
                }

                projects = append(projects, p)
        }

        return projects, nil
}

func (db *Database) GetProjectByID(id int) (*models.Project, error) {
        ctx := context.Background()
        var p models.Project
        var imagesJSON []byte
        var aiAnalysis *string

        err := db.Pool.QueryRow(ctx,
                `SELECT p.id, p.title, p.description, p.category, p.district, p.budget,
                        p.lat, p.lng, p.images, p.status, p.ai_analysis, p.user_id, p.created_at,
                        COUNT(v.id) as vote_count
                 FROM projects p
                 LEFT JOIN votes v ON p.id = v.project_id
                 WHERE p.id = $1
                 GROUP BY p.id`,
                id,
        ).Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.District,
                &p.Budget, &p.Lat, &p.Lng, &imagesJSON, &p.Status, &aiAnalysis, &p.UserID, &p.CreatedAt, &p.VoteCount)

        if err != nil {
                return nil, err
        }

        if err := json.Unmarshal(imagesJSON, &p.Images); err != nil {
                p.Images = []string{}
        }

        if aiAnalysis != nil {
                p.AIAnalysis = *aiAnalysis
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

func (db *Database) CreateComment(projectID, userID int, content string) error {
        ctx := context.Background()

        _, err := db.Pool.Exec(ctx,
                "INSERT INTO comments (project_id, user_id, content) VALUES ($1, $2, $3)",
                projectID, userID, content,
        )

        return err
}

func (db *Database) GetProjectComments(projectID int) ([]models.Comment, error) {
        ctx := context.Background()
        rows, err := db.Pool.Query(ctx,
                `SELECT c.id, c.project_id, c.user_id, u.email, c.content, c.created_at 
                 FROM comments c 
                 JOIN users u ON c.user_id = u.id 
                 WHERE c.project_id = $1 
                 ORDER BY c.created_at DESC`,
                projectID,
        )
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var comments []models.Comment
        for rows.Next() {
                var c models.Comment
                if err := rows.Scan(&c.ID, &c.ProjectID, &c.UserID, &c.UserEmail, &c.Content, &c.CreatedAt); err != nil {
                        return nil, err
                }
                comments = append(comments, c)
        }

        return comments, nil
}

func (db *Database) UpdateProjectStatus(projectID int, status string, adminID int, comment string) error {
        ctx := context.Background()

        tx, err := db.Pool.Begin(ctx)
        if err != nil {
                return err
        }
        defer tx.Rollback(ctx)

        _, err = tx.Exec(ctx, "UPDATE projects SET status = $1 WHERE id = $2", status, projectID)
        if err != nil {
                return err
        }

        _, err = tx.Exec(ctx,
                "INSERT INTO project_status_history (project_id, status, comment, admin_id) VALUES ($1, $2, $3, $4)",
                projectID, status, comment, adminID,
        )
        if err != nil {
                return err
        }

        return tx.Commit(ctx)
}

func (db *Database) SetVotingPeriod(projectID int, voteStart, voteEnd string) error {
        ctx := context.Background()

        _, err := db.Pool.Exec(ctx,
                "UPDATE projects SET vote_start = $1, vote_end = $2 WHERE id = $3",
                voteStart, voteEnd, projectID,
        )

        return err
}

func (db *Database) GetProjectStatusHistory(projectID int) ([]models.ProjectStatusHistory, error) {
        ctx := context.Background()
        rows, err := db.Pool.Query(ctx,
                `SELECT id, project_id, status, comment, admin_id, created_at 
                 FROM project_status_history 
                 WHERE project_id = $1 
                 ORDER BY created_at DESC`,
                projectID,
        )
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var history []models.ProjectStatusHistory
        for rows.Next() {
                var h models.ProjectStatusHistory
                if err := rows.Scan(&h.ID, &h.ProjectID, &h.Status, &h.Comment, &h.AdminID, &h.CreatedAt); err != nil {
                        return nil, err
                }
                history = append(history, h)
        }

        return history, nil
}

func (db *Database) GetProjectsByStatus(status string) ([]models.Project, error) {
        ctx := context.Background()
        rows, err := db.Pool.Query(ctx,
                `SELECT p.id, p.title, p.description, p.category, p.district, p.budget, 
                        p.lat, p.lng, p.images, p.status, p.ai_analysis, p.user_id, p.created_at,
                        COUNT(v.id) as vote_count
                 FROM projects p
                 LEFT JOIN votes v ON p.id = v.project_id
                 WHERE p.status = $1
                 GROUP BY p.id
                 ORDER BY p.created_at DESC`,
                status,
        )
        if err != nil {
                return nil, err
        }
        defer rows.Close()

        var projects []models.Project
        for rows.Next() {
                var p models.Project
                var imagesJSON []byte
                var aiAnalysis *string

                err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.Category, &p.District,
                        &p.Budget, &p.Lat, &p.Lng, &imagesJSON, &p.Status, &aiAnalysis, &p.UserID, &p.CreatedAt, &p.VoteCount)
                if err != nil {
                        return nil, err
                }

                if err := json.Unmarshal(imagesJSON, &p.Images); err != nil {
                        p.Images = []string{}
                }

                if aiAnalysis != nil {
                        p.AIAnalysis = *aiAnalysis
                }

                projects = append(projects, p)
        }

        return projects, nil
}

func (db *Database) UpdateProject(projectID int, title, description, category, district string, budget int) error {
        ctx := context.Background()

        _, err := db.Pool.Exec(ctx,
                "UPDATE projects SET title = $1, description = $2, category = $3, district = $4, budget = $5 WHERE id = $6",
                title, description, category, district, budget, projectID,
        )

        return err
}

func (db *Database) Close() {
        db.Pool.Close()
}
