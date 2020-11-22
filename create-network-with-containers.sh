#!/bin/bash


read -p "address of the registery: " address_of_registery
read -p "number of nodes: " number_of_nodes


for (( i=1; i<=$number_of_nodes; i++ ))
do  
   
    docker run -d --network app-tier --cap-add NET_ADMIN  --mount source=node-data,target=/application/data   algorand:latest -registery="${address_of_registery}:1234"  

    #2> /applicationdata/"$i.log"

done
