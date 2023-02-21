#! /bin/bash

#cat ./hosts.txt | while read machine
#do
#  echo "download node file on machine ${machine}"
#  rsync -rtuv root@${machine}:~/Occam/nodefile/* ../temp2/
#done

echo "download experimental results on machine 118.31.60.98"
rsync -rtuv root@118.31.60.98:~/Occam/experiments/* ./
ssh -n root@118.31.60.98 "rm -rf ~/Occam/experiments/*"
