#! /bin/bash

i=0
cat ./hosts.txt | while read machine
do
   echo "deploy server code on machine ${machine}"
   rsync -rtuv ./install.sh root@${machine}:~
   ssh -n root@${machine} "./install.sh"
   rsync -rtuv ../launch/start_server root@${machine}:~/Occam/launch
   rsync -rtuv ../nodefile/node$i/* root@${machine}:~/Occam/nodefile
   i=$((i+1))
#   ssh -n root@${machine} "rm -rf ~/Occam/nodefile/*"
done

echo "deploy client code on machine 118.31.50.188 & 118.31.60.98"
rsync -rtuv ../launch/start_client root@118.31.50.188:~/Occam/launch
rsync -rtuv ../launch/start_client root@118.31.60.98:~/Occam/launch
rsync -rtuv ./loads.txt root@118.31.50.188:~/Occam/experiments