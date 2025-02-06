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
  bin/serverledge-cli invoke -f fibo -p "n:$n" &
  wait  
  # Wait for 10 seconds before the next iteration
  sleep $t
done

