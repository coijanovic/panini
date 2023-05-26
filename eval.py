import pandas as pd
import datetime as dt
import sys

LOGFILE = str(sys.argv[1])

# import data
df = pd.read_csv(LOGFILE)

df['SenderCreateTime'] = pd.to_datetime(df['SenderCreateTime'], format='%H:%M:%S.%f')
df['SenderCreateTime'] = (df['SenderCreateTime'] - dt.datetime(1900,1,1)).dt.total_seconds()
df['ReceiverDecryptTime'] = pd.to_datetime(df['ReceiverDecryptTime'], format='%H:%M:%S.%f')
df['ReceiverDecryptTime'] = (df['ReceiverDecryptTime'] - dt.datetime(1900,1,1)).dt.total_seconds()
df['latency'] = (df['ReceiverDecryptTime'] - df['SenderCreateTime'])

print(df)

print("LATENCY vs. #RECEIVERS")
print("======================")
for r in df['Receivers'].unique():
    rf = df.loc[df['Receivers'] == r].loc[df['MsgBytes'] == 512]
    print(f"Median latency for {r} Receivers: {rf['latency'].median()}")
    print(f"Std.Dev. for {r} Receivers: {rf['latency'].std()}\n")

print("\n\nLATENCY vs. #MSGBYTES")
print("=====================")
for b in df['MsgBytes'].unique():
    rf = df.loc[df['Receivers'] == 8].loc[df['MsgBytes'] == b]
    print(f"Median latency for {b} Byte messages: {rf['latency'].median()}")
    print(f"Std.Dev. for {b} Byte messages: {rf['latency'].std()}\n")
