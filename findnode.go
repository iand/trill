package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"

	tutil "github.com/plprobelab/go-kademlia/examples/util"
	"github.com/plprobelab/go-kademlia/kad"
	"github.com/plprobelab/go-kademlia/key"
	"github.com/plprobelab/go-kademlia/routing/simplert"

	"github.com/iand/zikade/core"
	"github.com/iand/zikade/kademlia"
	"github.com/iand/zikade/libp2p"
)

var findNodeCommand = &cli.Command{
	Name:   "findnode",
	Usage:  "find the address of a node in the network",
	Action: FindNode,
	Flags: mergeFlags(loggingFlags, []cli.Flag{
		&cli.StringFlag{
			Name:  "target",
			Usage: "Target peer id",
			Value: "12D3KooWGWcyxn3JfihYiu2HspbE5XHzfgZiLwihVCeyXQQU8yC1",
		},
		&cli.StringFlag{
			Name:  "bootstrap",
			Usage: "Bootstrap peer id",
			Value: "12D3KooWGjgvfDkpuVAoNhd7PRRvMTEG4ZgzHBFURqDe1mqEzAMS",
		},
	}),
}

func FindNode(cc *cli.Context) error {
	ctx := cc.Context
	setupLogging(cc)

	bootstrapNodeID := cc.String("bootstrap")
	targetNodeID := cc.String("target")

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// create a libp2p host
	h, err := tutil.Libp2pHost(ctx, "8888")
	if err != nil {
		return fmt.Errorf("failed to create libp2p host: %w", err)
	}

	pid := libp2p.NewPeerID(h.ID())

	// create a simple routing table, with bucket size 20
	rt := simplert.New[key.Key256, kad.NodeID[key.Key256]](pid, 20)

	rtr := libp2p.NewRouter(h, 10*time.Minute)

	// friend is the first peer we know in the IPFS DHT network (bootstrap node)
	friend, err := peer.Decode(bootstrapNodeID)
	if err != nil {
		return fmt.Errorf("failed to parse peer id of friend: %w", err)
	}

	// multiaddress of friend
	a, err := multiaddr.NewMultiaddr("/ip4/45.32.75.236/udp/4001/quic")
	if err != nil {
		return fmt.Errorf("failed to parse multiaddr of friend: %w", err)
	}

	slog.Info("bootstrapping from friend", "id", friend)

	friendAddr := peer.AddrInfo{ID: friend, Addrs: []multiaddr.Multiaddr{a}}
	if err := h.Connect(ctx, friendAddr); err != nil {
		return fmt.Errorf("failed to connect to friend: %w", err)
	}
	slog.Info("connected to friend", "id", friend)

	// target is the peer we want to find
	target, err := peer.Decode(targetNodeID)
	if err != nil {
		return fmt.Errorf("failed to decode peer id: %w", err)
	}
	targetID := libp2p.NewPeerID(target)

	cfg := kademlia.DefaultConfig()
	cfg.RequestConcurrency = 1
	cfg.RequestTimeout = 5 * time.Second

	d, err := kademlia.NewDht[key.Key256, multiaddr.Multiaddr](pid, rtr, rt, cfg)
	if err != nil {
		return fmt.Errorf("failed to create dht: %w", err)
	}

	d.AddNodes(ctx, []kad.NodeInfo[key.Key256, multiaddr.Multiaddr]{libp2p.NewAddrInfo(friendAddr)})
	time.Sleep(1 * time.Second)

	var foundNode core.Node[key.Key256, multiaddr.Multiaddr]
	fn := func(ctx context.Context, node core.Node[key.Key256, multiaddr.Multiaddr], stats core.QueryStats) error {
		slog.Info("visiting node", "id", node.ID())
		if key.Equal(node.ID().Key(), targetID.Key()) {
			foundNode = node
			return core.SkipRemaining
		}
		return nil
	}

	// Run a query to find the node, using the DHT's own Query method if it has one.
	slog.Info("starting query")
	_, err = core.Query[key.Key256, multiaddr.Multiaddr](ctx, d, targetID.Key(), fn)
	if err != nil {
		return fmt.Errorf("failed to run query: %w", err)
	}

	slog.Info("found node", "id", foundNode.ID())
	for _, addr := range foundNode.Addresses() {
		slog.Info("node address", "addr", addr)
	}

	return nil
}
