## Deploy & Run

- #### Run in a local node environment

Step 1: configure the node connection file

```
./nodeconfig_local.sh $num
```

`$num` denotes the number of MorphDAG nodes run locally. Generated node address files are stored in `../Prototype/nodefile`.

Step 2: run the daemon (server program) of MorphDAG

```
./server_local.sh $num $cycles
```

`$cycles` denotes the number of cycles MorphDAG servers run. It will take 2~3 minutes for node discovery and connections. After all server processes print `Server $id starts` on the terminal , please run the client program immediately.

Step 3: run the client program of MorphDAG

```
./client_local.sh $send $large $observe
```

`$send` denotes the number of transaction sending cycles. `$large` denotes whether to use the large workload size (0 is the Ethereum workload size and 1 is the large workload size). `loads.txt` depicts the Ethereum workload size, and `large_loads.txt` depicts the large workload size. You can modify `large_loads.txt` to change the transaction sending rate (each row represents the number of transactions sent per second). $observe denotes the number of cycles run by anther client process to observe system tps and latency.

Step 4: kill the server and client programs

```
./kill_local.sh
```

After running the speicific number of cycles, the server program can automatically stop, or you can dircetly kill the server and client programs. You can find two experimental results in the current directory, where `tps_appending_result.txt` presents the appending tps and latency of MorphDAG, and `tps_overall_result.txt`  presents the system overall tps and latency.

- #### Run in a distributed node environment

Step 1: modify `hosts.txt` and deploy compiled codes to each remote node

```
./deploy.sh
```

Please enter the ip address of each remote node in `hosts.txt`. Notice that you should enable your work computer to login in these remote nodes without passwords in advance.

Step 2: configure the node connection file

```
./nodeconfig.sh $num
```

`$num` denotes the number of MorphDAG nodes run in a distributed environment. Generated node address files are stored in `~/MorphDAG/nodefile` in each remote node.

Step 3: run the daemon (server program) of MorphDAG

```
./server.sh $num $cycles
```

Likewise, wait for each remote node to complete connections and print `Server $id starts` on the terminal.

Step 4: run the client program of MorphDAG

```
./client.sh $send $large $observe
```

We use the first two remote machines to run client programs for tx sending and observation by default.

Step 4: kill the server and client programs

```
./kill.sh
```

Step 5: download the experimental results

```
./download.sh
```

Since the observation results are stored on the remote machine who runs client programs, please download the relevant files `tps_appending_result.txt` and `tps_overall_result.txt` from the client side.