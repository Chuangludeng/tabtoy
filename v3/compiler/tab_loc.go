package compiler

import (
	"github.com/davyxu/tabtoy/v3/model"
)

func parseLocRow(tab *model.DataTable, symbols *model.TypeTable, locMap map[string]*model.LocDefine) {

	for row := 1; row < len(tab.Rows); row++ {

		var pragma model.LocDefine
		if !ParseRow(&pragma, tab, row, symbols) {
			continue
		}

		locMap[pragma.Key] = &pragma
	}

	return
}

func LoadLocTable(globals *model.Globals, fileName string) error {

	if fileName == "" {
		return nil
	}

	// 加载原始数据
	tabs, err := LoadDataTable(globals.LocalizationGetter, fileName, "LocDefine", "LocDefine", globals.Types, true)

	if err != nil {
		return err
	}

	globals.LocMap = make(map[string]*model.LocDefine)

	for _, tab := range tabs {
		parseLocRow(tab, globals.Types, globals.LocMap)
	}

	return nil
}
