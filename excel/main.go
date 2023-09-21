package main

import (
	"embed"
	"fmt"

	"github.com/xuri/excelize/v2"
)

//go:embed _data/template.xlsx
var tmpl embed.FS

func main() {
	tmplFile, err := tmpl.Open("_data/template.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}

	f, err := excelize.OpenReader(tmplFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	activeSheetName := f.GetSheetName(f.GetActiveSheetIndex())
	// Get all the rows in the Sheet1.
	rows, err := f.GetRows(activeSheetName)
	if err != nil {
		fmt.Println(err)
		return
	}
    head := rows[0]
    
	for _, colCell := range head {
		fmt.Print(colCell, "\t")
	}
	fmt.Println()

    for idx, row := range [][]interface{}{
        {nil, "Apple", "Orange", "Pear"}, {"Small", 2, 3, 3},
        {"Normal", 5, 2, 4}, {"Large", 6, 7, 8},
    } {
        cell, err := excelize.CoordinatesToCellName(1, idx+1)
        if err != nil {
            fmt.Println(err)
            return
        }
        f.SetSheetRow(activeSheetName, cell, &row)
    }

     // Save spreadsheet by the given path.
    if err := f.SaveAs("Book1.xlsx"); err != nil {
        fmt.Println(err)
    }

}
