package receiver

import (
    "fmt"
    "net"
    "log"

    "main/message"
    "main/utils"

    "crypto/elliptic"
    "crypto/ecdsa"
    "crypto/x509"
    "crypto/rand"
    "crypto/aes"

    "encoding/asn1"

    "github.com/zbohm/lirisi/ring"
)

var curveType = elliptic.P256
var hashFn, _ = ring.HashCodes["sha3-256"]
var vkblen = utils.GetVKByteLen()

type Receiver struct {
    PrivateKey ecdsa.PrivateKey
    SenderNym string
    Ring []ecdsa.PublicKey
    EncKey []byte
}

func New(pk *ecdsa.PrivateKey) Receiver {
    var ring []ecdsa.PublicKey
    ring = append(ring, *pk.Public().(*ecdsa.PublicKey))
    k := make([]byte, 32)
    _, _ = rand.Read(k)

    r := Receiver{*pk, "", ring, k}

    return r
}
func (r *Receiver) handleKeyReq(m message.Message) *message.Message {
    log.Println("Received KeyReq, replying with my key")

    r.SenderNym = string(m.Content)
    vk := r.PrivateKey.Public().(*ecdsa.PublicKey)
    vkbytes, err := x509.MarshalPKIXPublicKey(vk)
    if err != nil {
        panic(err)
    }
    keyRep := message.CreateMessage(byte(1), vkbytes, &r.PrivateKey)
    return &keyRep
}

func (r *Receiver) handleRingMsg(m message.Message) *message.Message {
    log.Println("Received message with keyring")

    var newRing []ecdsa.PublicKey
    for i := 0; i < len(m.Content)/vkblen; i++ {
        vk, err := x509.ParsePKIXPublicKey(m.Content[i*vkblen:i*vkblen+vkblen])
        if err != nil {
            panic(err)
        }
        newRing = append(newRing, *(vk.(*ecdsa.PublicKey)))
    }
    r.Ring = newRing
    r.SendKeyMsg()
    return nil
}

func (r *Receiver) handleCiphertxt(m message.Message) *message.Message {
    log.Println("Received ciphertxt")

    // decrypt
    dcipher, err := aes.NewCipher(r.EncKey)
    if err != nil {
        panic(err)
    }

    mprime := make([]byte, len(m.Content))
    dcipher.Decrypt(mprime, m.Content)

    // compare tags
    if mprime[0] == byte(123) {
        // r is actual receiver!
        log.Println("Received anycast message:", string(mprime[1:]))
    }
    return nil
}

func (r Receiver) SendKeyMsg() {
    log.Println("Sending my key via the anonymous channel")
    // sign key w/ lirisi
    ringps := utils.PointyPointy(r.Ring)
    status, sig := ring.Create(curveType, hashFn, &r.PrivateKey, ringps, r.EncKey, []byte{})
    if status != ring.Success {
        panic(ring.ErrorMessages[status])
    }

    sigbytes, err := asn1.Marshal(*sig)
    if err != nil {
        panic(err)
    }
    content := append(r.EncKey,sigbytes...)

    fmt.Println(sig)

    keyMsg := message.CreateMessage(byte(3), content, &r.PrivateKey)
    keyMsg.SendAnonymous(r.SenderNym)
}

/***
Message Kinds:
0: KeyReq    (s -> r)
1: KeyRep    (r -> s)
2: RingMsg   (s -> r)
3: KeyMsg    (r -> s)
4: Ciphertxt (s -> r)
***/
func (r *Receiver) Handle(m message.Message) *message.Message {
    switch m.Kind {
        case byte(0):
            return r.handleKeyReq(m)
        case byte(2):
            return r.handleRingMsg(m)
        case byte(4):
            return r.handleCiphertxt(m)
        default:
            log.Fatalln("Receiver cannot handle message of kind", m.Kind)
            return nil
    }
}

func (r *Receiver) Listen() {
    socket, err := net.Listen("tcp", ":9009")
    if err != nil {
        panic(err)
    }
    for {
        conn, err := socket.Accept()
        if err != nil {
            panic(err)
        }

        for {
            buffer := make([]byte, 4069)
            length, err := conn.Read(buffer)
            if err != nil {
                log.Println(err)
                break
            }

            message, ok := message.Deserialize(buffer[:length])
            if !ok {
                log.Println("Received malformed message")
                break
            }
            response := r.Handle(*message)
            if response != nil {
                conn.Write(response.Serialize())
            }
        }
    }
}
