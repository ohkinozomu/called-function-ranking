package main

import (
	"errors"
	"flag"
	"go/parser"
	"go/token"

	"github.com/apache/arrow/go/arrow"
	"github.com/bmatcuk/doublestar/v3"
	"github.com/emer/etable/agg"
	"github.com/emer/etable/etable"
	"github.com/emer/etable/etensor"
	"github.com/emer/etable/split"
	calledge "github.com/ohkinozomu/go-call-edge"
	"github.com/pterm/pterm"
)

func createTable(edges []calledge.CallEdge) (*etable.Table, error) {
	schema := []etable.Column{
		{
			Name: "caller",
			Type: etensor.Type(arrow.STRING),
		},
		{
			Name: "callee",
			Type: etensor.Type(arrow.STRING),
		},
	}
	table := etable.NewTable("calledge")
	table.SetFromSchema(schema, len(edges))
	for i, edge := range edges {
		if !table.SetCellString("caller", i, edge.Caller) {
			return nil, errors.New("set fails")
		}
		if !table.SetCellString("callee", i, edge.Callee) {
			return nil, errors.New("set fails")
		}
	}
	return table, nil
}

func printRanking(edges []calledge.CallEdge) error {
	table, err := createTable(edges)
	if err != nil {
		return err
	}
	ix1 := etable.NewIdxView(table)
	splits := split.GroupBy(ix1, []string{"callee"})
	_ = split.Agg(splits, "callee", agg.AggCount)
	ix2 := etable.NewIdxView(splits.AggsToTable(etable.AddAggName))
	ix2.SortColName("callee:Count", etable.Descending)
	t := ix2.NewTable()

	var tableData pterm.TableData
	tableData = append(tableData, []string{"callee", "callee:Count"})
	for i := 0; i < t.Rows; i++ {
		tableData = append(tableData, []string{t.CellStringIdx(0, i), t.CellStringIdx(1, i)})
	}
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	return nil
}

func findGoFiles(path string) ([]string, error) {
	goFles, err := doublestar.Glob(path + "/**/*.go")
	if err != nil {
		return nil, err
	}
	return goFles, nil
}

func main() {
	var (
		dir = flag.String("dir", "", "directory")
	)
	flag.Parse()

	if *dir == "" {
		panic("Input -dir")
	}
	files, err := findGoFiles(*dir)
	if err != nil {
		panic(err)
	}

	edges := []calledge.CallEdge{}
	for _, file := range files {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, file, nil, parser.AllErrors)
		if err != nil {
			panic(err)
		}
		edges = append(edges, calledge.GetCallEdges(f)...)
	}
	err = printRanking(edges)
	if err != nil {
		panic(err)
	}
}
