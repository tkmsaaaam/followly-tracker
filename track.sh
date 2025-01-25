#!/bin/bash

current_dir=$(cd $(dirname $0); pwd)

jq --version
if [ $? -gt 0 ]; then
    echo "jq is not installed."
    exit 1
fi

git --version
if [ $? -gt 0 ]; then
    echo "jq is not installed."
    exit 1
fi

ls $current_dir/follows | while read line; do
  target_dir=$current_dir/follows/$line
  if [ -d $target_dir ]; then
    cat $target_dir/result.json > $target_dir/result_old.json
    TARGET_PATH=$target_dir go run main.go
    added=`diff $target_dir/result.json $target_dir/result_old.json | grep '^<[^<]' | grep -e url -e title`
    if [ -n "$added" ]; then
      summary="auto: `cat $target_dir/setting.json | jq .url` is updated `date "+%y/%m/%d %H:%M:%S"`"
      detail="$added"
      echo $summary > $target_dir/message.txt
      echo -e "\n" >> $target_dir/message.txt
      echo -e $detail >> $target_dir/message.txt
      git config user.name  "github-actions[bot]"
      git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
      git add $target_dir/result.json
      git commit -F $target_dir/message.txt
      git push
    fi
  fi
done
