package message

import (
    "fmt"
    "net/url"
  	"github.com/gorilla/websocket"

    "encoding/json"
    "encoding/base64"

    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
)

var curveType = elliptic.P256

type Message struct {
    Kind byte
    Content []byte
    Vk ecdsa.PublicKey
    Sig []byte
}

type JsonMessage struct {
    Kind byte `json:"kind"`
    Content []byte `json:"content"`
    Vk ecdsa.PublicKey `json:"key"`
    Sig []byte `json:"signature"`
}

func CreateMessage(kind byte, content []byte, sk *ecdsa.PrivateKey) Message {
    m := &Message{kind, content, *sk.Public().(*ecdsa.PublicKey), []byte("")}
    m.Sign(sk)
    return *m
}

func (m Message) Hash() []byte {
    h := sha256.New()

    return h.Sum(append([]byte{m.Kind}, m.Content...))
}

func (m *Message) Sign(sk *ecdsa.PrivateKey) {
    sig, err := ecdsa.SignASN1(rand.Reader, sk, m.Hash())
    if (err != nil) {
        fmt.Println(err)
    }
    m.Sig = sig
}

func (m Message) Verify() bool {
    return ecdsa.VerifyASN1(&m.Vk, m.Hash(), m.Sig)
}

func (m Message) Anonymize() []byte {
    return m.Content
}

func (m *Message) Serialize() []byte {
    msg := JsonMessage {
        m.Kind,
        m.Content,
        m.Vk,
        m.Sig,
    }
    serialized, _ := json.Marshal(msg)
    return serialized
}

func Deserialize(input []byte) (*Message, bool) {
    var msg JsonMessage
    json.Unmarshal(input, &msg)
    return &Message {
        msg.Kind,
        msg.Content,
        msg.Vk,
        msg.Sig,
    }, true
}

func (m Message) SendAnonymous(nymaddr string) error {
    content := m.Anonymize()

    u := url.URL{Scheme: "ws", Host: "127.0.0.1:1977", Path: "/"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
    if err != nil {
        return err
    }
    defer c.Close()

    data := make(map[string]interface{})
    data["type"] = "send"
    data["message"] = base64.StdEncoding.EncodeToString(content)
    data["recipient"] = nymaddr
    data["withReplySurb"] = false

    serialized, _ := json.Marshal(data)
    c.WriteMessage(websocket.TextMessage, serialized)

    fmt.Println(string(serialized))
    return nil
}
