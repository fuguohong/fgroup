#!/bin/zsh

cnt=$1
if [ -z $cnt ]; then
  cnt=100
fi
echo "run test $cnt times"

for i in $(seq $cnt); do
  go test -timeout 5s
  if [ $? -ne 0 ];then
    exit
  fi
done
