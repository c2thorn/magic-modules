#!/usr/bin/env bash
set -e
outputPath=$1
files=$2

yamllist=()
erblist=()
otherlist=()
yamlstring=""
erbstring=""

IFS=',' read -ra file <<< "$files"
for i in "${file[@]}"; do
  filename=$(basename -- "$i")
  extension="${filename##*.}"
  if [[ $extension == "yaml" ]]; then
    yamllist+=($i)
    if [[ $yamlstring == "" ]]; then
        yamlstring=$i
    else
        yamlstring="${yamlstring},${i}"
    fi
  elif [[ $extension == "erb" ]]; then
    erblist+=($i)
    if [[ $erbstring == "" ]]; then
        erbstring=$i
    else
        erbstring="${erbstring},${i}"
    fi
  else
    otherlist+=($i)
  fi
done
# echo "parsing results"
# echo "${yamllist[*]}"
# echo $yamlstring
# echo "${erblist[*]}"
# echo $erbstring
# echo "${otherlist[*]}"
# bundle exec compiler.rb -e terraform -o $1 -v beta -p products/accessapproval --go-yaml-files $yamlstring
bundle exec compiler.rb -e terraform -o $1 -v beta -a --go-yaml-files $yamlstring

go run . --yaml-temp
go run . --template-temp $erbstring
go run . --handwritten-temp $erbstring

for i in "${otherlist[@]}"
do
    cp "$i" "${i}.temp" 
done