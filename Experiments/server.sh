#! /bin/bash

cat ./hosts.txt | while read machine
do
   ssh -n root@${machine} "rm -rf ~/MorphDAG/dagdata/*"
done

sleep 2

i=0
cat ./hosts.txt | while read machine
do
  # starts the tx sender
  if [ $i == 0 ]
  then
    rpc=9545
    p2p=8520
    id=10000
    ssh -n root@${machine} "cd ~/MorphDAG/launch; export NODE_ID=${id}; ./start_server --rpcport ${rpc} --p2pport ${p2p} --number $1 --sender=true &" &
  fi
  # starts MorphDAG servers
  for((j=0;j<2;j++))
  do
    rpc=$((6500+j))
    p2p=$((7500+j))
    id=$((i+j))
    ssh -n root@${machine} "cd ~/MorphDAG/launch; export NODE_ID=${id}; ./start_server --rpcport ${rpc} --p2pport ${p2p} --number $1 --cycles $2 &" &
  done
  i=$((i+2))
done
