package sender

import (
    "fmt"
    "log"
    "net"
    "net/url"
    "sync"

    "main/message"
    "main/utils"

    "crypto/elliptic"
    "crypto/ecdsa"
    "crypto/x509"
    "crypto/rand"
    "crypto/aes"

    "github.com/zbohm/lirisi/ring"
    "github.com/gorilla/websocket"

    "math/big"

    "encoding/asn1"
    "encoding/json"
    "encoding/base64"
)

var curveType = elliptic.P256
var vkblen = utils.GetVKByteLen()

type Sender struct {
    PrivateKey ecdsa.PrivateKey
    NymAddr string
    ReceiverIPs []net.IP
    Plaintext string
    Ring []ecdsa.PublicKey
    ReceiverEncKeys [][]byte
    ReceiverRingSigs []ring.Signature
    Mutex *sync.Mutex
    Cond *sync.Cond
}

func New(pk *ecdsa.PrivateKey, ips []net.IP, plaintext string) Sender {
    // construct sender
    s := Sender{
        PrivateKey: *pk,
        NymAddr: "",
        ReceiverIPs: ips,
        Plaintext: plaintext,
    }
    var mutex sync.Mutex
    s.Mutex = &mutex
    s.Cond = sync.NewCond(s.Mutex)

    return s
}

func (s *Sender) handleKeyRep(m message.Message) {
    vk, err := x509.ParsePKIXPublicKey(m.Content)
    if err != nil {
        panic(err)
    }
    s.Ring = append(s.Ring, *(vk.(*ecdsa.PublicKey)))
}

func (s *Sender) handleKeyMsg(m []byte) {
    // Split content into key and sig
    key := m[:32]
    sigbytes := m[32:]
    var sig ring.Signature
    _, err := asn1.Unmarshal(sigbytes, &sig)
    if err != nil {
        panic(err)
    }
    // verify signature
    ringps := utils.PointyPointy(s.Ring)
    status := ring.Verify(&sig, ringps, key, []byte{})
    if status != ring.Success {
        panic(ring.ErrorMessages[status])
    }
    // Append key and sig
    s.ReceiverEncKeys = append(s.ReceiverEncKeys, key)
    s.ReceiverRingSigs = append(s.ReceiverRingSigs, sig)
}

func (s *Sender) BuildRingMsg() *message.Message {
    // serialize and concat ring pks
    ringBytes := make([]byte, len(s.Ring)*vkblen)
    for i, vk := range(s.Ring) {
        vkp := &vk
        vkbytes, err := x509.MarshalPKIXPublicKey(vkp)
        if err != nil {
            panic(err)
        }
        for j, b := range(vkbytes) {
            ringBytes[i*vkblen + j] = b
        }
    }

    msg := message.CreateMessage(byte(2), ringBytes, &s.PrivateKey)
    return &msg
}

func (s *Sender) BuildCiphertxtMsg() *message.Message {
    // Choose random receiver key
    index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(s.ReceiverEncKeys))))
    key := s.ReceiverEncKeys[index.Int64()]

    log.Println("Selected key", index.Int64(), "out of", len(s.ReceiverEncKeys))

    // Prepend tag to message
    p := append([]byte{byte(123)}, []byte(s.Plaintext)...)

    // Pad message to full length (might be insecure)
    for ; len(p) % aes.BlockSize != 0; {
        p = append(p, 0)
    }

    // Encrypt p with key
    cipher, err := aes.NewCipher(key)
    if err != nil {
        panic(err)
    }
    ciphertxt := make([]byte, len(p))

    cipher.Encrypt(ciphertxt, p)

    m := message.CreateMessage(byte(4), ciphertxt, &s.PrivateKey)
    return &m
}

func (s *Sender) SigsLinked() bool {
    for i := 0; i < len(s.ReceiverRingSigs); i++ {
        for j := 0; j < len(s.ReceiverRingSigs); j++ {
            if i != j {
                linked := utils.Link(&s.ReceiverRingSigs[i], &s.ReceiverRingSigs[j])
                if linked {
                    return true
                }
            }
        }
    }
    return false
}
/***
Message Kinds:
0: KeyReq    (s -> r)
1: KeyRep    (r -> s)
2: RingMsg   (s -> r)
3: KeyMsg    (r -> s)
4: Ciphertxt (s -> r)
***/
func (s *Sender) Handle(m message.Message) {
    switch m.Kind {
        case byte(1):
            s.handleKeyRep(m)
        default:
            log.Fatalln("Sender cannot handle message of kind", m.Kind)
    }
}

func (s *Sender) WebsocketListener() {
    u := url.URL{Scheme: "ws", Host: "127.0.0.1:1977", Path: "/"}

    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        panic(err)
    }
    defer c.Close()

    for {
        _, message, err := c.ReadMessage()
        if err != nil {
            panic(err)
        }
        message = message[10:]
        decoded, err := base64.StdEncoding.DecodeString(string(message))
        if err != nil {
            panic(err)
        }
        s.handleKeyMsg(decoded)

        s.Mutex.Lock()
        s.Cond.Broadcast()
        s.Mutex.Unlock()
    }
}

func endpoint(ip net.IP, port int) string {
    return fmt.Sprintf("[%v]:%d", ip.To16(), port)
}

func (s *Sender) Handshake() error {
    connections := make([]net.Conn, len(s.ReceiverIPs))
    for i, ip := range s.ReceiverIPs {
        port := 9009
        conn, err := net.Dial("tcp", endpoint(ip, port))
        if err != nil {
            return err
        }
        connections[i] = conn
    }

    log.Println("Sending key request to recipients...")
    // Step 1.: Send KeyReq
    for i := range s.ReceiverIPs {
        m := message.CreateMessage(byte(0), []byte(s.NymAddr), &s.PrivateKey)
        connections[i].Write(m.Serialize())
    }

    // Step 2: Await replies
    for _, conn := range connections {
        buffer := make([]byte, 4069)
        length, err := conn.Read(buffer)
        if err != nil {
            return err
        }
        msg, ok := message.Deserialize(buffer[:length])
        if !ok {
            panic("Malformed reply")
        }
        s.Handle(*msg)
    }

    log.Println("Distributing key ring to all recipients...")
    // Step 3: Send ring message to all recipients
    msg := s.BuildRingMsg()
    for _, conn := range connections {
        conn.Write(msg.Serialize())
    }
    return nil
}

func (s *Sender) BroadcastCiphertext() error {
    message := s.BuildCiphertxtMsg()

    log.Println("Broadcasting ciphertext")
    for _, ip := range s.ReceiverIPs {
        port := 9009
        conn, err := net.Dial("tcp", endpoint(ip, port))
        if err != nil {
            return err
        }
        conn.Write(message.Serialize())
    }
    return nil
}

func (s* Sender) GetOwnNymAddress() error {
    u := url.URL{Scheme: "ws", Host: "127.0.0.1:1977", Path: "/"}

    c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        return err
    }
    defer c.Close()

    c.WriteMessage(websocket.TextMessage, []byte(`{"type": "selfAddress"}`))

    _, message, err := c.ReadMessage()

    var data map[string]string
    json.Unmarshal(message, &data)

    s.NymAddr = data["address"]

    c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

    return nil
}

func (s *Sender) WaitForAllKeys() {
    s.Mutex.Lock()
    for ; len(s.ReceiverEncKeys) < len(s.Ring); {
        s.Cond.Wait()
    }
    s.Mutex.Unlock()
}
