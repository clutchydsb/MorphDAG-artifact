#! /bin/bash

#../launch/start_client --dest $1 --rpcport 5000 ssg
#
#../launch/start_client --dest $1 --rpcport 5000 sts -c 30 &
#
#sleep 10
#
#../launch/start_client --dest $1 --rpcport 5000 ob

# ssh -n root@118.31.50.188 "cd ~/Occam/launch; ./start_client --rpcport 5000 ssg"

ssh -n root@118.31.50.188 "cd ~/Occam/launch; ./start_client --rpcport 9545 sts -c $1 &" &

sleep 12

ssh -n root@118.31.60.98 "cd ~/Occam/launch; ./start_client --rpcport 5000 ob -c $2 &"
