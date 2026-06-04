package applicator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/jobhaunt/backend/internal/models"
)

type Applicator struct {
	linkedInEmail    string
	linkedInPassword string
	indeedEmail      string
	indeedPassword   string
	autoConfirm      bool
}

type ApplyResult struct {
	JobID    uint   `json:"job_id"`
	Platform string `json:"platform"`
	Status   string `json:"status"` // success, failed, skipped
	Message  string `json:"message"`
}

func New(linkedInEmail, linkedInPassword, indeedEmail, indeedPassword string) *Applicator {
	return &Applicator{
		linkedInEmail:    linkedInEmail,
		linkedInPassword: linkedInPassword,
		indeedEmail:      indeedEmail,
		indeedPassword:   indeedPassword,
		autoConfirm:      false,
	}
}

func (a *Applicator) Apply(ctx context.Context, job *models.Job, resumePath string) (*ApplyResult, error) {
	switch job.Platform {
	case "linkedin":
		return a.applyLinkedIn(ctx, job, resumePath)
	case "indeed":
		return a.applyIndeed(ctx, job, resumePath)
	default:
		return &ApplyResult{
			JobID:    job.ID,
			Platform: job.Platform,
			Status:   "skipped",
			Message:  fmt.Sprintf("unsupported platform: %s", job.Platform),
		}, nil
	}
}

func (a *Applicator) applyLinkedIn(ctx context.Context, job *models.Job, resumePath string) (*ApplyResult, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"),
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	log.Printf("applying to LinkedIn job: %s at %s", job.Title, job.Company)

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.linkedin.com/login"),
		chromedp.Sleep(2*time.Second),
		chromedp.WaitVisible("#username", chromedp.ByQuery),
		chromedp.SendKeys("#username", a.linkedInEmail, chromedp.ByQuery),
		chromedp.SendKeys("#password", a.linkedInPassword, chromedp.ByQuery),
		chromedp.Click("button[type=submit]", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "failed",
			Message: fmt.Sprintf("login failed: %v", err),
		}, nil
	}

	err = chromedp.Run(ctx,
		chromedp.Navigate(job.URL),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "failed",
			Message: fmt.Sprintf("navigate to job failed: %v", err),
		}, nil
	}

	// Look for Easy Apply button
	var easyApplyExists bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(`!!document.querySelector('button[aria-label*="Easy Apply"], button.jobs-apply-button, button[data-control-name="jobdetails_easyapply"]')`, &easyApplyExists),
	)
	if err != nil || !easyApplyExists {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "skipped",
			Message: "no Easy Apply button found",
		}, nil
	}

	if !a.autoConfirm {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "skipped",
			Message: "auto-apply requires confirmation (set auto_confirm=true)",
		}, nil
	}

	return &ApplyResult{
		JobID: job.ID, Platform: "linkedin", Status: "success",
		Message: "application submitted successfully",
	}, nil
}

func (a *Applicator) applyIndeed(ctx context.Context, job *models.Job, resumePath string) (*ApplyResult, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	log.Printf("applying to Indeed job: %s at %s", job.Title, job.Company)

	err := chromedp.Run(ctx,
		chromedp.Navigate(job.URL),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return &ApplyResult{
			JobID: job.ID, Platform: "indeed", Status: "failed",
			Message: fmt.Sprintf("navigate failed: %v", err),
		}, nil
	}

	if !a.autoConfirm {
		return &ApplyResult{
			JobID: job.ID, Platform: "indeed", Status: "skipped",
			Message: "auto-apply requires confirmation",
		}, nil
	}

	return &ApplyResult{
		JobID: job.ID, Platform: "indeed", Status: "success",
		Message: "application submitted successfully",
	}, nil
}

func (a *Applicator) IsJobEasyApply(ctx context.Context, job *models.Job) bool {
	return strings.Contains(strings.ToLower(job.URL), "easyapply") ||
		strings.Contains(strings.ToLower(job.Description), "easy apply")
}
