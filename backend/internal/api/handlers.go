package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
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
		indeedScr:     scraper.NewIndeedScraper("", ""),
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
		api.GET("/jobs/:id/details", h.getJobDetails)
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
		api.POST("/settings/auto-apply", h.toggleAutoApply)

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
		Location string `json:"location"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	allJobs := make([]scraper.JobResult, 0)

	linkedinJobs, err := h.linkedinScr.Search(ctx, req.Query, req.Location)
	if err != nil {
		log.Printf("linkedin search failed (non-fatal): %v", err)
	} else {
		allJobs = append(allJobs, linkedinJobs...)
	}

	indeedJobs, err := h.indeedScr.Search(ctx, req.Query, req.Location)
	if err != nil {
		log.Printf("indeed search failed (non-fatal): %v", err)
	} else {
		allJobs = append(allJobs, indeedJobs...)
	}

	for _, j := range allJobs {
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
		"total": len(allJobs),
		"jobs":  allJobs,
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

func (h *Handler) getJobDetails(c *gin.Context) {
	id := c.Param("id")
	var job models.Job
	if err := db.DB.First(&job, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	var err error
	switch job.Platform {
	case "linkedin":
		err = h.linkedinScr.GetJobDetails(ctx, &job)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("platform %s not supported for details", job.Platform)})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	db.DB.Model(&job).Update("description", job.Description)

	c.JSON(http.StatusOK, gin.H{
		"id":          job.ID,
		"description": job.Description,
	})
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

	user := h.getOrCreateUser()

	// Step 1: Get job details if description is empty
	if job.Description == "" {
		detailCtx, detailCancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		if job.Platform == "linkedin" {
			if err := h.linkedinScr.GetJobDetails(detailCtx, &job); err == nil && job.Description != "" {
				db.DB.Model(&job).Update("description", job.Description)
			}
		}
		detailCancel()
	}

	resumePath := user.ResumePath

	// Step 2: Tailor resume if AI is configured and description exists
	if h.aiClient != nil && job.Description != "" && user.ResumePath != "" {
		parsed, err := resume.ParsePDF(user.ResumePath)
		if err == nil && parsed.RawText != "" {
			tailored, err := h.tailorEngine.TailorForJob(c.Request.Context(), parsed, &job, "")
			if err == nil && tailored.TailoredResume != "" {
				// Generate tailored PDF
				tailoredPath := strings.Replace(user.ResumePath, ".pdf", "-tailored-"+fmt.Sprint(job.ID)+".pdf", 1)
				if err := resume.GeneratePDF(tailored.TailoredResume, tailoredPath); err == nil {
					resumePath = tailoredPath
					log.Printf("generated tailored resume at %s", tailoredPath)
				}
			}
		}
	}

	// Step 3: Apply
	result, err := h.applicator.Apply(c.Request.Context(), &job, resumePath)
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

	user := h.getOrCreateUser()
	db.DB.Model(&user).Update("resume_path", savePath)

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

	// Get job description if empty
	if job.Description == "" {
		detailCtx, detailCancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		if job.Platform == "linkedin" {
			if err := h.linkedinScr.GetJobDetails(detailCtx, &job); err == nil && job.Description != "" {
				db.DB.Model(&job).Update("description", job.Description)
			}
		}
		detailCancel()
	}

	// Read user's uploaded resume
	user := h.getOrCreateUser()
	if user.ResumePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no resume uploaded. go to Resume page first"})
		return
	}

	parsed, err := resume.ParsePDF(user.ResumePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to parse resume: %v", err)})
		return
	}

	if parsed.RawText == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "resume PDF is empty or unreadable"})
		return
	}

	var req struct {
		Instructions string `json:"instructions"`
	}
	c.ShouldBindJSON(&req)

	resp, err := h.tailorEngine.TailorForJob(c.Request.Context(), parsed, &job, req.Instructions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate tailored PDF
	baseName := strings.TrimSuffix(user.ResumePath, ".pdf")
	tailoredPDFPath := fmt.Sprintf("%s-tailored-%d.pdf", baseName, job.ID)
	if err := resume.GeneratePDF(resp.TailoredResume, tailoredPDFPath); err != nil {
		log.Printf("warning: failed to generate tailored PDF: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"tailored_resume": resp.TailoredResume,
		"match_score":     resp.MatchScore,
		"missing_skills":  resp.MissingSkills,
		"notes":           resp.Notes,
		"tailored_pdf":    tailoredPDFPath,
	})
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

func (h *Handler) getOrCreateUser() models.User {
	var user models.User
	result := db.DB.First(&user)
	if result.Error != nil {
		user = models.User{Name: "Default User"}
		db.DB.Create(&user)
	}
	return user
}

func (h *Handler) getSettings(c *gin.Context) {
	user := h.getOrCreateUser()
	c.JSON(http.StatusOK, gin.H{
		"name":              user.Name,
		"email":             user.Email,
		"linkedin_email":    user.LinkedInEmail,
		"resume_path":       user.ResumePath,
		"has_google_ai_key": user.OpenAIKey != "",
	})
}

func (h *Handler) updateSettings(c *gin.Context) {
	var req struct {
		Name          string `json:"name"`
		Email         string `json:"email"`
		LinkedInEmail string `json:"linkedin_email"`
		LinkedInPass  string `json:"linkedin_password"`
		GoogleAIKey   string `json:"google_ai_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := h.getOrCreateUser()

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

