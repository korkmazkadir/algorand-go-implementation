#!/bin/bash

read -p "The registery address: " address_of_registery
read -p "The number of nodes: " number_of_nodes

#read parameters from the config file
limit_bandwidth=$( jq .Host.LimitBandwidth config.json )
bandwidth_value=$( jq .Host.Bandwidth config.json )
apply_tc_delay=$( jq .Host.ApplyTCDelay config.json )
tc_delay_value=$( jq .Host.TCDelay config.json )
nic=$( jq -r .Host.InterfaceName config.json )


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

   # adds tasks to specific cgroup one by one
   for task_folder in /proc/"${pid}"/task/*; do
      task_id="${task_folder##*/}"
      sudo cgclassify -g "${group_name}" "${task_id}"
   done

   #tasks_of_process=$(ls /proc/"${pid}"/task)
   #echo "${task_id}" > sudo tee -a "/sys/fs/cgroup/net_cls/${group_name_suffix}/tasks" > /dev/null

   # By default gmajor is 1
   printf -v class_id "1:%x" "$process_index"

   # Rate limit packets in cgroup class
   sudo tc class add dev $nic parent 1: classid "${class_id}" htb rate "${bandwidth_value}mbit"
   # Adds delay

   if [ "$apply_tc_delay" = true ] ; then
      sudo tc qdisc add dev $nic parent "${class_id}" netem delay "${tc_delay_value}ms"
   fi

}

#Delete previous control groups
sudo cgdelete -r net_cls:/

#Defines network interface to apply tc rules
#nic="eno1"

#Delete previous tc rules
sudo tc qdisc del dev $nic root


#Adds root qdisc
sudo tc qdisc add dev $nic root handle 1: htb
sudo tc filter add dev $nic parent 1: handle 1: cgroup


# tc -s -d class show dev lo

for (( i=1; i<=$number_of_nodes; i++ ))
do

   node_id=${HOSTNAME}_${i}
   ./algorand-go-implementation -registery="${address_of_registery}:1234" -node_id="${node_id}"  >> addressbook.txt 2> output/"$i.log" &
   algorand_pid=$!

   if [ "$limit_bandwidth" = true ] ; then
      throttle $i $algorand_pid
   fi

   echo $algorand_pid >> process.pids
   #sleep 0.1
done
