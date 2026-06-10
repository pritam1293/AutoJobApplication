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

type IndeedScraper struct {
	email    string
	password string
}

func NewIndeedScraper(email, password string) *IndeedScraper {
	return &IndeedScraper{
		email:    email,
		password: password,
	}
}

func (s *IndeedScraper) Name() string {
	return "indeed"
}

func (s *IndeedScraper) Search(ctx context.Context, query string, location string) ([]JobResult, error) {
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

	loc := location
	if loc == "" {
		loc = "India"
	}
	isIndia := strings.EqualFold(loc, "India") || strings.Contains(strings.ToLower(loc), "india")

	q := strings.ToLower(query)
	if strings.Contains(q, "software engineer") || strings.Contains(q, "software development engineer") || strings.Contains(q, "sde") {
		query = "(software engineer OR software development engineer OR sde)"
	}

	encodedQuery := url.QueryEscape(query)
	encodedLocation := url.QueryEscape(loc)
	baseURL := "https://www.indeed.com"
	if isIndia {
		baseURL = "https://www.indeed.co.in"
	}
	searchURL := fmt.Sprintf("%s/jobs?q=%s&l=%s&explvl=entry_level", baseURL, encodedQuery, encodedLocation)

	var jobCards []JobResult
	var pageHTML string

	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(3*time.Second),
		chromedp.WaitVisible("div.job_seen_beacon, div.jobsearch-SerpJobCard, div.cardOutline", chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &pageHTML, chromedp.ByQuery),
		chromedp.Evaluate(`
			(() => {
				const cards = document.querySelectorAll('.job_seen_beacon, .jobsearch-SerpJobCard, .cardOutline, div[data-testid="job-card"]');
				const results = [];
				cards.forEach(card => {
					const titleEl = card.querySelector('h2.jobTitle a, a.jobtitle, a[data-testid="job-card-title"]');
					const companyEl = card.querySelector('.companyName, span[data-testid="company-name"], .company');
					const locationEl = card.querySelector('.companyLocation, div[data-testid="job-card-location"]');
					const salaryEl = card.querySelector('.salary-snippet, .salaryOnly, span[data-testid="job-card-salary"]');
					if (titleEl) {
						const jobHref = titleEl.getAttribute('href') || '';
						const fullUrl = jobHref.startsWith('http') ? jobHref : 'https://www.indeed.com' + jobHref;
						results.push({
							title: titleEl.innerText.trim() || titleEl.getAttribute('title') || '',
							company: companyEl ? companyEl.innerText.trim() : '',
							location: locationEl ? locationEl.innerText.trim() : '',
							url: fullUrl,
							salary: salaryEl ? salaryEl.innerText.trim() : ''
						});
					}
				});
				return results;
			})()
		`, &jobCards),
	)
	if err != nil {
		_ = pageHTML // used for debug
		return nil, fmt.Errorf("indeed search failed: %w", err)
	}

	for i := range jobCards {
		jobCards[i].Platform = "indeed"
		parts := strings.Split(jobCards[i].URL, "jk=")
		if len(parts) > 1 {
			jobCards[i].JobID = strings.Split(parts[1], "&")[0]
		} else {
			jobCards[i].JobID = fmt.Sprintf("indeed-%d", time.Now().UnixNano())
		}
	}

	log.Printf("indeed: found %d jobs for query '%s' in '%s'", len(jobCards), query, location)
	return jobCards, nil
}

func (s *IndeedScraper) GetJobDetails(ctx context.Context, job *models.Job) error {
	// Navigate to the specific job page and extract full description
	return nil
}
