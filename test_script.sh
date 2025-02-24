#!/bin/bash

MIN=1
MAX=20

# Loop indefinitely
while true
do
  # Generate a random number between 10 and 1000
  n=$((RANDOM % 991 + 10))
  t=$((RANDOM % (MAX - MIN + 1) + MIN))

  # Run the command with the current random value of n
  # bin/serverledge-cli invoke -f fibo -p "n:$n" &
  
  bin/serverledge-cli invoke -f kmeans --params_file examples/kmeans_clustering/input.json
  bin/serverledge-cli invoke -f lif -p "n:2000"
  bin/serverledge-cli invoke -f rsa -p "m:ciaoatuttiquantillllll1111111111"
  wait  
  # Wait for 10 seconds before the next iteration
  sleep $t
done

bin/serverledge-cli invoke -f rsa -p m:ciaoatuttiquantillllll1111111111
