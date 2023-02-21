#! /bin/bash

cat ./hosts.txt | while read machine
do
   ssh -n root@${machine} "rm -rf ~/Occam/dagdata/*"
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
    ssh -n root@${machine} "cd ~/Occam/launch; export NODE_ID=${id}; ./start_server --rpcport ${rpc} --p2pport ${p2p} --sender=true &" &
  fi
  # starts Occam servers
  for((j=0;j<2;j++))
  do
    rpc=$((5000+j))
    p2p=$((7000+j))
    id=$((i+j))
    ssh -n root@${machine} "cd ~/Occam/launch; export NODE_ID=${id}; ./start_server --rpcport ${rpc} --p2pport ${p2p} --cycles $1 &" &
  done
  i=$((i+2))
done

#for((i=0;i<$1;i++))
#do
#  rpc=$((9545+i))
#  p2p=$((8520+i))
#  id=$((3001+i))
#  export NODE_ID=${id}; ../launch/start_server --rpcport ${rpc} --p2pport ${p2p} &
#  sleep 1
#done
