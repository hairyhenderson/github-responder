#!/bin/bash
#
# A sample script that formats a couple types of GitHub Webhook payloads.
# Intended as an example for use with github-responder
#
# Usage:
#   cat payload | ./format_webhook <event type> <delivery ID>
#

eventType=$1
deliveryID=$2

N="\033[0m"
B="\033[1m"

echo -e "$B---[ $eventType ]---$N"
echo -e "${B}delivery ID:$N\t$deliveryID"

read -r payload

payloadPart () {
  echo $payload | jq -crM $1
}

if [ "$eventType" == "push" ]; then
  echo -e "${B}commit ID:$N\t$(payloadPart .commits[0].id)"
  echo -e "${B}commit URL:$N\t$(payloadPart .commits[0].url)"
  echo -e "${B}message:$N"
  payloadPart .commits[0].message
elif [ "$eventType" == "ping" ]; then
  echo -e "${B}zen:$N\t$(payloadPart .zen)"
  echo -e "${B}events:$N\t$(payloadPart .hook.events)"
else
  payloadPart .
fi

echo -e "$B---[ end of $eventType ]---$N"
