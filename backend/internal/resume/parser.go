package resume

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ledongthuc/pdf"
)

type ResumeData struct {
	RawText   string            `json:"raw_text"`
	FileName  string            `json:"file_name"`
	FileSize  int64             `json:"file_size"`
	PageCount int               `json:"page_count"`
	Structured  *StructuredResume `json:"structured,omitempty"`
}

type StructuredResume struct {
	Name         string       `json:"name"`
	Email        string       `json:"email"`
	Phone        string       `json:"phone"`
	LinkedIn     string       `json:"linkedin"`
	Summary      string       `json:"summary"`
	Skills       []string     `json:"skills"`
	Experience   []Experience `json:"experience"`
	Education    []Education  `json:"education"`
	Certificates []string     `json:"certificates"`
}

type Experience struct {
	Company     string `json:"company"`
	Title       string `json:"title"`
	Duration    string `json:"duration"`
	Description string `json:"description"`
}

type Education struct {
	Institution string `json:"institution"`
	Degree      string `json:"degree"`
	Field       string `json:"field"`
	Year        string `json:"year"`
}

func ParsePDF(filePath string) (*ResumeData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open resume file: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot stat resume file: %w", err)
	}

	reader, err := pdf.NewReader(f, stat.Size())
	if err != nil {
		return nil, fmt.Errorf("cannot read PDF: %w", err)
	}

	var text string
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		pageText, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text += pageText + "\n"
	}

	return &ResumeData{
		RawText:   text,
		FileName:  stat.Name(),
		FileSize:  stat.Size(),
		PageCount: reader.NumPage(),
	}, nil
}

func ParsePDFFromReader(r io.Reader, filename string) (*ResumeData, error) {
	tmpFile, err := os.CreateTemp("", "resume-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("cannot create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	written, err := io.Copy(tmpFile, r)
	if err != nil {
		return nil, fmt.Errorf("cannot write temp file: %w", err)
	}
	tmpFile.Close()

	data, err := ParsePDF(tmpFile.Name())
	if err != nil {
		return nil, err
	}
	data.FileName = filename
	data.FileSize = written

	return data, nil
}

func GeneratePDF(resumeContent string, outputPath string) error {
	return nil
}

type Manager struct {
	resumeDir string
}

func NewManager(resumeDir string) *Manager {
	return &Manager{resumeDir: resumeDir}
}

func (m *Manager) SaveResume(data *ResumeData, userID uint) (string, error) {
	dir := filepath.Join(m.resumeDir, fmt.Sprintf("user_%d", userID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create resume dir: %w", err)
	}

	dstPath := filepath.Join(dir, data.FileName)
	return dstPath, nil
}

func (m *Manager) GetResumePath(userID uint) string {
	return filepath.Join(m.resumeDir, fmt.Sprintf("user_%d", userID))
}
