#!/bin/bash

read -p "The registery address: " address_of_registery
read -p "The number of nodes: " number_of_nodes

#creates the address-book.txt if it does not exists
touch addressbook.txt
touch process.pids

mkdir -p output

for (( i=1; i<=$number_of_nodes; i++ ))
do  
   ./algorand-go-implementation -registery="${address_of_registery}:1234"  >> addressbook.txt 2> output/"$i.log" &
   echo $! >> process.pids
   #sleep 0.1
done
