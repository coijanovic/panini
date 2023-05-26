package utils

import (
    "net"
    "fmt"
    "log"
    "bytes"

    "crypto/elliptic"
    "crypto/ecdsa"
    "crypto/rand"
    "crypto/x509"

    "github.com/zbohm/lirisi/ring"
)

func ParseIPs(addr []string) []net.IP {
    var ips []net.IP
    for _, a := range addr {
        ip := net.ParseIP(a)
        if ip == nil {
            // Check if we can resolve the hostname
            addr, err := net.LookupIP(a)
            if err != nil {
                panic(fmt.Sprintf("Invalid IP/hostname: %v", a))
            }
            log.Printf("Resolved %s as %s\n", a, addr[0])
            ips = append(ips, addr[0])
        } else {
            ips = append(ips, ip)
        }
    }
    return ips
}

var curveType = elliptic.P256

func GetVKByteLen() int {
    privateKey, _ := ecdsa.GenerateKey(curveType(), rand.Reader)
    vk := privateKey.Public().(*ecdsa.PublicKey)
    vkbytes, _ := x509.MarshalPKIXPublicKey(vk)
    return len(vkbytes)
}

func PointyPointy(ring []ecdsa.PublicKey) []*ecdsa.PublicKey {
    var ringps []*ecdsa.PublicKey
    for i, _ := range ring {
        ringps = append(ringps, &ring[i])
    }
    return ringps
}

func Link(a, b *ring.Signature) bool {
    return bytes.Equal(a.KeyImage.X, b.KeyImage.X) && bytes.Equal(a.KeyImage.Y, b.KeyImage.Y)
}
