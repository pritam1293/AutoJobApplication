package resume

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jung-kurt/gofpdf/v2"
	"github.com/ledongthuc/pdf"
)

type ResumeData struct {
	RawText    string            `json:"raw_text"`
	FileName   string            `json:"file_name"`
	FileSize   int64             `json:"file_size"`
	PageCount  int               `json:"page_count"`
	Structured *StructuredResume `json:"structured,omitempty"`
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
		if strings.TrimSpace(pageText) != "" {
			text += pageText + "\n"
		}
	}

	text = formatResumeText(text)

	return &ResumeData{
		RawText:   text,
		FileName:  stat.Name(),
		FileSize:  stat.Size(),
		PageCount: reader.NumPage(),
	}, nil
}

func formatResumeText(raw string) string {
	sectionRe := regexp.MustCompile(`(?i)(Education|Work\s*Experience|Experience|Projects?|Technical\s*Skills?|Achievements?|Certifications?|Relevant\s*Coursework|Coursework|Summary|Professional\s*Summary|Profile|Publications?|Leadership|Languages?|Interests?|References?|Additional)`)

	result := sectionRe.ReplaceAllString(raw, "\n\n$1")

	result = strings.ReplaceAll(result, "•", "\n• ")
	result = strings.ReplaceAll(result, "●", "\n● ")
	result = strings.ReplaceAll(result, "○", "\n○ ")
	result = strings.ReplaceAll(result, "●", "\n● ")

	result = regexp.MustCompile(`\n{3,}`).ReplaceAllString(result, "\n\n")

	lines := strings.Split(result, "\n")
	var formatted []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			formatted = append(formatted, trimmed)
		}
	}

	return strings.Join(formatted, "\n")
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

func CompileLatex(latexSource string, outputDir string, filename string) (string, error) {
	absDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("cannot resolve output dir: %w", err)
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create output dir: %w", err)
	}

	name := strings.TrimSuffix(filename, ".tex")
	if name == "" {
		name = "resume"
	}

	texPath := filepath.Join(absDir, name+".tex")
	pdfPath := filepath.Join(absDir, name+".pdf")

	if err := os.WriteFile(texPath, []byte(latexSource), 0644); err != nil {
		return "", fmt.Errorf("cannot write latex file: %w", err)
	}

	pdflatex, err := exec.LookPath("pdflatex")
	if err != nil {
		return "", fmt.Errorf("pdflatex not found; install texlive: %w", err)
	}

	cmd := exec.Command(pdflatex, "-interaction=nonstopmode", "-output-directory", absDir, texPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pdflatex failed: %w\noutput: %s", err, string(out))
	}

	_ = os.Remove(filepath.Join(absDir, name+".aux"))
	_ = os.Remove(filepath.Join(absDir, name+".log"))
	_ = os.Remove(filepath.Join(absDir, name+".out"))

	if _, err := os.Stat(pdfPath); os.IsNotExist(err) {
		return "", fmt.Errorf("pdflatex ran but no PDF was generated\noutput: %s", string(out))
	}

	return pdfPath, nil
}

func GeneratePDF(resumeContent string, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	lines := strings.Split(resumeContent, "\n")
	inBullet := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			inBullet = false
			continue
		}

		isBullet := strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "•")
		isHeader := false

		uppercase := strings.ToUpper(trimmed)
		clean := strings.TrimLeft(trimmed, "-*•# ")
		cleanTrimmed := strings.TrimSpace(clean)

		if strings.Count(trimmed, "#") > 0 || trimmed == strings.ToUpper(trimmed) && len(trimmed) > 3 && len(trimmed) < 60 {
			upperRatio := 0.0
			if len(trimmed) > 0 {
				upperCount := 0
				for _, ch := range trimmed {
					if ch >= 'A' && ch <= 'Z' {
						upperCount++
					}
				}
				upperRatio = float64(upperCount) / float64(len(trimmed))
			}

			if strings.HasPrefix(trimmed, "#") || (upperRatio > 0.7 && len(trimmed) > 5) {
				isHeader = true
			}
		}

		if strings.HasPrefix(uppercase, "SKILLS") || strings.HasPrefix(uppercase, "EXPERIENCE") ||
			strings.HasPrefix(uppercase, "EDUCATION") || strings.HasPrefix(uppercase, "SUMMARY") ||
			strings.HasPrefix(uppercase, "PROJECT") || strings.HasPrefix(uppercase, "CERTIFICATION") ||
			strings.HasPrefix(uppercase, "WORK") || strings.HasPrefix(uppercase, "PROFILE") {
			isHeader = true
		}

		if isHeader {
			pdf.SetFont("Helvetica", "B", 13)
			pdf.SetTextColor(30, 60, 180)
			pdf.CellFormat(0, 8, tr(cleanTrimmed), "", 1, "L", false, 0, "")
			pdf.SetDrawColor(30, 60, 180)
			pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
			pdf.Ln(3)
			inBullet = false
		} else if isBullet || inBullet {
			pdf.SetFont("Helvetica", "", 10)
			pdf.SetTextColor(40, 40, 40)
			text := cleanTrimmed
			if pdf.GetStringWidth(tr(text)) > 160 {
				pdf.MultiCell(170, 5, tr("• "+text), "", "L", false)
			} else {
				pdf.CellFormat(0, 6, tr("• "+text), "", 1, "L", false, 0, "")
			}
			inBullet = true
		} else {
			pdf.SetFont("Helvetica", "", 11)
			pdf.SetTextColor(50, 50, 50)
			if pdf.GetStringWidth(tr(trimmed)) > 160 {
				pdf.MultiCell(170, 5.5, tr(trimmed), "", "L", false)
			} else {
				pdf.CellFormat(0, 6, tr(trimmed), "", 1, "L", false, 0, "")
			}
		}
	}

	return pdf.OutputFileAndClose(outputPath)
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
