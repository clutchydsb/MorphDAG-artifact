package main

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql" // 导入包但不使用, init()
	"math/big"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var rwChan chan *RWRate

func GetRWRateByBlockNumber(blockNumber *big.Int) *RWRate {
	var wg sync.WaitGroup
	numCpu := runtime.NumCPU()
	rwChan = make(chan *RWRate, numCpu)
	wg.Add(numCpu)
	var rwRate *RWRate

	tables := tableOfBlocks[0:numCpu]
	for index, table := range tables {
		go getRWRateByBlockNumber(table, blockNumber.String(), index, &wg)
	}
	var k = 0
	for {
		if k == numCpu {
			close(rwChan)
		}
		k += 1
		val, ok := <-rwChan
		if !ok {
			break
		}
		if val.StateWrite == -1 {
			continue
		}
		rwRate = val
	}

	wg.Wait()
	rwChan = nil
	return rwRate
}

// getRWRateByBlockNumber 获取某一区块号的区块的读写比例
func getRWRateByBlockNumber(table string, blockNumber string, index int, wg *sync.WaitGroup) {
	defer wg.Done()
	sqlServer := sqlServers[index]
	rows, err := sqlServer.Query("SELECT txInfos FROM " + table + " WHERE number=\"" + blockNumber + "\";")
	defer rows.Close() // 非常重要：关闭rows释放持有的数据库链接
	if err != nil {
		fmt.Println("Error: Query failed", err)
		return
	}

	rwRate := newRWRate()
	for rows.Next() {
		var info string
		err = rows.Scan(&info)
		if err != nil {
			fmt.Println("Error: Scan failed", err)
			return
		}
		if info == ""{
			break
		}
		infos := strings.Split(info, "|")
		//fmt.Println(infos)
		var stateStr, storageStr string
		if len(infos) == 8{
			stateStr, storageStr = infos[6], infos[7]
		}else {
			stateStr = infos[6]
		}
		rwRate.StateRead, rwRate.StateWrite = dealRW(stateStr)
		rwRate.StorageRead, rwRate.StorageWrite = dealRW(storageStr)
	}
	rwChan <- rwRate
}

func dealRW(rwStr string) (r, w int) {
	if rwStr == ""{
		return 0,0
	}
	rwStrs := strings.Split(rwStr, ";")
	var rStr, wStr string
	if len(rwStrs) == 2{
		rStr, wStr = rwStrs[0][2:], rwStrs[1][2:]
	}else {
		rStr = rwStrs[0][2:]
	}
	r = dealRWTimes(rStr)
	w = dealRWTimes(wStr)
	return r, w
}

func dealRWTimes(str string) int {
	if str == ""{
		return 0
	}
	var num int = 0
	strs := strings.Split(str, " ")
	for _, s := range strs{
		ss := strings.Split(s[2:], ",")[1]
		n, err := strconv.Atoi(ss)
		if err != nil {
			fmt.Println(err)
		}
		num += n
	}
	return num
}


