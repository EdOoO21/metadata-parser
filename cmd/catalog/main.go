package main

import (
	"fmt"
	"os"

	diffapp "github.com/EdOoO21/metadata-parser/internal/application/diff"
	reportapp "github.com/EdOoO21/metadata-parser/internal/application/report"
	runapp "github.com/EdOoO21/metadata-parser/internal/application/run"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/cli"
	connectorfactory "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/factory"
	filescsv "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/filescsv"
	parquetsrc "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/parquet"
	postgressrc "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/postgressrc"
	restopenapi "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/restopenapi"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/db/postgres"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/logging"
	"github.com/EdOoO21/metadata-parser/internal/settings"
	"github.com/spf13/cobra"
)

func main() {
	if err := settings.LoadEnvFileIfPresent(".env"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	logger := logging.NewLogger()
	configLoader := settings.NewLoader()
	catalogOpener := postgres.NewRepositoryOpener()

	csvScanner := filescsv.NewCSVParser()
	parquetScanner := parquetsrc.NewScanner()
	postgresScanner := postgressrc.NewScanner()
	restScanner := restopenapi.NewScanner()

	scannerFactory := connectorfactory.New(csvScanner, parquetScanner, postgresScanner, restScanner)
	sourceHandler := runapp.NewScannerSourceHandler(logger)
	sourceProcessor := runapp.NewSourceProcessor(logger, scannerFactory, sourceHandler)

	runCatalogUseCase := runapp.NewRunCatalogUseCase(
		logger,
		sourceProcessor,
	)
	reportCatalogUseCase := reportapp.NewReportCatalogUseCase()
	diffCatalogUseCase := diffapp.NewDiffCatalogUseCase()

	return cli.NewRootCmd(cli.Dependencies{
		ConfigLoader:         configLoader,
		CatalogOpener:        catalogOpener,
		RunCatalogUseCase:    runCatalogUseCase,
		ReportCatalogUseCase: reportCatalogUseCase,
		DiffCatalogUseCase:   diffCatalogUseCase,
	})
}
