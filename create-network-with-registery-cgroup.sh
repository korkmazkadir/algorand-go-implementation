#!/bin/bash

read -p "The registery address: " address_of_registery
read -p "The number of nodes: " number_of_nodes

#creates the address-book.txt if it does not exists
touch addressbook.txt
touch process.pids

mkdir -p output

function throttle()
{

   process_index=$1
   pid=$2

   gmajor="1"
   printf -v gminor "%04x" "$process_index"
   
   # Create a net_cls cgroup
   group_name="net_cls:algorand_${process_index}"
   cgcreate -g "${group_name}"

   # Set the class id for the cgroup
   echo "0x${gmajor}${gminor}" > /sys/fs/cgroup/net_cls/slow/net_cls.classid

   # Classify packets from pid into cgroup
   cgclassify -g "${group_name}" $pid


   printf -v class_id "1:%d" "$process_index"

   # Rate limit packets in cgroup class
   #tc qdisc add dev eno1 root handle 1: htb
   tc filter add dev eno1 parent 1: handle 1: cgroup
   tc class add dev eno1 parent 1: classid $class_id htb rate 20mbps
}

#Delete previous control groups
cgdelete -r net_cls:/test-subgroup

#Adds root qdisc
tc qdisc add dev eno1 root handle 1: htb

for (( i=1; i<=$number_of_nodes; i++ ))
do  
   ./algorand-go-implementation -registery="${address_of_registery}:1234"  >> addressbook.txt 2> output/"$i.log" &
   algorand_pid=$!

   throttle $i $algorand_pid

   echo $algorand_pid >> process.pids
   #sleep 0.1
done
