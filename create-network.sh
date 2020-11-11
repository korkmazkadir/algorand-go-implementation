#!/bin/bash

read -p "number of nodes: " number_of_nodes

#creates the address-book.txt if it does not exists
touch addressbook.txt
touch pids.txt

mkdir -p output

for (( i=1; i<=$number_of_nodes; i++ ))
do  
   ./algorand-go-implementation < addressbook.txt >> addressbook.txt 2> output/"$i.log" &
   echo $! >> process.pids
   sleep 0.1
done
