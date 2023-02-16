#! /bin/bash

rm -rf ../dagdata/*

sleep 1

for((i=0;i<$1;i++))
do
  # starts the tx sender
  if [ $i == 0 ]
    then
      rpc=9545
      p2p=8520
      id=10000
      export NODE_ID=${id}; ../launch/start_server --rpcport ${rpc} --p2pport ${p2p} --sender=true &
  fi
  sleep 1
  continue

  # starts MorphDAG servers
  rpc=$((6499+i))
  p2p=$((7499+i))
  id=$((i-1))
  export NODE_ID=${id}; ../launch/start_server --rpcport ${rpc} --p2pport ${p2p} --cycles $2 &
  sleep 1
done
