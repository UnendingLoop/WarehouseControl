// Package cleaner provides a struct BookCleaner with only method StartBookCleaner to periodically check and cancel expired bookings in DB
package cleaner

import (
	"context"
	"log"
	"time"
)

type BookCleaner struct {
	bsvc CleanerService
}

type CleanerService interface {
	CleanExpiredBooks(ctx context.Context) error
}

func NewBookCleaner(svc CleanerService) *BookCleaner {
	return &BookCleaner{bsvc: svc}
}

func (bc *BookCleaner) StartBookCleaner(ctx context.Context, interval int) {
	if interval <= 0 {
		log.Println("Invalid interval provided for running BookCleaner. Using default value: 60 seconds")
		interval = 60
	}
	tckr := time.NewTicker(time.Duration(interval) * time.Second)

	go func() {
		defer tckr.Stop()
		for {
			select {
			case <-tckr.C:
				bc.runOnce()
			case <-ctx.Done():
				log.Println("BookCleaner ctx is cancelled. Finishing work...")
				return
			}
		}
	}()

	log.Println("BookCleaner started working...")
}

func (bc *BookCleaner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bc.bsvc.CleanExpiredBooks(ctx)
	if err != nil {
		log.Printf("Failed to cancel expired bookings: %v", err)
	}
}
