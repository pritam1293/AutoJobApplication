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
	var sampleHTML string
	var debugText string

	err = chromedp.Run(searchCtx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible("div[data-view-name*='search'], div[class*='search'], div[class*='jobs-search'], div.jobs-search-results-list", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for i := 0; i < 5; i++ {
				chromedp.Evaluate(`window.scrollBy(0, 800)`, nil).Do(ctx)
				chromedp.Sleep(1 * time.Second).Do(ctx)
			}
			return nil
		}),
		chromedp.OuterHTML("html", &sampleHTML, chromedp.ByQuery),
		chromedp.Evaluate(`(() => {
				const cards = document.querySelectorAll('li[data-occludable-job-id]');
				let debug = 'li[data-occludable-job-id] count: ' + cards.length + '\\n';
				cards.forEach((c, i) => { if (i < 3) debug += 'CARD ' + i + ': ' + (c.innerText || '').replace(/\\s+/g, ' ').substring(0, 300) + '\\n'; });

				if (cards.length === 0) {
					document.querySelectorAll('[class*="base-card"], [class*="job-card"], [class*="search-result"], article').forEach((c, i) => {
						if (i < 3) {
							const t = c.querySelector('[class*="card__title"]');
							const l = c.querySelector('[class*="card__location"]');
							const co = c.querySelector('[class*="card__subtitle"]');
							debug += 'FB' + i + ': class=' + (c.className || '').substring(0, 80) + '\\n';
							debug += '  title=' + (t ? (t.innerText || '').trim() : 'NOTFOUND') + '\\n';
							debug += '  company=' + (co ? (co.innerText || '').trim() : 'NOTFOUND') + '\\n';
							debug += '  location=' + (l ? (l.innerText || '').trim() : 'NOTFOUND') + '\\n';
						}
					});
				}
				return debug;
			})()`, &debugText),

		chromedp.Evaluate(`
			(() => {
				const results = [];
				const seen = new Set();

				const _href = function(el) {
					if (!el) return '';
					const h = el.getAttribute('href') || '';
					return h.startsWith('http') ? h : 'https://www.linkedin.com' + h;
				};

				const extract = function(root) {
					const titleEl = root.querySelector('[class*="card__title"]') || root.querySelector('a[href*="/jobs/"]');
					const locEl = root.querySelector('[class*="card__location"]') || root.querySelector('[class*="location"]');
					const companyEl = root.querySelector('[class*="card__subtitle"]') || root.querySelector('[class*="company"]') || root.querySelector('[class*="org"]');

					const title = (titleEl && (titleEl.innerText || '').trim()) || '';
					if (!title || seen.has(title)) return;
					seen.add(title);

					const company = (companyEl && (companyEl.innerText || '').trim()) || '';
					const location = (locEl && (locEl.innerText || '').trim()) || '';

					let url = '';
					if (titleEl && titleEl.tagName === 'A') {
						url = _href(titleEl);
					} else if (root.tagName === 'A') {
						url = _href(root);
					} else {
						const a = root.querySelector('a[href*="/jobs/"]');
						if (a) url = _href(a);
					}

					const jobId = root.getAttribute('data-occludable-job-id') || root.getAttribute('data-job-id') || String(Math.random());
					results.push({ job_id: jobId, title, company, location, url });
				};

				const cards = document.querySelectorAll('li[data-occludable-job-id]');
				if (cards.length > 0) {
					cards.forEach(extract);
				} else {
					document.querySelectorAll('[class*="base-card"], [class*="job-card"], [class*="search-result"], article').forEach(extract);
				}

				return results;
			})()
		`, &jobs),
	)
	if err != nil {
		if sampleHTML != "" {
			log.Printf("linkedin search failed but got HTML (len=%d)", len(sampleHTML))
		}
		return nil, fmt.Errorf("linkedin search failed: %w", err)
	}

	log.Printf("linkedin debug: %s", debugText)
	log.Printf("linkedin page HTML length: %d, jobs extracted: %d", len(sampleHTML), len(jobs))

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

	var info struct {
		Description string `json:"description"`
		Debug       string `json:"debug"`
	}

	err = chromedp.Run(detailCtx,
		chromedp.Navigate(job.URL),
		chromedp.Sleep(3*time.Second),
		chromedp.Evaluate(`(() => {
			const getText = (sel) => {
				const el = document.querySelector(sel);
				return el ? (el.innerText || el.textContent || '').trim() : '';
			};

			const selectors = [
				'.jobs-description-content__text',
				'.jobs-box__html-content',
				'.show-more-less-html',
				'article',
				'[class*="description"]',
				'[class*="Description"]',
				'main',
			];

			let description = '';
			for (const sel of selectors) {
				description = getText(sel);
				if (description.length > 100) break;
			}

			if (!description || description.length < 50) {
				const main = document.querySelector('main') || document.body;
				description = (main.innerText || '').trim();
			}

			if (description.length > 10000) {
				description = description.substring(0, 10000);
			}

			let debug = '';
			selectors.forEach(sel => {
				const t = getText(sel);
				debug += sel + ': ' + t.substring(0, 120).replace(/\\n/g, ' ') + '\\n';
			});

			return {description, debug};
		})()`, &info),
	)
	if err != nil {
		return fmt.Errorf("linkedin get details failed: %w", err)
	}

	log.Printf("linkedin detail debug for %s: %s", job.Title, info.Debug)

	if len(info.Description) > 0 {
		job.Description = info.Description
		log.Printf("linkedin: got description for %s (%d chars)", job.Title, len(job.Description))
	} else {
		log.Printf("linkedin: no description found for %s", job.Title)
	}

	return nil
}
