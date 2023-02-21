#! /bin/bash

cat ./hosts.txt | while read machine
do
  echo "kill servers on machine ${machine}"
  ssh -n root@${machine} "killall start_server 2> /dev/null" &
done

echo "kill client on 118.31.50.188 & 118.31.60.98"
ssh -n root@118.31.50.188 "killall start_client 2> /dev/null"
ssh -n root@118.31.60.98 "killall start_client 2> /dev/null"
