#!/bin/sh
set -e

host="$1"
port="$2"
shift 2
cmd="$@"

echo "Waiting for $host:$port to be available..."

until curl -fs "$host":"$port" > /dev/null; do
  sleep 5
done

sleep 5

echo "$host:$port is available. Starting application..."
exec $cmd