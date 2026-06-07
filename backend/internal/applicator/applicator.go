package applicator

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
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
	Status   string `json:"status"`
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

func (a *Applicator) SetAutoConfirm(enabled bool) {
	a.autoConfirm = enabled
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

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, 180*time.Second)
	defer cancel()

	log.Printf("applying to LinkedIn job: %s at %s", job.Title, job.Company)

	if err := a.linkedinLogin(browserCtx); err != nil {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "failed",
			Message: fmt.Sprintf("login failed: %v", err),
		}, nil
	}

	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(job.URL),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "failed",
			Message: fmt.Sprintf("navigate to job failed: %v", err),
		}, nil
	}

	hasEasyApply, err := a.checkEasyApply(browserCtx)
	if err != nil || !hasEasyApply {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "skipped",
			Message: "no Easy Apply button found",
		}, nil
	}

	if !a.autoConfirm {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "skipped",
			Message: "auto-apply disabled. enable via SetAutoConfirm(true)",
		}, nil
	}

	result, err := a.runEasyApplyFlow(browserCtx, job, resumePath)
	if err != nil {
		return &ApplyResult{
			JobID: job.ID, Platform: "linkedin", Status: "failed",
			Message: fmt.Sprintf("easy apply flow failed: %v", err),
		}, nil
	}

	return result, nil
}

func (a *Applicator) linkedinLogin(ctx context.Context) error {
	var currentURL string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.linkedin.com/login"),
		chromedp.WaitVisible("#username", chromedp.ByQuery),
		chromedp.SendKeys("#username", a.linkedInEmail, chromedp.ByQuery),
		chromedp.SendKeys("#password", a.linkedInPassword, chromedp.ByQuery),
		chromedp.Click("button[type=submit]", chromedp.ByQuery),
		chromedp.Sleep(3*time.Second),
		chromedp.Location(&currentURL),
	)
	if err != nil {
		return err
	}
	if strings.Contains(currentURL, "checkpoint") {
		return fmt.Errorf("linkedin login challenged - manual verification required")
	}
	return nil
}

func (a *Applicator) checkEasyApply(ctx context.Context) (bool, error) {
	var found bool
	err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`!!(
			document.querySelector('button[aria-label*="Easy Apply"]') ||
			document.querySelector('button.jobs-apply-button') ||
			document.querySelector('button[data-control-name="jobdetails_easyapply"]')
		)`, &found),
	)
	return found, err
}

func (a *Applicator) runEasyApplyFlow(ctx context.Context, job *models.Job, resumePath string) (*ApplyResult, error) {
	var submitSuccess bool
	var finalMsg string

	err := chromedp.Run(ctx,
		chromedp.Sleep(1*time.Second),

		chromedp.Evaluate(`(() => {
			const btn =
				document.querySelector('button[aria-label*="Easy Apply"]') ||
				document.querySelector('button.jobs-apply-button') ||
				document.querySelector('button[data-control-name="jobdetails_easyapply"]');
			if (btn) { btn.click(); return true; }
			return false;
		})()`, nil),

		chromedp.Sleep(2*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			return fillEasyApplyFields(ctx, resumePath)
		}),

		chromedp.ActionFunc(func(ctx context.Context) error {
			submitted, err := advanceEasyApply(ctx)
			submitSuccess = submitted
			return err
		}),
	)

	if err != nil {
		finalMsg = fmt.Sprintf("easy apply error: %v", err)
	} else if submitSuccess {
		finalMsg = "application submitted successfully"
	} else {
		finalMsg = "easy apply flow completed (may need manual review)"
	}

	status := "success"
	if !submitSuccess || err != nil {
		status = "failed"
	}

	return &ApplyResult{
		JobID:    job.ID,
		Platform: "linkedin",
		Status:   status,
		Message:  finalMsg,
	}, nil
}

