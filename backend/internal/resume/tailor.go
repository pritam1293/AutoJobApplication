package resume

import (
	"context"
	"fmt"
	"strings"

	"github.com/jobhaunt/backend/internal/ai"
	"github.com/jobhaunt/backend/internal/models"
)

type TailorEngine struct {
	aiClient *ai.Client
}

func NewTailorEngine(aiClient *ai.Client) *TailorEngine {
	return &TailorEngine{
		aiClient: aiClient,
	}
}

func (e *TailorEngine) TailorForJob(ctx context.Context, resumeData *ResumeData, job *models.Job, instructions string, latexSource string) (*ai.TailorResponse, error) {
	isLatex := strings.TrimSpace(latexSource) != "" && strings.HasPrefix(strings.TrimSpace(latexSource), `\documentclass`)
	resumeText := resumeData.RawText
	if isLatex {
		resumeText = latexSource
	}

	req := ai.TailorRequest{
		ResumeData:   resumeText,
		JobTitle:     job.Title,
		Company:      job.Company,
		JobDesc:      job.Description,
		Skills:       job.Skills,
		Instructions: instructions,
		IsLatex:      isLatex,
	}

	resp, err := e.aiClient.TailorResume(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("resume tailoring failed: %w", err)
	}

	return resp, nil
}

func (e *TailorEngine) ExtractJobSkills(ctx context.Context, job *models.Job) (*ai.SkillExtract, error) {
	skills, err := e.aiClient.ExtractSkills(ctx, job.Description)
	if err != nil {
		return nil, fmt.Errorf("skill extraction failed: %w", err)
	}
	return skills, nil
}
