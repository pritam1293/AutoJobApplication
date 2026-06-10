package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name             string `json:"name"`
	Email            string `json:"email" gorm:"uniqueIndex"`
	LinkedInEmail    string `json:"linkedin_email"`
	LinkedInPassword string `json:"-"`
	IndeedEmail      string `json:"indeed_email"`
	IndeedPassword   string `json:"-"`
	OpenAIKey        string `json:"-"`
	ResumePath       string `json:"resume_path"`
	LatexSource      string `json:"latex_source" gorm:"type:text"`
}

type Job struct {
	gorm.Model
	Platform    string `json:"platform" gorm:"index"`
	JobID       string `json:"job_id" gorm:"uniqueIndex"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Location    string `json:"location"`
	URL         string `json:"url"`
	Description string `json:"description" gorm:"type:text"`
	Salary      string `json:"salary"`
	PostedAt    time.Time `json:"posted_at"`
	Skills      string `json:"skills" gorm:"type:text"`
	Status      string `json:"status" gorm:"default:'new'"` // new, matched, applied, rejected, skipped
}

type Application struct {
	gorm.Model
	JobID       uint      `json:"job_id" gorm:"index"`
	Job         Job       `json:"job" gorm:"foreignKey:JobID"`
	UserID      uint      `json:"user_id" gorm:"index"`
	User        User      `json:"user" gorm:"foreignKey:UserID"`
	Status      string    `json:"status" gorm:"default:'pending'"` // pending, submitted, failed, success
	ResumeUsed  string    `json:"resume_used" gorm:"type:text"`
	Score       float64   `json:"score"`
	AppliedAt   time.Time `json:"applied_at"`
	Notes       string    `json:"notes" gorm:"type:text"`
	TailoredJD  string    `json:"tailored_jd" gorm:"type:text"`
}

type Resume struct {
	gorm.Model
	UserID     uint   `json:"user_id" gorm:"index"`
	FilePath   string `json:"file_path"`
	BaseData   string `json:"base_data" gorm:"type:text"` // parsed base resume as JSON
	Tailored   bool   `json:"tailored" gorm:"default:false"`
	ForJobID   *uint  `json:"for_job_id"`
	TailoredPDF string `json:"tailored_pdf" gorm:"type:text"`
}

type SearchQuery struct {
	gorm.Model
	UserID     uint   `json:"user_id"`
	Query      string `json:"query"`
	Location   string `json:"location"`
	Platforms  string `json:"platforms"` // comma-separated
	Active     bool   `json:"active" gorm:"default:true"`
	AutoApply  bool   `json:"auto_apply" gorm:"default:false"`
	MaxApplied int    `json:"max_applied" gorm:"default:50"`
}
