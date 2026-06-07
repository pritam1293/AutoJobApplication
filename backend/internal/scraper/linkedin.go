package scraper

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/jobhaunt/backend/internal/models"
)

type LinkedInScraper struct {
	email       string
	password    string
	mu          sync.Mutex
	allocCtx    context.Context
	allocCancel context.CancelFunc
	loggedIn    bool
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

func (s *LinkedInScraper) allocOpts() []chromedp.ExecAllocatorOption {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),
		chromedp.Flag("disable-notifications", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"),
	)
	return opts
}

func (s *LinkedInScraper) ensureAllocator(ctx context.Context) (context.Context, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.allocCtx == nil {
		allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, s.allocOpts()...)
		s.allocCtx = allocCtx
		s.allocCancel = allocCancel

		if s.email != "" && s.password != "" {
			browserCtx, browserCancel := chromedp.NewContext(allocCtx)
			defer browserCancel()

			loginCtx, loginCancel := context.WithTimeout(browserCtx, 45*time.Second)
			defer loginCancel()

			if err := s.login(loginCtx); err != nil {
				log.Printf("linkedin login failed (continuing without auth): %v", err)
			} else {
				s.loggedIn = true
				log.Println("linkedin login successful")
			}
		} else {
			log.Println("linkedin: no credentials provided, running without login")
		}
	}

	return s.allocCtx, nil
}

func (s *LinkedInScraper) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.allocCancel != nil {
		s.allocCancel()
		s.allocCtx = nil
		s.allocCancel = nil
		s.loggedIn = false
	}
}

func (s *LinkedInScraper) Search(ctx context.Context, query string, location string) ([]JobResult, error) {
	allocCtx, err := s.ensureAllocator(ctx)
	if err != nil {
		return nil, err
	}

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	searchCtx, searchCancel := context.WithTimeout(browserCtx, 90*time.Second)
	defer searchCancel()

	encodedQuery := url.QueryEscape(query)
	encodedLocation := url.QueryEscape(location)
	searchURL := fmt.Sprintf("https://www.linkedin.com/jobs/search/?keywords=%s&location=%s", encodedQuery, encodedLocation)

	var jobs []JobResult
	err = chromedp.Run(searchCtx,
		chromedp.Emulate(device.IPhoneXR),
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < 5; i++ {
				chromedp.Evaluate(`window.scrollBy(0, 800)`, nil).Do(ctx)
				time.Sleep(1 * time.Second)
			}
			return nil
		}),
		chromedp.Evaluate(`
			(() => {
				const cards = document.querySelectorAll('li[data-occludable-job-id]');
				const results = [];
				cards.forEach(card => {
					const titleEl = card.querySelector('a.job-card-list__title, strong');
					const companyEl = card.querySelector('.job-card-container__company-name, a.job-card-container__company-name');
					const locationEl = card.querySelector('.job-card-container__metadata-item');
					const linkEl = card.querySelector('a.job-card-list__title');
					const jobId = card.getAttribute('data-occludable-job-id') || '';
					if (titleEl && companyEl) {
						const href = linkEl ? linkEl.getAttribute('href') || '' : '';
						const fullUrl = href.startsWith('http') ? href : 'https://www.linkedin.com' + href;
						results.push({
							job_id: jobId,
							title: (titleEl.innerText || titleEl.textContent || '').trim(),
							company: (companyEl.innerText || companyEl.textContent || '').trim(),
							location: locationEl ? (locationEl.innerText || locationEl.textContent || '').trim() : '',
							url: fullUrl
						});
					}
				});

				if (results.length === 0) {
					const fallbackCards = document.querySelectorAll('.job-card-container, .job-search-card, article');
					fallbackCards.forEach(card => {
						const titleEl = card.querySelector('a, h3, strong');
						const companyEl = card.querySelector('.company-name, .job-card-container__company-name, span');
						if (titleEl) {
							results.push({
								job_id: card.getAttribute('data-job-id') || String(Math.random()),
								title: (titleEl.innerText || titleEl.textContent || '').trim(),
								company: companyEl ? (companyEl.innerText || companyEl.textContent || '').trim() : '',
								location: '',
								url: titleEl.getAttribute('href') ? (titleEl.getAttribute('href').startsWith('http') ? titleEl.getAttribute('href') : 'https://www.linkedin.com' + titleEl.getAttribute('href')) : ''
							});
						}
					});
				}

				return results;
			})()
		`, &jobs),
	)
	if err != nil {
		return nil, fmt.Errorf("linkedin search failed: %w", err)
	}

	for i := range jobs {
		jobs[i].Platform = "linkedin"
	}

	log.Printf("linkedin: found %d jobs for query '%s' in '%s'", len(jobs), query, location)
	return jobs, nil
}

func (s *LinkedInScraper) login(ctx context.Context) error {
	var currentURL string
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.linkedin.com/login"),
		chromedp.WaitVisible("#username", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.Evaluate(`document.querySelector("#username").value = ""`, nil).Do(ctx)
			return nil
		}),
		chromedp.SendKeys("#username", s.email, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.Evaluate(`document.querySelector("#password").value = ""`, nil).Do(ctx)
			return nil
		}),
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

	if strings.Contains(currentURL, "login") {
		return fmt.Errorf("linkedin login failed - still on login page")
	}

	return nil
}

func (s *LinkedInScraper) GetJobDetails(ctx context.Context, job *models.Job) error {
	allocCtx, err := s.ensureAllocator(ctx)
	if err != nil {
		return err
	}

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	detailCtx, detailCancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer detailCancel()

	var detail struct {
		Description string `json:"description"`
	}
	err = chromedp.Run(detailCtx,
		chromedp.Navigate(job.URL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`
			(() => {
				const descEl = document.querySelector('.jobs-description-content__text, .jobs-box__html-content, .description, .show-more-less-html, article');
				return {
					description: descEl ? (descEl.innerText || descEl.textContent || '').trim() : ''
				};
			})()
		`, &detail),
	)
	if err != nil {
		return fmt.Errorf("linkedin get details failed: %w", err)
	}

	job.Description = detail.Description
	return nil
}
