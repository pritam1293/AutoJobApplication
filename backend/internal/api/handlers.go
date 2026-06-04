package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jobhaunt/backend/internal/ai"
	"github.com/jobhaunt/backend/internal/applicator"
	"github.com/jobhaunt/backend/internal/db"
	"github.com/jobhaunt/backend/internal/models"
	"github.com/jobhaunt/backend/internal/resume"
	"github.com/jobhaunt/backend/internal/scraper"
)

type Handler struct {
	aiClient      *ai.Client
	tailorEngine  *resume.TailorEngine
	applicator    *applicator.Applicator
	linkedinScr   *scraper.LinkedInScraper
	indeedScr     *scraper.IndeedScraper
	resumeManager *resume.Manager
}

func NewHandler(googleAIKey, linkedInEmail, linkedInPass, resumeDir string) *Handler {
	aiClient := ai.NewClient(googleAIKey)
	tailorEngine := resume.NewTailorEngine(aiClient)
	app := applicator.New(linkedInEmail, linkedInPass, "", "")

	return &Handler{
		aiClient:      aiClient,
		tailorEngine:  tailorEngine,
		applicator:    app,
		linkedinScr:   scraper.NewLinkedInScraper(linkedInEmail, linkedInPass),
		resumeManager: resume.NewManager(resumeDir),
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		api.GET("/health", h.healthCheck)

		// Search & Jobs
		api.POST("/jobs/search", h.searchJobs)
		api.GET("/jobs", h.listJobs)
		api.GET("/jobs/:id", h.getJob)
		api.PUT("/jobs/:id/status", h.updateJobStatus)

		// Applications
		api.POST("/applications/apply/:jobId", h.applyToJob)
		api.GET("/applications", h.listApplications)

		// Resume
		api.POST("/resume/upload", h.uploadResume)
		api.POST("/resume/tailor/:jobId", h.tailorResume)

		// Search Queries
		api.POST("/search-queries", h.createSearchQuery)
		api.GET("/search-queries", h.listSearchQueries)

		// Settings
		api.GET("/settings", h.getSettings)
		api.PUT("/settings", h.updateSettings)

		// Analytics
		api.GET("/analytics", h.getAnalytics)
	}
}

func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
}

