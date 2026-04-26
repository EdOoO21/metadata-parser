package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	diffapp "github.com/EdOoO21/metadata-parser/internal/application/diff"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	reportapp "github.com/EdOoO21/metadata-parser/internal/application/report"
	runapp "github.com/EdOoO21/metadata-parser/internal/application/run"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type configLoaderStub struct {
	cfg *settings.AppConfig
	err error
}

func (l *configLoaderStub) Load(path string) (*settings.AppConfig, error) {
	if l.err != nil {
		return nil, l.err
	}
	return l.cfg, nil
}

type catalogConnStub struct {
	repo appports.CatalogRepository
}

func (c *catalogConnStub) Repository() appports.CatalogRepository { return c.repo }
func (c *catalogConnStub) Close()                                 {}

type catalogOpenerStub struct {
	conn appports.CatalogConnection
	err  error
}

func (o *catalogOpenerStub) Open(ctx context.Context, dsnEnv string) (appports.CatalogConnection, error) {
	if o.err != nil {
		return nil, o.err
	}
	return o.conn, nil
}

type runUseCaseStub struct {
	runID int64
	err   error
}

func (u *runUseCaseStub) Execute(ctx context.Context, input runapp.ExecuteInput) (int64, error) {
	if u.err != nil {
		return 0, u.err
	}
	return u.runID, nil
}

type reportUseCaseStub struct {
	result *reportapp.Result
	err    error
}

func (u *reportUseCaseStub) Execute(ctx context.Context, input reportapp.ExecuteInput) (*reportapp.Result, error) {
	if u.err != nil {
		return nil, u.err
	}
	return u.result, nil
}

type diffUseCaseStub struct {
	message string
	err     error
}

func (u *diffUseCaseStub) Execute(ctx context.Context, input diffapp.ExecuteInput) (string, error) {
	if u.err != nil {
		return "", u.err
	}
	return u.message, nil
}

type noopRepoStub struct{}

func (s *noopRepoStub) WithTx(ctx context.Context, fn func(repo appports.CatalogRepository) error) error {
	return fn(s)
}
func (s *noopRepoStub) EnsureSource(ctx context.Context, source model.Source) (*model.Source, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) CreateRun(ctx context.Context, run model.Run) (*model.Run, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) GetRun(ctx context.Context, runID int64) (*model.Run, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) UpdateRunStatus(ctx context.Context, runID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	return errors.New("unexpected test call")
}
func (s *noopRepoStub) CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) UpdateRunSourceStatus(ctx context.Context, runSourceID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	return errors.New("unexpected test call")
}
func (s *noopRepoStub) CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) CreateColumn(ctx context.Context, column model.Column) (*model.Column, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error) {
	return nil, errors.New("unexpected test call")
}
func (s *noopRepoStub) CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error {
	return errors.New("unexpected test call")
}
func (s *noopRepoStub) ListReportRows(ctx context.Context, runID int64) ([]appports.ReportRow, error) {
	return nil, errors.New("unexpected test call")
}

func testConfig() *settings.AppConfig {
	return &settings.AppConfig{
		Version: 1,
		Catalog: settings.CatalogConfig{DSNEnv: "CATALOG_DSN"},
	}
}

func TestNewRunCmd_Success(t *testing.T) {
	t.Parallel()

	cmd := NewRunCmd(
		&configLoaderStub{cfg: testConfig()},
		&catalogOpenerStub{conn: &catalogConnStub{repo: &noopRepoStub{}}},
		&runUseCaseStub{runID: 42},
	)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", "demo.yaml"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "run completed successfully: run_id=42") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestNewReportCmd_WritesMarkdownHTMLAndCSV(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	markdownPath := filepath.Join(dir, "report.md")
	htmlPath := filepath.Join(dir, "report.html")
	csvPath := filepath.Join(dir, "report.csv")

	cmd := NewReportCmd(
		&configLoaderStub{cfg: testConfig()},
		&catalogOpenerStub{conn: &catalogConnStub{repo: &noopRepoStub{}}},
		&reportUseCaseStub{
			result: &reportapp.Result{
				Markdown: "# report\n",
				HTML:     "<html></html>\n",
				CSV:      []byte("a,b\n"),
			},
		},
	)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{
		"--config", "demo.yaml",
		"--run-id", "7",
		"--output", markdownPath,
		"--html-output", htmlPath,
		"--csv-output", csvPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for path, expected := range map[string]string{
		markdownPath: "# report\n",
		htmlPath:     "<html></html>\n",
		csvPath:      "a,b\n",
	} {
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file %s: %v", path, err)
		}
		if string(payload) != expected {
			t.Fatalf("unexpected file %s content: %q", path, string(payload))
		}
	}
}

func TestNewDiffCmd_PrintsMessage(t *testing.T) {
	t.Parallel()

	cmd := NewDiffCmd(
		&configLoaderStub{cfg: testConfig()},
		&catalogOpenerStub{conn: &catalogConnStub{repo: &noopRepoStub{}}},
		&diffUseCaseStub{message: "diff output"},
	)

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", "demo.yaml", "--from-run-id", "1", "--to-run-id", "2"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "diff output") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestNewRootCmd_ContainsCommandsAndHelp(t *testing.T) {
	t.Parallel()

	root := NewRootCmd(Dependencies{})
	commands := root.Commands()
	if len(commands) < 3 {
		t.Fatalf("expected run/report/diff commands, got %d", len(commands))
	}

	found := map[string]bool{}
	for _, cmd := range commands {
		found[cmd.Name()] = true
	}

	for _, name := range []string{"run", "report", "diff"} {
		if !found[name] {
			t.Fatalf("expected command %q to exist", name)
		}
	}

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected help error: %v", err)
	}
	if !strings.Contains(out.String(), "Доступные команды") {
		t.Fatalf("unexpected help output: %s", out.String())
	}
}

func TestCommandDependencyErrorsAndWriteFileError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  func() error
	}{
		{
			name: "run missing config loader",
			cmd: func() error {
				command := NewRunCmd(nil, &catalogOpenerStub{conn: &catalogConnStub{repo: &noopRepoStub{}}}, &runUseCaseStub{runID: 1})
				command.SetArgs([]string{"--config", "demo.yaml"})
				return command.Execute()
			},
		},
		{
			name: "report missing opener",
			cmd: func() error {
				command := NewReportCmd(&configLoaderStub{cfg: testConfig()}, nil, &reportUseCaseStub{result: &reportapp.Result{}})
				command.SetArgs([]string{"--config", "demo.yaml", "--run-id", "1"})
				return command.Execute()
			},
		},
		{
			name: "diff missing usecase",
			cmd: func() error {
				command := NewDiffCmd(&configLoaderStub{cfg: testConfig()}, &catalogOpenerStub{conn: &catalogConnStub{repo: &noopRepoStub{}}}, nil)
				command.SetArgs([]string{"--config", "demo.yaml", "--from-run-id", "1", "--to-run-id", "2"})
				return command.Execute()
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := tt.cmd(); err == nil {
				t.Fatal("expected error")
			}
		})
	}

	dir := t.TempDir()
	cmd := NewReportCmd(
		&configLoaderStub{cfg: testConfig()},
		&catalogOpenerStub{conn: &catalogConnStub{repo: &noopRepoStub{}}},
		&reportUseCaseStub{result: &reportapp.Result{Markdown: "x", HTML: "y", CSV: []byte("z")}},
	)
	cmd.SetArgs([]string{
		"--config", "demo.yaml",
		"--run-id", "1",
		"--output", dir,
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected write file error")
	}
}
