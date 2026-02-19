package testutil

import (
    "fmt"
    "net"
    "sync"
)

const localhost = "127.0.0.1"

var (
    mu          sync.Mutex
    lastPort    = 49152 
    maxPort     = 65535
)

func GetNextFreePort() (int, error) {
    mu.Lock()
    defer mu.Unlock()

    for port := lastPort; port <= maxPort; port++ {
        addr := fmt.Sprintf("%v:%d", localhost, port)
        ln, err := net.Listen("tcp", addr)
        if err == nil {
            ln.Close()
            lastPort = port + 1
            return port, nil
        }
    }

    return 0, fmt.Errorf("No free port in the range %d-%d", 49152, maxPort)
}


func MakeFreeSocketAddr() string {
	port,_ := GetNextFreePort()
    return fmt.Sprintf("%v:%d",localhost,port)
}
