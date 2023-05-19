package pbdata

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/davyxu/tabtoy/v3/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func exportTable(globals *model.Globals, pbFile protoreflect.FileDescriptor, tab *model.DataTable, combineRoot *dynamicpb.Message) bool {
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
						pbValue := tableValue2PbValue(globals, valueCell.Value, field)
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

	// 所有的表
	for _, tab := range globals.Datas.AllTables() {
		exportTable(globals, pbFile, tab, combineRoot)
	}

	return proto.MarshalOptions{Deterministic: true}.Marshal(combineRoot)
}

func Output(globals *model.Globals, param string) (err error) {

	pbFile, err := buildDynamicType(globals)
	if err != nil {
		return err
	}

	for _, tab := range globals.Datas.AllTables() {

		if len(globals.OnlyDispose) != 0 && tab.HeaderType != globals.OnlyDispose {
			continue
		}

		combineD := pbFile.Messages().ByName(protoreflect.Name(globals.CombineStructName))

		combineRoot := dynamicpb.NewMessage(combineD)

		//这里做一个Hack,先导包含tagAction的,然后导一次不包含tagAction,分别存放
		hasIgnore := exportTable(globals, pbFile, tab, combineRoot)

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
			exportTable(globals, pbFile, tab, combineRoot)
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

	return nil
}
