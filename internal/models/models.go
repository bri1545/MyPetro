package models

import "time"

type User struct {
        ID           int       `json:"id"`
        Email        string    `json:"email"`
        PasswordHash string    `json:"-"`
        Role         string    `json:"role"`
        CreatedAt    time.Time `json:"created_at"`
}

type Project struct {
        ID          int       `json:"id"`
        Title       string    `json:"title"`
        Description string    `json:"description"`
        Category    string    `json:"category"`
        District    string    `json:"district"`
        Budget      int       `json:"budget"`
        Lat         float64   `json:"lat"`
        Lng         float64   `json:"lng"`
        Images      []string  `json:"images"`
        Status      string    `json:"status"`
        VoteStart   *time.Time `json:"vote_start,omitempty"`
        VoteEnd     *time.Time `json:"vote_end,omitempty"`
        UserID      int       `json:"user_id"`
        CreatedAt   time.Time `json:"created_at"`
        VoteCount   int       `json:"vote_count"`
}

type Vote struct {
        ID        int       `json:"id"`
        ProjectID int       `json:"project_id"`
        UserID    int       `json:"user_id"`
        Comment   string    `json:"comment"`
        CreatedAt time.Time `json:"created_at"`
}

type ProjectSubmission struct {
        Title       string
        Description string
        Category    string
        District    string
        Budget      int
        Lat         float64
        Lng         float64
}

type Comment struct {
        ID        int       `json:"id"`
        ProjectID int       `json:"project_id"`
        UserID    int       `json:"user_id"`
        UserEmail string    `json:"user_email"`
        Content   string    `json:"content"`
        CreatedAt time.Time `json:"created_at"`
}

type ProjectStatusHistory struct {
        ID        int       `json:"id"`
        ProjectID int       `json:"project_id"`
        Status    string    `json:"status"`
        Comment   string    `json:"comment"`
        AdminID   int       `json:"admin_id"`
        CreatedAt time.Time `json:"created_at"`
}