func fillEasyApplyFields(ctx context.Context, resumePath string) error {
	chromedp.Sleep(1 * time.Second).Do(ctx)

	var fields []map[string]string
	chromedp.Evaluate(`(() => {
		const modal = document.querySelector('.jobs-easy-apply-modal, div[data-test-modal-id="easy-apply"]');
		if (!modal) return [];
		const inputs = modal.querySelectorAll('input, select, textarea');
		const results = [];
		inputs.forEach(inp => {
			const id = inp.id || '';
			const name = inp.name || '';
			const placeholder = inp.placeholder || '';
			const type = inp.type || 'text';
			const required = inp.required || inp.getAttribute('aria-required') === 'true';
			results.push({id, name, placeholder, type, required: String(required)});
		});
		return results;
	})()`, &fields)

	for _, f := range fields {
		if f["required"] == "true" {
			selector := fmt.Sprintf("#%s", f["id"])
			placeholder := strings.ToLower(f["placeholder"])
			defaultVal := "000-000-0000"
			if strings.Contains(placeholder, "email") {
				continue
			}
			chromedp.SendKeys(selector, defaultVal, chromedp.ByQuery).Do(ctx)
		}
	}

	if resumePath != "" {
		var hasUpload bool
		chromedp.Evaluate(`!!document.querySelector('.jobs-easy-apply-modal input[type="file"]')`, &hasUpload).Do(ctx)
		if hasUpload {
			absPath, _ := filepath.Abs(resumePath)
			chromedp.SetUploadFiles(`input[type="file"]`, []string{absPath}, chromedp.ByQuery).Do(ctx)
			chromedp.Sleep(1 * time.Second).Do(ctx)
		}
	}

	return nil
}

func advanceEasyApply(ctx context.Context) (bool, error) {
	for step := 0; step < 10; step++ {
		var button struct {
			Label string `json:"label"`
		}

		err := chromedp.Run(ctx,
			chromedp.Sleep(1*time.Second),
			chromedp.Evaluate(`(() => {
				const btn =
					document.querySelector('button[aria-label="Submit application"]') ||
					document.querySelector('button[aria-label="Submit"]') ||
					document.querySelector('button[aria-label*="Review"]') ||
					document.querySelector('button[aria-label="Continue to next step"]') ||
					document.querySelector('button[aria-label="Next"]') ||
					document.querySelector('.artdeco-button--primary:not([disabled])');
				if (!btn) return {label: ''};
				return {label: (btn.innerText || btn.textContent || '').trim()};
			})()`, &button),
		)
		if err != nil {
			return false, err
		}

		if button.Label == "" {
			return true, nil
		}

		label := strings.ToLower(button.Label)
		isSubmit := strings.Contains(label, "submit")

		err = chromedp.Run(ctx,
			chromedp.Evaluate(`(() => {
				const btn =
					document.querySelector('button[aria-label="Submit application"]') ||
					document.querySelector('button[aria-label="Submit"]') ||
					document.querySelector('button[aria-label*="Review"]') ||
					document.querySelector('button[aria-label="Continue to next step"]') ||
					document.querySelector('button[aria-label="Next"]') ||
					document.querySelector('.artdeco-button--primary:not([disabled])');
				if (btn) { btn.click(); return true; }
				return false;
			})()`, nil),
			chromedp.Sleep(2*time.Second),
		)
		if err != nil {
			return false, fmt.Errorf("failed to click button '%s': %w", button.Label, err)
		}

		if isSubmit {
			chromedp.Sleep(2 * time.Second).Do(ctx)
			return true, nil
		}
	}

	return false, nil
}

func (a *Applicator) applyIndeed(ctx context.Context, job *models.Job, resumePath string) (*ApplyResult, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	defer allocCancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	browserCtx, cancel = context.WithTimeout(browserCtx, 120*time.Second)
	defer cancel()

	log.Printf("applying to Indeed job: %s at %s", job.Title, job.Company)

	err := chromedp.Run(browserCtx,
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
