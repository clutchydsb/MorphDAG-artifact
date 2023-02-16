#! /bin/bash

i=0
client1=''
client2=''
cat ./hosts.txt | while read machine
do
  if [ $i == 1 ]
  then
    client1=machine
  elif [ $i == 2 ]
  then
    client2=machine
  elif [ $i == 0 ]
  then
    continue
  else
    break
  fi
  i=$((i+1))
done

ssh -n root@${client1} "cd ~/MorphDAG/launch; ./start_client --rpcport 9545 sts -c $1 -l $2 &" &

sleep 12

ssh -n root@${client2} "cd ~/MorphDAG/launch; ./start_client --rpcport 6500 ob -c $3 &"