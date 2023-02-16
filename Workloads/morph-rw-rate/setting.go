package main

import (
	"database/sql"
)

// 一些数据库连接的配置
var (
	database = "replay" // 数据库名
	// 连接相关
	driver      = "mysql" // 数据库引擎
	user        = "morph"
	passwd      = "morphdag"
	protocol    = "tcp" //连接协议
	port        = "3306"
	useDatabase = "USE " + database

	dataSource string

	tableOfBlocks = []string{"block1", "block2", "block3", "block4", "block5", "block6", "block7", "block8", "block9", "block10",
		"block11", "block12", "block13", "block14", "block15", "block16", "block17", "block18", "block19", "block20",
		"block21", "block22", "block23", "block24", "block25", "block26", "block27", "block28", "block29", "block30",
		"block31", "block32", "block33", "block34", "block35", "block36", "block37", "block38", "block39", "block40",
		"block41", "block42", "block43", "block44", "block45", "block46", "block47", "block48", "block49", "block50",
		"block51", "block52", "block53", "block54", "block55", "block56", "block57", "block58", "block59", "block60"} // 表名
	sqlServers []*sql.DB
)

func SetHost(host string) {
	dataSource = user + ":" + passwd + "@" + protocol + "(" + host + ":" + port + ")/" // 用户名:密码@tcp(ip:端口)/
}

type RWRate struct {
	StateRead    int
	StateWrite   int
	StorageRead  int
	StorageWrite int
}

func newRWRate() *RWRate {
	return &RWRate{
		StateRead: -1,
		StateWrite: -1,
		StorageRead: -1,
		StorageWrite: -1,
	}
}