func (h *Handler) searchJobs(c *gin.Context) {
	var req struct {
		Query    string `json:"query" binding:"required"`
		Location string `json:"location" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	jobs, err := h.linkedinScr.Search(ctx, req.Query, req.Location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("linkedin search failed: %v", err)})
		return
	}

	for _, j := range jobs {
		job := models.Job{
			Platform:    j.Platform,
			JobID:       j.JobID,
			Title:       j.Title,
			Company:     j.Company,
			Location:    j.Location,
			URL:         j.URL,
			Description: j.Description,
			Salary:      j.Salary,
			PostedAt:    j.PostedAt,
			Status:      "new",
		}
		db.DB.Where("job_id = ?", j.JobID).Attrs(job).FirstOrCreate(&job)
	}

	c.JSON(http.StatusOK, gin.H{
		"total": len(jobs),
		"jobs":  jobs,
	})
}

func (h *Handler) listJobs(c *gin.Context) {
	status := c.Query("status")
	platform := c.Query("platform")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	var jobs []models.Job
	query := db.DB
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	query.Order("created_at DESC").Limit(limit).Find(&jobs)

	c.JSON(http.StatusOK, gin.H{"total": len(jobs), "jobs": jobs})
}

func (h *Handler) getJob(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := db.DB.First(&job, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *Handler) updateJobStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db.DB.Model(&models.Job{}).Where("id = ?", id).Update("status", req.Status)
	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}

func (h *Handler) applyToJob(c *gin.Context) {
	jobID := c.Param("jobId")
	var job models.Job
	if err := db.DB.First(&job, jobID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Get user's resume
	var user models.User
	db.DB.First(&user)

	result, err := h.applicator.Apply(c.Request.Context(), &job, user.ResumePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record application
	app := models.Application{
		JobID:    job.ID,
		UserID:   user.ID,
		Status:   result.Status,
		AppliedAt: time.Now(),
		Notes:    result.Message,
	}
	db.DB.Create(&app)

	// Update job status
	if result.Status == "success" {
		db.DB.Model(&job).Update("status", "applied")
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) uploadResume(c *gin.Context) {
	file, err := c.FormFile("resume")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no resume file provided"})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only PDF files are supported"})
		return
	}

	resumeData, err := resume.ParsePDFFromReader(nil, "") // read from context
	_ = resumeData

	// Save file
	savePath := filepath.Join("uploads", file.Filename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	parsed, err := resume.ParsePDF(savePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to parse PDF: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "resume uploaded successfully",
		"file_path": savePath,
		"pages":     parsed.PageCount,
		"size":      parsed.FileSize,
	})
}

func (h *Handler) tailorResume(c *gin.Context) {
	jobID := c.Param("jobId")
	var job models.Job
	if err := db.DB.First(&job, jobID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	var req struct {
		Instructions string `json:"instructions"`
		ResumeData   string `json:"resume_data"`
	}
	c.ShouldBindJSON(&req)

	resp, err := h.tailorEngine.TailorForJob(c.Request.Context(), &resume.ResumeData{
		RawText: req.ResumeData,
	}, &job, req.Instructions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) listApplications(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	var apps []models.Application
	db.DB.Preload("Job").Order("created_at DESC").Limit(limit).Find(&apps)
	c.JSON(http.StatusOK, gin.H{"total": len(apps), "applications": apps})
}

func (h *Handler) createSearchQuery(c *gin.Context) {
	var req struct {
		Query     string `json:"query" binding:"required"`
		Location  string `json:"location" binding:"required"`
		Platforms string `json:"platforms"`
		AutoApply bool   `json:"auto_apply"`
		MaxApply  int    `json:"max_applied"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Platforms == "" {
		req.Platforms = "linkedin,indeed"
	}
	if req.MaxApply == 0 {
		req.MaxApply = 50
	}

	sq := models.SearchQuery{
		Query:     req.Query,
		Location:  req.Location,
		Platforms: req.Platforms,
		AutoApply: req.AutoApply,
		MaxApplied: req.MaxApply,
	}
	db.DB.Create(&sq)

	c.JSON(http.StatusCreated, sq)
}

func (h *Handler) listSearchQueries(c *gin.Context) {
	var queries []models.SearchQuery
	db.DB.Find(&queries)
	c.JSON(http.StatusOK, gin.H{"total": len(queries), "queries": queries})
}

func (h *Handler) getSettings(c *gin.Context) {
	var user models.User
	db.DB.First(&user)
	c.JSON(http.StatusOK, gin.H{
		"name":            user.Name,
		"email":           user.Email,
		"linkedin_email":  user.LinkedInEmail,
		"resume_path":     user.ResumePath,
		"has_google_ai_key": user.OpenAIKey != "",
	})
}

func (h *Handler) updateSettings(c *gin.Context) {
	var req struct {
		Name           string `json:"name"`
		Email          string `json:"email"`
		LinkedInEmail  string `json:"linkedin_email"`
		LinkedInPass   string `json:"linkedin_password"`
		GoogleAIKey    string `json:"google_ai_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	db.DB.First(&user)

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.LinkedInEmail != "" {
		updates["linkedin_email"] = req.LinkedInEmail
	}
	if req.LinkedInPass != "" {
		updates["linkedin_password"] = req.LinkedInPass
	}
	if req.GoogleAIKey != "" {
		updates["openai_key"] = req.GoogleAIKey
	}

	db.DB.Model(&user).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "settings updated"})
}

func (h *Handler) getAnalytics(c *gin.Context) {
	type CountResult struct {
		TotalJobs       int64
		NewJobs         int64
		AppliedJobs     int64
		FailedApps      int64
		SuccessApps     int64
		PlatformStats   []map[string]interface{}
		TopCompanies    []map[string]interface{}
	}

	var result CountResult
	db.DB.Model(&models.Job{}).Count(&result.TotalJobs)
	db.DB.Model(&models.Job{}).Where("status = ?", "new").Count(&result.NewJobs)
	db.DB.Model(&models.Job{}).Where("status = ?", "applied").Count(&result.AppliedJobs)

	db.DB.Model(&models.Application{}).Where("status = ?", "success").Count(&result.SuccessApps)
	db.DB.Model(&models.Application{}).Where("status = ?", "failed").Count(&result.FailedApps)

	c.JSON(http.StatusOK, result)
}
