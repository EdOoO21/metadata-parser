package main

import (
	"fmt"
	"os"

	diffapp "github.com/EdOoO21/metadata-parser/internal/application/diff"
	reportapp "github.com/EdOoO21/metadata-parser/internal/application/report"
	runapp "github.com/EdOoO21/metadata-parser/internal/application/run"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/cli"
	filescsv "github.com/EdOoO21/metadata-parser/internal/infrastructure/connectors/filescsv"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/db/postgres"
	"github.com/EdOoO21/metadata-parser/internal/infrastructure/logging"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

func main() {
	logger := logging.NewLogger()
	configLoader := settings.NewLoader()
	catalogOpener := postgres.NewRepositoryOpener()

	csvParser := filescsv.NewCSVParser()
	filesSourceHandler := runapp.NewFilesSourceHandler(csvParser, logger)
	sourceProcessor := runapp.NewSourceProcessor(logger, filesSourceHandler)

	runCatalogUseCase := runapp.NewRunCatalogUseCase(
		logger,
		sourceProcessor,
	)
	reportCatalogUseCase := reportapp.NewReportCatalogUseCase()
	diffCatalogUseCase := diffapp.NewDiffCatalogUseCase()

	if err := cli.NewRootCmd(cli.Dependencies{
		ConfigLoader:         configLoader,
		CatalogOpener:        catalogOpener,
		RunCatalogUseCase:    runCatalogUseCase,
		ReportCatalogUseCase: reportCatalogUseCase,
		DiffCatalogUseCase:   diffCatalogUseCase,
	}).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
