#! /bin/bash

# obtain host keys
i=0
cat ./hosts.txt | while read machine
do
  if [ $i == 0 ]
  then
    id=10000
    p2p=8520
    ssh -n root@${machine} "cd ~/MorphDAG/launch; export NODE_ID=${id}; ./start_server --p2pport ${p2p} --config=true &" &
  fi

  for((j=0;j<2;j++))
  do
    id=$((i+j))
    p2p=$((7500+j))
    ssh -n root@${machine} "cd ~/MorphDAG/launch; export NODE_ID=${id}; ./start_server --p2pport ${p2p} --config=true &" &
  done
  i=$((i+2))
done

sleep 5

# download config files remotely
cat ./hosts.txt | while read machine
do
  echo "download config files on machine ${machine}"
  rsync -rtuv root@${machine}:~/MorphDAG/nodefile/* ../nodefile/config
done

# generate node files
../launch/node_config -le=false

# upload node files
j=0
cat ./hosts.txt | while read machine
do
  echo "upload node files to machine ${machine}"
  rsync -rtuv ../nodefile/ecs/node$j/* root@${machine}:~/MorphDAG/nodefile
  j=$((j+1))
done