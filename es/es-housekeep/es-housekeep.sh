#!/bin/sh

while getopts ":e:p:i:d:D" opt
do
    case ${opt} in
    e ) ES_HOST=${OPTARG} ;;
    p ) ES_PORT=${OPTARG} ;;
    i ) INDEX_PREFIX=${OPTARG} ;;
    d ) DAYS=${OPTARG} ;;
    D ) MODE=DELETE ;;
    esac
done

if [ -z $MODE ] ; then
    MODE=DRYRUN
fi

if [ -z $ES_PORT ] ; then
    ES_PORT=9200
fi

if [ -z $ES_HOST ] || [ -z $INDEX_PREFIX ] || [ -z $DAYS ] ; then
    echo "options: -e [elasticsearch hostname] -p [elasticsearch port] -i [index prefix] -d [days before] -D [do delete]"
    exit 1
fi

before_date=`date "+%Y/%m/%d" -d "-$DAYS days"`
before_seconds=`date -d "$before_date" '+%s'`
indices=`curl -s http://${ES_HOST}:${ES_PORT}/_cat/indices/${INDEX_PREFIX}-* | cut -d' ' -f3`

if [ $MODE == "DRYRUN" ] ; then
  echo "Indices to delete:"
fi

cnt=0
for index in $indices;
do
  index_name=`echo $index | cut -d' ' -f3`
  date=`echo $index_name | cut -d'-' -f2 | sed 's/\./\//g'`
  seconds=`date -d "$date" '+%s'`
  if [ $seconds -lt $before_seconds ] ; then
    if [ $MODE == "DRYRUN" ] ; then
      echo $index - age seconds:$seconds
    else
      echo deleteting:
      echo $index - age seconds:$seconds
      curl -XDELETE http://${ES_HOST}:${ES_PORT}/$index_name
      echo " - done"
    fi
    cnt=$((cnt+1))
  fi
done

if [ $cnt -eq 0 ] ; then
  echo "[no indices to delete]"
fi

exit 0
