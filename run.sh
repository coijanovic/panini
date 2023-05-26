#!/bin/bash
########################################################
# Startup script for nym-latency-observer
#  
# Usage: run.sh [rx|tx] <message> <receivers0 receiver1>
########################################################

# 1. Init and start nym client
if [[ "$1" == "tx" ]]
then
    /nym/target/release/nym-client init --id tx
    /nym/target/release/nym-client run --id tx &
    sleep 25
    echo "Addr: ${@:3}"
    /runpanini -sender -m "$2" "${@:3}"
else
    /nym/target/release/nym-client init --id rx
    /nym/target/release/nym-client run --id rx &
    sleep 20
    /runpanini -receiver
fi
