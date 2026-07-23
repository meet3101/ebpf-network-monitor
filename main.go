package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "time"

    "github.com/cilium/ebpf/link"
)

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: sudo ./monitor <interface>")
    }
    ifaceName := os.Args[1]

    iface, err := net.InterfaceByName(ifaceName)
    if err != nil {
        log.Fatalf("interface lookup: %s", err)
    }

    objs := xdpmonitorObjects{}
    if err := loadXdpmonitorObjects(&objs, nil); err != nil {
        log.Fatalf("loading objects: %s", err)
    }
    defer objs.Close()

    l, err := link.AttachXDP(link.XDPOptions{
    Program:   objs.XdpMonitor,
    Interface: iface.Index,
    Flags:     link.XDPGenericMode,
})
    if err != nil {
        log.Fatalf("attach XDP: %s", err)
    }
    defer l.Close()

    fmt.Println("Monitoring on", ifaceName, "- Ctrl+C to stop")

    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt)

    ticker := time.NewTicker(2 * time.Second)
    for {
        select {
        case <-ticker.C:
            printStats(&objs)
        case <-stop:
            fmt.Println("\nStopping...")
            return
        }
    }
}

func printStats(objs *xdpmonitorObjects) {
    var key uint32
    var val uint64
    entries := objs.PktCount.Iterate()
    fmt.Println("---- Packet counts by source IP ----")
    for entries.Next(&key, &val) {
        ip := make(net.IP, 4)
        ip[0] = byte(key)
        ip[1] = byte(key >> 8)
        ip[2] = byte(key >> 16)
        ip[3] = byte(key >> 24)
        fmt.Printf("%s: %d packets\n", ip.String(), val)
    }
}