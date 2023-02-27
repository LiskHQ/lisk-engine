package sync

import (
	"bytes"
	"context"

	"go.uber.org/ratelimit"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
)

type Downloader struct {
	ctx               context.Context
	chain             *blockchain.Chain
	conn              *p2p.P2P
	peerID            string
	logger            log.Logger
	start             []byte
	startHeight       uint32
	end               []byte
	endHeight         uint32
	lastFetchedHeight uint32
	lastFetchedID     []byte
	downloaded        chan *downloadedContent
	cancel            context.CancelFunc
	limitter          ratelimit.Limiter
}

type downloadedContent struct {
	block *blockchain.Block
	err   error
}

func NewDownloader(
	ctx context.Context,
	logger log.Logger,
	conn *p2p.P2P,
	chain *blockchain.Chain,
	peerID string,
	startID []byte,
	startHeight uint32,
	endID []byte,
	endHeight uint32,
) *Downloader {
	downloadLimiter := ratelimit.New(10)
	return &Downloader{
		ctx:         ctx,
		conn:        conn,
		chain:       chain,
		logger:      logger,
		peerID:      peerID,
		start:       startID,
		startHeight: startHeight,
		end:         endID,
		endHeight:   endHeight,
		limitter:    downloadLimiter,
		downloaded:  make(chan *downloadedContent, 1000),
	}
}

func (d *Downloader) Downloaded() <-chan *downloadedContent {
	return d.downloaded
}

func (d *Downloader) Start() {
	ctx, cancel := context.WithCancel(d.ctx)

	d.cancel = cancel
	defer d.cancel()
	d.logger.Infof("Downloading blocks from height %d to %d", d.startHeight, d.endHeight)
	defer close(d.downloaded)
	startID := d.start
	for {
		select {
		case <-ctx.Done():
			return
		default:
			d.limitter.Take()
			blocks, err := requestBlocksFromID(ctx, d.conn, d.peerID, startID)
			if err != nil {
				d.logger.Errorf("Failed to fetch blocks from %s with %v", d.peerID, err)
				d.downloaded <- &downloadedContent{err: err}
				return
			}
			// height asc
			blockchain.SortBlockByHeightAsc(blocks)
			for _, block := range blocks {
				d.lastFetchedID = block.Header.ID
				d.lastFetchedHeight = block.Header.Height
				IsLastBlock := bytes.Equal(block.Header.ID, d.end)
				d.downloaded <- &downloadedContent{block: block}
				if IsLastBlock {
					return
				}
			}
			d.logger.Infof("Downloaded blocks until %d", d.lastFetchedHeight)
			startID = d.lastFetchedID
		}
	}
}

func (d *Downloader) Stop() {
	d.cancel()
}
