#!/bin/sh

DF_DIR=$1

# disk size and usage
df=$(df $DF_DIR | tail -1)
name=$(echo $df | awk '{print $1}')
size=$(echo $df | awk '{print $2}')
used=$(echo $df | awk '{print $3}')
free=$(echo $df | awk '{print $4}')

# Output json
echo {\"filesystem\":\"${name}\",\"size\":${size},\"used\":${used},\"free\":${free}}

return 0
