package main

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func BenchmarkDynamicMapping(b *testing.B) {
	f, err := os.Open("output.json")
	if err != nil {
		panic("file")
	}

	output, err := io.ReadAll(f)
	if err != nil {
		panic("file read")
	}
	err = f.Close()
	if err != nil {
		panic("file close")
	}

	mapper := GetStream(output)
	for i := 0; i < b.N; i++ {
		mapper.Process(map[string]map[string]any{
			"host_enum": {"host": "Camino4"},
			"state": {
				"doc_id":       300012,
				"payout_id":    "12321",
				"banking_date": "2023-01-01T00:00:00Z",
			},
		})
	}
}

func BenchmarkStrictMapping(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MapToTBPayoutLink(Camino4State{
			DocID:       300012,
			PayoutID:    "12321",
			BankingDate: "2023-01-01T00:00:00Z",
		})
	}
}

func MapToTBPayoutLink(data Camino4State) (CanonicalTransactionPayoutLink, error) {
	_, err := time.Parse(time.RFC3339, data.BankingDate)
	if err != nil {
		return CanonicalTransactionPayoutLink{}, fmt.Errorf("error on banking date(%s): %w", data.BankingDate, err)
	}

	if data.DocID == 0 {
		return CanonicalTransactionPayoutLink{}, fmt.Errorf("empty docID for payout link")
	}

	if data.PayoutID == "" {
		return CanonicalTransactionPayoutLink{}, fmt.Errorf("empty payoutID for payout link")
	}

	return CanonicalTransactionPayoutLink{
		HostTransactionID: fmt.Sprint(data.DocID),
		Host:              "Camino4",
		PayoutID:          data.PayoutID,
		ProcessedAt:       data.BankingDate,
	}, nil
}
