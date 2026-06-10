package resume

import (
	"context"
	"fmt"

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

func (e *TailorEngine) TailorForJob(ctx context.Context, resumeData *ResumeData, job *models.Job, instructions string) (*ai.TailorResponse, error) {
	resumeText := resumeData.RawText

	req := ai.TailorRequest{
		ResumeData:   resumeText,
		JobTitle:     job.Title,
		Company:      job.Company,
		JobDesc:      job.Description,
		Skills:       job.Skills,
		Instructions: instructions,
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
