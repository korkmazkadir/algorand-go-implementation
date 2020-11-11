 #!/bin/bash

input="./process.pids"

while IFS= read -r line
do
  
  kill -9 "$line"

done < "$input"
