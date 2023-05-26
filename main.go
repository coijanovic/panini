package main

import (
    "main/sender"
    "main/receiver"
    "main/utils"

    "flag"
    "fmt"
    "log"
    "os"

    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
)


var curveType = elliptic.P256


func main() {
    senderFlag := flag.Bool("sender", false, "Role = Sender")
    receiverFlag := flag.Bool("receiver", false, "Role = Receiver")
    messageFlag := flag.String("m", "", "Message to be anycast (Only for sender)")
    flag.Parse()

    addr := flag.Args()
    ips := utils.ParseIPs(addr)

    privateKey, _ := ecdsa.GenerateKey(curveType(), rand.Reader)

    logPath := "/data/panini.log"
    logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
    if err != nil {
        log.Panic(err)
    }
    defer logFile.Close()

    log.SetOutput(logFile)
    log.SetFlags(log.LstdFlags | log.Lmicroseconds)

    log.Println("\n\nSTARTING PANINI RUN")


    if *senderFlag {
        if *messageFlag != "" && len(addr) >= 1 {
            log.Println("Creating Sender")
            log.Println("# Receiver:", len(ips))
            log.Println("# MSGBytes:", len(*messageFlag))

            client := sender.New(privateKey, ips, *messageFlag)


            err := client.GetOwnNymAddress()
            if err != nil {
                panic(err)
            }
            log.Println("My own nym address is:", client.NymAddr)
            go client.WebsocketListener()
            err = client.Handshake()
            if err != nil {
                panic(err)
            }

            client.WaitForAllKeys()

            log.Println("I have received all keys")
            if client.SigsLinked() {
                panic("Got two keys of the same recipient")
            }

            err = client.BroadcastCiphertext()
            if err != nil {
                panic(err)
            }
        } else {
            fmt.Println("To run sender use -sender -m 'message' -n <sender nym addr> <list of receiver IPs>")
        }
    } else if *receiverFlag {
        log.Println("Creating Receiver")
        client := receiver.New(privateKey)
        client.Listen()
    } else {
        fmt.Println("Either use -sender or -receiver!")
    }
}
