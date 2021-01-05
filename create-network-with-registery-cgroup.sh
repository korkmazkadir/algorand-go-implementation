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

   printf -v gminor "%04x" "$process_index"
   

   group_name_suffix="algorand_${process_index}"

   # Create a net_cls cgroup
   group_name="net_cls:${group_name_suffix}"
   sudo cgcreate -g "${group_name}"

   # Set the class id for the cgroup
   # By default gmajor is 1
   echo_cmd="echo 0x1${gminor} > /sys/fs/cgroup/net_cls/${group_name_suffix}/net_cls.classid"
   sudo sh -c  "${echo_cmd}"

   # Classify packets from pid into cgroup
   sudo cgclassify -g "${group_name}" "${pid}"

   # By default gmajor is 1
   printf -v class_id "1:%x" "$process_index"

   # Rate limit packets in cgroup class
   sudo tc class add dev $nic parent 1: classid "${class_id}" htb rate 20mbit
   # Adds delay
   sudo tc qdisc add dev $nic parent "${class_id}" netem delay 50ms
}

#Delete previous control groups
sudo cgdelete -r net_cls:/

#Defines network interface to apply tc rules
nic="eno1"

#Delete previous tc rules
sudo tc qdisc del dev $nic root


#Adds root qdisc
sudo tc qdisc add dev $nic root handle 1: htb
sudo tc filter add dev $nic parent 1: handle 1: cgroup


# tc -s -d class show dev lo

for (( i=1; i<=$number_of_nodes; i++ ))
do  
   ./algorand-go-implementation -registery="${address_of_registery}:1234"  >> addressbook.txt 2> output/"$i.log" &
   algorand_pid=$!

   throttle $i $algorand_pid

   echo $algorand_pid >> process.pids
   #sleep 0.1
done
