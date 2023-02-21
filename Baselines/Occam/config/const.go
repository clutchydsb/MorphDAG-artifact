package config

// Initialization parameter
const DBfile = "../dagdata/Occam_%s"
const DBfile2 = "../dagdata/Occam_State_%s"
const LoadScale = "../experiments/loads.txt"
const EthTxFile = "../data/EthTxs2.txt"

// PoW parameter
const TargetBits = 14

// P2P parameter
const CommandLength = 12
const MaximumPoolSize = 100000

// Blockchain parameter
const NodeNumber = 100
const BlockSize = 1000
const MaxBlockSize = 1200
const InitialConcurrency = 16
const MaximumConcurrency = 100
const EpochTime = 6
const MaximumProcessors = 100000

// Experimental results
const ExpResult1 = "../experiments/tps_result1.txt"
const ExpResult2 = "../experiments/tps_result2.txt"
