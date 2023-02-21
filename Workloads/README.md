## Generate Ethereum transaction workloads

There are two ways to fetch Ethereum transaction workloads for experimental evaluations:

1. (Recommended) We have generated the transaction workloads (the same as used in evaluations) and uploaded to the google drive. Please refer to the link to download: drive.google.com/xxx. The size of the generated workload file is about 3.1 GB, containing a total of 3,397,102 transactions in 30,000 blocks from block height 8,200,000.

2. In the directory `../Prototype/data`, complie `txData.go` and run with the number of blocks to fetch. Please refer [here](https://github.com/clutchydsb/MorphDAG-artifact/tree/master/Prototype) for detailed instructions. However, the Ethereum full-node server we built will be down for certain periods of time, frequent connection failures will be normal, thus it is better to take the first way to obtain Ethereum transaction workloads.

   