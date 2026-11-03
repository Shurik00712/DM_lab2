package export

import (
	"fmt"
	"io"
	"errors"
	"github.com/xuri/excelize/v2"

	"file_db/internal/schema"
)

func XLSX(w io.Writer, sheet string, rows []map[string]any, s *schema.Schema) error {
	if s == nil {
		return errors.New("nil schema")
	}
	f := excelize.NewFile()
	defer f.Close()

	const defaultSheet = "Sheet1"
	idx, err := f.GetSheetIndex(defaultSheet)
	if err != nil {
		return err
	}
	f.SetSheetName(defaultSheet, sheet)
	f.SetActiveSheet(idx)

	for col, field := range s.Fields {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		if err := f.SetCellValue(sheet, cell, field.Name); err != nil {
			return err
		}
	}

	for r, row := range rows {
		for col, field := range s.Fields {
			cell, _ := excelize.CoordinatesToCellName(col+1, r+2)
			v := row[field.Name]
			if v == nil {
				continue
			}
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				return fmt.Errorf("row %d col %s: %w", r+2, field.Name, err)
			}
		}
	}

	for col, field := range s.Fields {
		colName, _ := excelize.ColumnNumberToName(col + 1)
		width := float64(len(field.Name)) + 2
		if width < 10 {
			width = 10
		}
		f.SetColWidth(sheet, colName, colName, width)
	}

	if err := f.Write(w); err != nil {
		return fmt.Errorf("write xlsx: %w", err)
	}
	return nil
}