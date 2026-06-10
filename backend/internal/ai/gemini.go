package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/genai"
)

type Client struct {
	client *genai.Client
	models map[string]string
}

type TailorRequest struct {
	ResumeData   string `json:"resume_data"`
	JobTitle     string `json:"job_title"`
	Company      string `json:"company"`
	JobDesc      string `json:"job_desc"`
	Skills       string `json:"skills"`
	Instructions string `json:"instructions"`
}

type TailorResponse struct {
	TailoredResume string  `json:"tailored_resume"`
	MatchScore     float64 `json:"match_score"`
	MissingSkills  string  `json:"missing_skills"`
	Notes          string  `json:"notes"`
}

type SkillExtract struct {
	Skills     []string `json:"skills"`
	Experience string   `json:"experience"`
	Education  string   `json:"education"`
}

func NewClient(apiKey string) *Client {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return &Client{models: map[string]string{
			"default":  "gemini-2.5-flash",
			"light":    "gemini-2.5-flash",
			"thinking": "gemini-2.5-pro",
		}}
	}

	return &Client{
		client: client,
		models: map[string]string{
			"default":  "gemini-2.5-flash",
			"light":    "gemini-2.5-flash",
			"thinking": "gemini-2.5-pro",
		},
	}
}

func (c *Client) TailorResume(ctx context.Context, req TailorRequest) (*TailorResponse, error) {
	if c.client == nil {
		return nil, fmt.Errorf("gemini client not initialized: no API key configured")
	}

	prompt := fmt.Sprintf(`You are an expert ATS resume tailor. Given a candidate's base resume and a job description, update ONLY the "Technical Skills" and "Projects" sections. Keep ALL other sections (header, Education, Work Experience, Achievements, Relevant Coursework, etc.) EXACTLY as-is.

CRITICAL — PRESERVE THIS FORMATTING EXACTLY:
The resume uses plain text with these specific formatting conventions:
- Section headers (Education, Work Experience, Projects, etc.) appear on their own line with a blank line before.
- Sub-headings have dates/links on the SAME LINE, right-aligned (e.g., "National Institute of Technology, Rourkela November 2022 – May 2026").
- Profile links use special characters like ï, §, € as separators on the header line.
- Bullet points start with "•" — each on its own indented line.
- Multi-line bullet text should wrap naturally, NOT on separate bullet lines.
- Achievements section uses "•Header:" followed by description on the next line.
- Technical Skills section uses "•Category: item1, item2, ...".

RULES:
1. "Technical Skills" section: reorder categories/skills to put most relevant first. Add missing skills from the JD that the candidate plausibly has. Remove irrelevant ones. Keep the "•Category: skill1, skill2, ..." format.
2. "Projects" section: tweak descriptions to highlight JD keywords. Do NOT fabricate experience. Keep the "ProjectName - Subtitle Date" header format.
3. ALL other sections: copy VERBATIM — every line break, bullet, date position, special character. Change NOTHING.
4. Return the COMPLETE resume text preserving the raw formatting (newlines, spacing, indentation).
5. Do NOT rewrite, rephrase, or "improve" the formatting or structure of any section.

Base Resume:
%s

Target Job:
Title: %s
Company: %s

Job Description:
%s

Required Skills Mentioned: %s

Additional Instructions: %s

Return a JSON object with these fields:
{
  "tailored_resume": "the COMPLETE resume with only skills and projects sections updated, all other sections verbatim from the original formatting",
  "match_score": a float between 0 and 100 indicating how well the candidate matches the job,
  "missing_skills": "comma-separated list of important skills from the JD not found in the resume",
  "notes": "brief notes on what was changed and why"
}`, req.ResumeData, req.JobTitle, req.Company, req.JobDesc, req.Skills, req.Instructions)

	resp, err := c.client.Models.GenerateContent(ctx, c.models["default"], genai.Text(prompt), &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "You are an expert ATS resume optimization assistant. Always return valid JSON."}},
		},
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("gemini tailor request failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response from gemini")
	}

	content := resp.Text()
	var result TailorResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return &TailorResponse{
			TailoredResume: content,
			MatchScore:     0,
		}, nil
	}

	return &result, nil
}

func (c *Client) ExtractSkills(ctx context.Context, jd string) (*SkillExtract, error) {
	if c.client == nil {
		return nil, fmt.Errorf("gemini client not initialized: no API key configured")
	}

	prompt := fmt.Sprintf(`Extract key skills, experience requirements, and education requirements from this job description. Return as JSON.

Job Description:
%s

{
  "skills": ["skill1", "skill2", ...],
  "experience": "years and type of experience required",
  "education": "education requirements"
}`, jd)

	resp, err := c.client.Models.GenerateContent(ctx, c.models["light"], genai.Text(prompt), &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: "Extract structured data from job descriptions. Always return valid JSON."}},
		},
		ResponseMIMEType: "application/json",
	})
	if err != nil {
		return nil, fmt.Errorf("gemini skill extraction failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response from gemini")
	}

	content := resp.Text()
	var result SkillExtract
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return &SkillExtract{}, nil
	}

	return &result, nil
}
