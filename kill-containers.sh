#!/bin/bash


docker stop $(docker ps -q --filter ancestor=algorand)
docker rm $(docker ps -a -q)