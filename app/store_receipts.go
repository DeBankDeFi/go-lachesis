package app

/*
	In LRU cache data stored like value
*/

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/Fantom-foundation/go-lachesis/inter/idx"
)

type receiptRLP struct {
	Receipt *types.ReceiptForStorage
	// These fields aren't serialized in types.ReceiptForStorage
	ContractAddress common.Address
	GasUsed         uint64
}

// SetReceipts stores transaction receipts.
func (s *Store) SetReceipts(n idx.Block, receipts types.Receipts) {
	receiptsStorage := make([]*receiptRLP, len(receipts))
	for i, r := range receipts {
		receiptsStorage[i] = &receiptRLP{
			Receipt:         (*types.ReceiptForStorage)(r),
			ContractAddress: r.ContractAddress,
			GasUsed:         r.GasUsed,
		}
	}

	s.set(s.table.Receipts, n.Bytes(), receiptsStorage)

	// Add to LRU cache.
	if s.cache.Receipts != nil {
		s.cache.Receipts.Add(n, receiptsStorage)
	}
}

func (s *Store) GetReceiptsV2(blkHash common.Hash, blkN uint64, txs types.Transactions, config *params.ChainConfig) types.Receipts {
	receiptsStorage := make([]*receiptRLP, 0)
	// Get data from LRU cache first.
	if s.cache.Receipts != nil {
		if c, ok := s.cache.Receipts.Get(blkN); ok {
			if cv, ok := c.([]*receiptRLP); ok {
				receiptsStorage = cv
			}
		}
	}

	iblkN := idx.Block(blkN)
	if len(receiptsStorage) == 0 {
		s.get(s.table.Receipts, iblkN.Bytes(), &receiptsStorage)
		if receiptsStorage == nil {
			return nil
		}

		// Add to LRU cache.
		if s.cache.Receipts != nil {
			s.cache.Receipts.Add(iblkN, receiptsStorage)
		}
	}

	receipts := make(types.Receipts, len(receiptsStorage))
	for i, r := range receiptsStorage {
		receipts[i] = (*types.Receipt)(r.Receipt)
		receipts[i].ContractAddress = r.ContractAddress
		receipts[i].GasUsed = r.GasUsed
	}

	if err := receipts.DeriveFields(config, blkHash, blkN, txs); err != nil {
		log.Error("failed to derive block receipts fields", "hash", blkHash, "number", blkN, "err", err)
		return nil
	}

	return receipts
}

// GetReceipts returns stored transaction receipts.
func (s *Store) GetReceipts(n idx.Block) types.Receipts {
	var receiptsStorage *[]*receiptRLP

	// Get data from LRU cache first.
	if s.cache.Receipts != nil {
		if c, ok := s.cache.Receipts.Get(n); ok {
			if cv, ok := c.([]*receiptRLP); ok {
				receiptsStorage = &cv
			}
		}
	}

	if receiptsStorage == nil {
		receiptsStorage, _ = s.get(s.table.Receipts, n.Bytes(), &[]*receiptRLP{}).(*[]*receiptRLP)
		if receiptsStorage == nil {
			return nil
		}

		// Add to LRU cache.
		if s.cache.Receipts != nil {
			s.cache.Receipts.Add(n, *receiptsStorage)
		}
	}

	receipts := make(types.Receipts, len(*receiptsStorage))
	for i, r := range *receiptsStorage {
		receipts[i] = (*types.Receipt)(r.Receipt)
		receipts[i].ContractAddress = r.ContractAddress
		receipts[i].GasUsed = r.GasUsed
	}

	return receipts
}
