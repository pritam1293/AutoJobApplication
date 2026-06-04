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
	prompt := fmt.Sprintf(`You are an expert resume tailor. Given a candidate's base resume data and a job description, tailor the resume to maximize ATS (Applicant Tracking System) compatibility and match the job requirements.

Base Resume Data:
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
  "tailored_resume": "the fully tailored resume in markdown format, optimized with relevant keywords from the JD, reordered skills, and tailored experience descriptions",
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
