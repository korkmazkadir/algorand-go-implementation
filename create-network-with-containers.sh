#!/bin/bash

read -p "number of nodes: " number_of_nodes

for (( i=1; i<=$number_of_nodes; i++ ))
do  
   
    docker run -d --network app-tier --cap-add NET_ADMIN  --mount source=node-data,target=/application/data   algorand:latest -registery="6e0525c7945c:1234"  

    #2> /applicationdata/"$i.log"

done