package contracts

import "github.com/EdOoO21/metadata-parser/internal/domain/model"

// SourceScanResult содержит результат чтения одного источника до записи в каталог.
// Lifecycle поля run/run_source намеренно не включены: они заполняются orchestration-слоем.
type SourceScanResult struct {
	Source              model.Source
	EffectiveConfigJSON []byte
	Datasets            []ScannedDataset
}

// ScannedDataset описывает найденный датасет и его колонки.
// ID и внешние ключи должны оставаться пустыми до этапа сохранения.
type ScannedDataset struct {
	Dataset model.Dataset
	Columns []ScannedColumn
}

// ScannedColumn описывает колонку и связанные с ней результаты профилирования.
// ID и внешние ключи должны оставаться пустыми до этапа сохранения.
type ScannedColumn struct {
	Column    model.Column
	Stat      *model.ColumnStat
	TopValues []model.ColumnTopValue
}
