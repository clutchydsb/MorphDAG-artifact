#! /bin/bash

for((i=0;i<5;i++))
do
  ./Occam2 -b $1 -s $2 -f $3 -r $4
  rm -rf Morph_Test
  rm -rf Morph_Test2
  rm -rf Morph_Test3
done

