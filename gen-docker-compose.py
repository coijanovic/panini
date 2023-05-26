#!python
#
# Usage gen-docker-compose.py <number of receivers> <msg bytes>

import sys
from random import randbytes

RECCOUNT = int(sys.argv[1])
MSGBYTES = int(sys.argv[2])

if RECCOUNT <= 1:
    print("Number of receivers must be greater than one!")
    exit

if MSGBYTES < 1:
    print("Message bytes must be greater equal 1!")
    exit

RECEIVERS = [f"receiver{i}" for i in range(RECCOUNT)]

def list_to_string(RS):
    out = ""
    for R in RS:
        out = out + R + " "
    return out

MSG=f"{MSGBYTES * '🎅'}"

HEAD=f'''
version: '2'

services:
  sender:
    image: panini
    volumes:
      - ./data:/data
    networks:
      - panininet
    init: true
    command: /run.sh tx "{MSG}" {list_to_string(RECEIVERS)}
'''
print(HEAD)

for r in RECEIVERS:
    BODY=f'''
  {r}:
    image: panini
    volumes:
      - ./data:/data
    networks:
      - panininet
    init: true
    command: /run.sh rx
    '''
    print(BODY)

TAIL=f'''
networks:
  panininet:
'''

print(TAIL)


