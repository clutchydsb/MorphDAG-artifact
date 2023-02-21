package config

// Initialization parameter
const DBfile = "../dagdata/MorphDAG_%s"
const DBfile2 = "../dagdata/MorphDAG_State_%s"
const CommonLoads = "../experiments/loads.txt"
const LargeLoads = "../experiments/large_loads.txt"
const EthTxFile = "../data/EthTxs4.txt"

// PoW parameter
// const TargetBits = 14
const TargetBits = 10

// P2P parameter
const CommandLength = 12
const MaximumPoolSize = 100000

// Blockchain parameter
// const BlockSize = 1000
const BlockSize = 500
const InitialConcurrency = 80
const EpochTime = 6
const MaximumProcessors = 100000
const Frequency = 50

// Experimental results
const ExpResult1 = "../experiments/tps_appending_result.txt"
const ExpResult2 = "../experiments/tps_overall_result.txt"
