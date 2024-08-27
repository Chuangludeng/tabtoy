package model

type LocDefine struct {
	Key     string `tb_name:"Key"`
	Chinese string `tb_name:"中文"`
	Cht     string `tb_name:"繁中"`
	State   string
}
