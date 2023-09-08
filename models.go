package main

type Camino4State struct {
	DocID       int64  `bson:"doc_id"`
	PayoutID    string `bson:"payout_id"`
	BankingDate string `bson:"banking_date"`
}

type CanonicalTransactionPayoutLink struct {
	HostTransactionID string `json:"host_transaction_id"`
	Host              string `json:"host"`
	PayoutID          string `json:"payout_id"`
	ProcessedAt       string `json:"processed_at"`
}
