package postgressrc

import (
	"errors"
	"testing"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

func TestBuildDiscoveryDatasets(t *testing.T) {
	t.Parallel()

	comment := "people table"
	columnComment := "identifier"
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)

	datasets, err := buildDiscoveryDatasets([]discoveryRow{
		{
			schemaName:     "public",
			datasetName:    "people",
			datasetKind:    "VIEW",
			datasetComment: &comment,
			columnName:     "id",
			columnType:     "integer",
			columnNullable: false,
			columnComment:  &columnComment,
			ordinal:        1,
		},
		{
			schemaName:     "public",
			datasetName:    "people",
			datasetKind:    "VIEW",
			columnName:     "email",
			columnType:     "text",
			columnNullable: true,
			ordinal:        2,
		},
	}, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(datasets) != 1 {
		t.Fatalf("expected 1 dataset, got %d", len(datasets))
	}
	dataset := datasets[0]
	if dataset.Dataset.Kind != types.DatasetKindView {
		t.Fatalf("unexpected dataset kind: %s", dataset.Dataset.Kind)
	}
	if dataset.Dataset.Name != "people" || dataset.Dataset.Location != "public.people" {
		t.Fatalf("unexpected dataset: %+v", dataset.Dataset)
	}
	if !dataset.Dataset.DiscoveredAt.Equal(now) {
		t.Fatalf("unexpected discovered time: %v", dataset.Dataset.DiscoveredAt)
	}
	if len(dataset.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(dataset.Columns))
	}
	if dataset.Columns[0].Column.NormalizedType != types.CanonicalTypeNumber {
		t.Fatalf("unexpected first column type: %+v", dataset.Columns[0].Column)
	}
}

func TestApplyProfileStatus(t *testing.T) {
	t.Parallel()

	dataset := contracts.ScannedDataset{}
	applyProfileStatus(&dataset, nil)
	if dataset.Dataset.ProfileStatus != types.ProfileStatusProfiled || dataset.Dataset.ProfileError != nil {
		t.Fatalf("unexpected success profile state: %+v", dataset.Dataset)
	}

	applyProfileStatus(&dataset, errors.New("boom"))
	if dataset.Dataset.ProfileStatus != types.ProfileStatusFailed {
		t.Fatalf("unexpected failed profile status: %+v", dataset.Dataset)
	}
	if dataset.Dataset.ProfileError == nil || *dataset.Dataset.ProfileError != "boom" {
		t.Fatalf("unexpected failed profile error: %+v", dataset.Dataset.ProfileError)
	}
}
