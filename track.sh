#!/bin/bash

current_dir=$(cd $(dirname $0); pwd)

jq --version
if [ $? -gt 0 ]; then
    echo "jq is not installed."
    exit 1
fi

git --version
if [ $? -gt 0 ]; then
    echo "git is not installed."
    exit 1
fi

grep_command="grep"
sed_command="sed"
debug="false"

if [ "$(uname)" == 'Darwin' ]; then
  grep_command="ggrep"
  sed_command="gsed"
  debug="true"
fi

$grep_command --version
if [ $? -gt 0 ]; then
    echo "ggrep is not installed."
    exit 1
fi

$sed_command --version
if [ $? -gt 0 ]; then
    echo "gsed is not installed."
    exit 1
fi

find $current_dir/follows | grep "/setting.json" | while read line; do
  target_dir=`echo ${line//\/setting.json/}`
  if [ ! -d $target_dir ]; then
    continue
  fi
  cat $target_dir/result.json > $target_dir/result_old.json
  TARGET_PATH=$target_dir go run main.go
  added=`diff $target_dir/result.json $target_dir/result_old.json | $grep_command '^<[^<]' | $grep_command -e url -e title | $sed_command 's/^< \+//'`
  if [ -n "$added" ]; then
    summary="auto: `cat $target_dir/setting.json | jq .url` is updated `date "+%y/%m/%d %H:%M:%S"`"
    echo $summary > $target_dir/message.txt
    echo -e "\n" >> $target_dir/message.txt
    echo -e "$added" >> $target_dir/message.txt
    if [ "$debug" == "false" ]; then
      git config user.name  "github-actions[bot]"
      git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
      git add $target_dir/result.json
      git commit -F $target_dir/message.txt
      git push
    fi
  fi
done

if [ "$debug" == "false" ]; then
  git config unset user.name
  git config unset user.email
  echo "unset git config"
fi
