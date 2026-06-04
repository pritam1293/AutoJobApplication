package scraper

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/jobhaunt/backend/internal/models"
)

type LinkedInScraper struct {
	email    string
	password string
}

func NewLinkedInScraper(email, password string) *LinkedInScraper {
	return &LinkedInScraper{
		email:    email,
		password: password,
	}
}

func (s *LinkedInScraper) Name() string {
	return "linkedin"
}

func (s *LinkedInScraper) Search(ctx context.Context, query string, location string) ([]JobResult, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := s.login(ctx); err != nil {
		return nil, fmt.Errorf("linkedin login failed: %w", err)
	}

	encodedQuery := url.QueryEscape(query)
	encodedLocation := url.QueryEscape(location)
	searchURL := fmt.Sprintf("https://www.linkedin.com/jobs/search/?keywords=%s&location=%s", encodedQuery, encodedLocation)

	var jobsHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible("div.jobs-search-results-list", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Scroll down to load more jobs
			for i := 0; i < 5; i++ {
				chromedp.Evaluate(`window.scrollBy(0, 800)`, nil).Do(ctx)
				time.Sleep(1 * time.Second)
			}
			return nil
		}),
		chromedp.OuterHTML("div.jobs-search-results-list", &jobsHTML, chromedp.ByQuery),
	)
	if err != nil {
		return nil, fmt.Errorf("linkedin search navigation failed: %w", err)
	}

	jobs := s.parseJobListings(jobsHTML, searchURL)

	log.Printf("linkedin: found %d jobs for query '%s' in '%s'", len(jobs), query, location)
	return jobs, nil
}

func (s *LinkedInScraper) login(ctx context.Context) error {
	var currentURL string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.linkedin.com/login"),
		chromedp.WaitVisible("#username", chromedp.ByQuery),
		chromedp.SendKeys("#username", s.email, chromedp.ByQuery),
		chromedp.SendKeys("#password", s.password, chromedp.ByQuery),
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

func (s *LinkedInScraper) parseJobListings(html string, baseURL string) []JobResult {
	var jobs []JobResult

	// Parse job cards from the HTML
	// We extract job IDs, titles, companies, and locations via chromedp evaluation
	// This is a simplified extraction; in production use a more robust DOM parsing

	// For now, we return an empty result and rely on GetJobDetails for individual job data
	return jobs
}

func (s *LinkedInScraper) GetJobDetails(ctx context.Context, job *models.Job) error {
	// Navigate to a specific job posting and extract full details
	return nil
}

// GetJobsViaAPI is a helper that extracts job listings by evaluating JavaScript
func (s *LinkedInScraper) GetJobsViaAPI(ctx context.Context, query, location string) ([]JobResult, error) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if err := s.login(ctx); err != nil {
		return nil, fmt.Errorf("linkedin login failed: %w", err)
	}

	searchURL := fmt.Sprintf("https://www.linkedin.com/jobs/search/?keywords=%s&location=%s",
		url.QueryEscape(query), url.QueryEscape(location))

	var jobs []JobResult
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible("div.jobs-search-results-list", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Evaluate(`
			(() => {
				const cards = document.querySelectorAll('.job-search-card');
				const results = [];
				cards.forEach(card => {
					const titleEl = card.querySelector('.job-search-card__title');
					const companyEl = card.querySelector('.job-search-card__subtitle');
					const locationEl = card.querySelector('.job-search-card__location');
					const linkEl = card.querySelector('a');
					if (titleEl && companyEl) {
						results.push({
							title: titleEl.innerText.trim(),
							company: companyEl.innerText.trim(),
							location: locationEl ? locationEl.innerText.trim() : '',
							url: linkEl ? linkEl.href : '',
							jobId: card.getAttribute('data-job-id') || ''
						});
					}
				});
				return JSON.stringify(results);
			})()
		`, &jobs),
	)
	if err != nil {
		return nil, fmt.Errorf("linkedin DOM extraction failed: %w", err)
	}

	return jobs, nil
}
