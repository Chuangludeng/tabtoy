package pbdata

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/Chuangludeng/tabtoy/v3/helper"
	"golang.org/x/exp/slices"
	"io/ioutil"
	"strconv"

	"github.com/davyxu/tabtoy/v3/model"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func exportTable(globals *model.Globals, pbFile protoreflect.FileDescriptor, tab *model.DataTable, combineRoot *dynamicpb.Message,
	localizationMap map[string]string) bool {
	md := pbFile.Messages().ByName(protoreflect.Name(tab.OriginalHeaderType))

	combineField := combineRoot.Descriptor().Fields().ByName(protoreflect.Name(tab.OriginalHeaderType))
	list := combineRoot.NewField(combineField).List()

	// 每个表的所有列
	headers := globals.Types.AllFieldByName(tab.OriginalHeaderType)

	hasIgnore := false

	// 遍历每一行
	for row := 1; row < len(tab.Rows); row++ {

		rowMsg := dynamicpb.NewMessage(md)

		var colIndex int
		for _, field := range headers {

			if globals.CanDoAction(model.ActionNoGenFieldPbBinary, field) {
				hasIgnore = true
				continue
			}

			fd := md.Fields().ByName(protoreflect.Name(field.FieldName))

			if fd.Kind() == protoreflect.MessageKind {
				var max int
				var list protoreflect.List
				if field.IsArray() {
					list = rowMsg.NewField(fd).List()
					max, _ = strconv.Atoi(field.Value)
				} else {
					max = 1
				}

				var structMsg *dynamicpb.Message
				for i := 0; i < max; i++ {
					structMD := pbFile.Messages().ByName(protoreflect.Name(field.FieldType))
					structMsg = dynamicpb.NewMessage(structMD)

					var nilNumber int
					structFields := globals.Types.AllFieldByName(field.FieldType)
					fieldsNum := len(structFields)
					for _, field := range structFields {
						// 在单元格找到值
						valueCell := tab.GetCell(row, colIndex)
						if valueCell == nil || valueCell.Value == "" {
							nilNumber++
							colIndex++
							continue
						}
						structFd := structMD.Fields().ByName(protoreflect.Name(field.FieldName))
						var pbValue protoreflect.Value
						if field.Localization && localizationMap != nil {
							hash := md5Hex(valueCell.Value)
							if v, ok := localizationMap[hash]; ok && v == valueCell.Value {
								hash = tab.OriginalHeaderType + hash
							}
							localizationMap[hash] = valueCell.Value
							pbValue = protoreflect.ValueOfString(hash)
						} else {
							pbValue = tableValue2PbValue(globals, valueCell.Value, field)
						}
						structMsg.Set(structFd, pbValue)
						colIndex++
					}

					if nilNumber != fieldsNum && field.IsArray() {
						list.Append(protoreflect.ValueOf(structMsg))
					}
				}

				if field.IsArray() {
					rowMsg.Set(fd, protoreflect.ValueOfList(list))
				} else {
					rowMsg.Set(fd, protoreflect.ValueOfMessage(structMsg))
				}
			} else {
				// 在单元格找到值
				valueCell := tab.GetCell(row, colIndex)
				if valueCell == nil {
					colIndex++
					continue
				}

				if field.IsArray() {
					list := rowMsg.NewField(fd).List()
					tableValue2PbValueList(globals, valueCell, field, list)
					rowMsg.Set(fd, protoreflect.ValueOfList(list))
				} else {
					pbValue := tableValue2PbValue(globals, valueCell.Value, field)
					rowMsg.Set(fd, pbValue)
				}
				colIndex++
			}
		}

		list.Append(protoreflect.ValueOf(rowMsg))
	}

	combineRoot.Set(combineField, protoreflect.ValueOfList(list))
	return hasIgnore
}

func Generate(globals *model.Globals) (data []byte, err error) {

	pbFile, err := buildDynamicType(globals)
	if err != nil {
		return nil, err
	}

	combineD := pbFile.Messages().ByName(protoreflect.Name(globals.CombineStructName))

	combineRoot := dynamicpb.NewMessage(combineD)

	localizationMap := make(map[string]string)

	// 所有的表
	for _, tab := range globals.Datas.AllTables() {
		exportTable(globals, pbFile, tab, combineRoot, localizationMap)
	}

	return proto.MarshalOptions{Deterministic: true}.Marshal(combineRoot)
}

func Output(globals *model.Globals, param string) (err error) {

	pbFile, err := buildDynamicType(globals)
	if err != nil {
		return err
	}

	localizationMap := make(map[string]string)

	for _, tab := range globals.Datas.AllTables() {

		if len(globals.OnlyDispose) != 0 && tab.HeaderType != globals.OnlyDispose {
			continue
		}

		combineD := pbFile.Messages().ByName(protoreflect.Name(globals.CombineStructName))

		combineRoot := dynamicpb.NewMessage(combineD)

		//这里做一个Hack,先导包含tagAction的,然后导一次不包含tagAction,分别存放
		hasIgnore := exportTable(globals, pbFile, tab, combineRoot, localizationMap)

		data, err := proto.MarshalOptions{Deterministic: true}.Marshal(combineRoot)
		if err != nil {
			return err
		}

		if hasIgnore {
			err = ioutil.WriteFile(fmt.Sprintf("%s/%s_Ignore.pbb", param, tab.HeaderType), data, 0666)
		} else {
			err = ioutil.WriteFile(fmt.Sprintf("%s/%s.pbb", param, tab.HeaderType), data, 0666)
		}

		if err != nil {
			return err
		}

		if hasIgnore {

			combineD := pbFile.Messages().ByName(protoreflect.Name(globals.CombineStructName))

			combineRoot := dynamicpb.NewMessage(combineD)

			globals.IgnoreTagActions = true
			exportTable(globals, pbFile, tab, combineRoot, nil)
			globals.IgnoreTagActions = false

			data, err := proto.MarshalOptions{Deterministic: true}.Marshal(combineRoot)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(fmt.Sprintf("%s/%s.pbb", param, tab.HeaderType), data, 0666)

			if err != nil {
				return err
			}
		}
	}

	if len(localizationMap) == 0 {
		return nil
	}

	keys := maps.Keys(globals.LocMap)
	slices.Sort(keys)
	for _, key := range keys {
		if v, ok := localizationMap[key]; ok {
			globals.LocMap[key].Chinese = v
			globals.LocMap[key].State = ""
		} else {
			globals.LocMap[key].State = "Delete"
		}
	}

	keys = maps.Keys(localizationMap)
	slices.Sort(keys)
	for _, key := range keys {
		if _, ok := globals.LocMap[key]; !ok {
			var newLoc model.LocDefine
			newLoc.Key = key
			newLoc.Chinese = localizationMap[key]
			newLoc.State = "New"
			globals.LocMap[key] = &newLoc
		}
	}

	keys = maps.Keys(globals.LocMap)
	slices.Sort(keys)

	memfile := helper.NewMemFile()
	sheet := memfile.CreateXLSXFile("Localization/TableLocalization.xlsx")
	sheet.WriteRow("Key", "中文", "繁中", "#状态")
	for _, key := range keys {
		loc := globals.LocMap[key]
		sheet.WriteRow(loc.Key, loc.Chinese, loc.Cht, loc.State)
	}

	memfile.VisitAllTable(func(data *helper.MemFileData) bool {
		ret := data.File.Save(data.FileName)
		if ret != nil {
			return false
		}

		return true
	})

	return nil
}