func (h *Handler) StartScheduler(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		h.runScheduledSearches(ctx)

		for {
			select {
			case <-ticker.C:
				h.runScheduledSearches(ctx)
			case <-ctx.Done():
				log.Println("scheduler stopped")
				return
			}
		}
	}()
	log.Printf("scheduler started with interval %s", interval)
}

func (h *Handler) runScheduledSearches(ctx context.Context) {
	var queries []models.SearchQuery
	db.DB.Where("active = ?", true).Find(&queries)

	for _, q := range queries {
		log.Printf("running scheduled search: %s in %s", q.Query, q.Location)

		searchCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		jobs, err := h.linkedinScr.Search(searchCtx, q.Query, q.Location)
		cancel()

		if err != nil {
			log.Printf("scheduled search failed for query '%s': %v", q.Query, err)
			continue
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

		if q.AutoApply && len(jobs) > 0 {
			count := 0
			for _, j := range jobs {
				if count >= q.MaxApplied {
					break
				}

				var job models.Job
				db.DB.Where("job_id = ? AND status = ?", j.JobID, "new").First(&job)
				if job.ID == 0 {
					continue
				}

				user := h.getOrCreateUser()

				result, err := h.applicator.Apply(ctx, &job, user.ResumePath)
				if err != nil {
					log.Printf("auto-apply failed for job %s: %v", j.Title, err)
					continue
				}

				app := models.Application{
					JobID:    job.ID,
					UserID:   user.ID,
					Status:   result.Status,
					AppliedAt: time.Now(),
					Notes:    result.Message,
				}
				db.DB.Create(&app)

				if result.Status == "success" {
					db.DB.Model(&job).Update("status", "applied")
					count++
				}
			}
			log.Printf("auto-applied to %d/%d jobs for query '%s'", count, len(jobs), q.Query)
		}
	}
}

func (h *Handler) toggleAutoApply(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.applicator.SetAutoConfirm(req.Enabled)
	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}
	log.Printf("auto-apply %s", status)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("auto-apply %s", status), "enabled": req.Enabled})
}

func (h *Handler) getAnalytics(c *gin.Context) {
	type CountResult struct {
		TotalJobs   int64
		NewJobs     int64
		AppliedJobs int64
		FailedApps  int64
		SuccessApps int64
	}

	var result CountResult
	db.DB.Model(&models.Job{}).Count(&result.TotalJobs)
	db.DB.Model(&models.Job{}).Where("status = ?", "new").Count(&result.NewJobs)
	db.DB.Model(&models.Job{}).Where("status = ?", "applied").Count(&result.AppliedJobs)

	db.DB.Model(&models.Application{}).Where("status = ?", "success").Count(&result.SuccessApps)
	db.DB.Model(&models.Application{}).Where("status = ?", "failed").Count(&result.FailedApps)

	c.JSON(http.StatusOK, result)
}
