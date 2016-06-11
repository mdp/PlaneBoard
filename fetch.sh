#!/bin/bash

timestamp=$(date +%s)
host=''
subdomain=''
account=''
cache_buster=$(cat /dev/urandom | env LC_CTYPE=C tr -dc 'a-z0-9' | fold -w 16 | head -n 1)

usage() { echo "Usage: fetch [-n <NumberOfTweets>] [-h <host>] [-t topic] account/topic/home" 1>&2; exit 1; }


while getopts ":h:n:t" o; do
  case "${o}" in
    h)
      host=${OPTARG}
      ;;
    n)
      n=${OPTARG}
      ;;
    t)
      t=true
      ;;
    *)
      usage
      ;;
  esac
done

shift "$((OPTIND - 1))"
account=${@: -1}
if [ -z "${account}" ]; then
  account='home'
fi

if [ -n "${t}" ]; then
  subdomain='t.'${subdomain}
fi

if [ -z "${n}" ]; then
  n=10
fi

if [ -z "${host}" ]; then
  usage
fi

for i in `seq 0 ${n}`;
do
  #echo c${cache_buster}.b${timestamp}.p${i}.${account}.${subdomain}${host}
  dig txt c${cache_buster}.b${timestamp}.p${i}.${account}.${subdomain}${host}| grep -A1 "ANSWER SECTION" | sed -n -e 's/.*TXT.//p'
done
