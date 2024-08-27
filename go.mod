module github.com/Chuangludeng/tabtoy

go 1.22

require (
	github.com/ahmetb/go-linq v3.0.0+incompatible
	github.com/davyxu/golexer v0.1.0
	github.com/davyxu/golog v0.1.0
	github.com/davyxu/protoplus v0.1.0
	github.com/davyxu/tabtoy v0.0.0
	github.com/pkg/errors v0.8.1
	github.com/pkg/profile v1.4.0
	github.com/tealeg/xlsx v1.0.4-0.20190601071628-e2d23f3c43dc
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948
	golang.org/x/text v0.17.0
	google.golang.org/protobuf v1.23.0
)

replace github.com/davyxu/tabtoy => ../tabtoy

replace github.com/tealeg/xlsx => github.com/FduxGame/xlsx v1.0.6-0.20240826072004-b08b4b9e0aab
