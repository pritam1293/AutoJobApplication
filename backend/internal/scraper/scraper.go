package scraper

import (
	"context"
	"time"

	"github.com/jobhaunt/backend/internal/models"
)

type JobResult struct {
	Platform    string    `json:"platform"`
	JobID       string    `json:"job_id"`
	Title       string    `json:"title"`
	Company     string    `json:"company"`
	Location    string    `json:"location"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Salary      string    `json:"salary"`
	PostedAt    time.Time `json:"posted_at"`
	Skills      string    `json:"skills"`
}

type Scraper interface {
	Name() string
	Search(ctx context.Context, query string, location string) ([]JobResult, error)
	GetJobDetails(ctx context.Context, job *models.Job) error
}

type Config struct {
	LinkedInEmail    string
	LinkedInPassword string
	IndeedEmail      string
	IndeedPassword   string
	MaxRetries       int
	Timeout          time.Duration
}

func NewConfig() Config {
	return Config{
		MaxRetries: 3,
		Timeout:    30 * time.Second,
	}
}
