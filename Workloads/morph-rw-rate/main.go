package main

import (
	"fmt"
	"math/big"
)

func main() {
	SetHost("202.114.7.176")
	err := OpenSqlServers()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer CloseSqlServers()
	//min, max := big.NewInt(7202102), big.NewInt(8852777)
	for i := big.NewInt(4500000); i.Cmp(big.NewInt(4550000)) == -1; i = i.Add(i, big.NewInt(1)) {
		rwRate := GetRWRateByBlockNumber(i)
		if rwRate == nil{
			fmt.Println(i, "StateRead", 0, "StateWrite", 0, "StorageRead", 0, "StorageWrite", 0)
			continue
		}
		StateNum := rwRate.StateRead + rwRate.StateWrite
		StorageNum := rwRate.StorageRead + rwRate.StorageWrite
		var StateRWRate, StorageRWRate string
		if StateNum > 0{
			r, w := rwRate.StateRead * 100 / StateNum, rwRate.StateWrite * 100 / StateNum
			if r + w < 100{
				w += 1
			}
			StateRWRate = fmt.Sprintf("%d:%d", r, w)

		}else {
			StateRWRate = "0:0"
		}
		if StorageNum != 0{
			r, w := rwRate.StorageRead * 100 / StorageNum, rwRate.StorageWrite * 100 / StorageNum
			if r + w < 100{
				w += 1
			}
			StorageRWRate = fmt.Sprintf("%d:%d", r, w)
		}else {
			StorageRWRate = "0:0"
		}
		fmt.Println(i, "StateRead", rwRate.StateRead, "StateWrite", rwRate.StateWrite,
			"StorageRead", rwRate.StorageRead, "StorageWrite", rwRate.StorageWrite,
			"StateRWRate", StateRWRate, "StorageRWRate", StorageRWRate)
	}
}
