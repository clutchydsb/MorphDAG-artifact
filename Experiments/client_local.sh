#! /bin/bash

../Prototype/launch/start_client --rpcport 9545 sts -c $1 -l $2 &

sleep 12

../Prototype/launch/start_client --rpcport 6500 ob -c $3
