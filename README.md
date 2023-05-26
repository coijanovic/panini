# Panini - Anonymous Anycast

This repository contains a prototype implementation of Panini.
To recreate the end-to-end latency benchmarks from the paper proceed as follows:

1. Generate a docker-compose file: `python gen-docker-compose.py <#Receiver> <Msg. Len> >> docker-compose.yml`
2. Start the containers: `docker-compose up`
3. Log data will be collected in `data/panini.log`
