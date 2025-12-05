package repository

import (
	"github.com/lib/pq"
	"time"
)

const (
	retriesAmount int           = 3
	retryDelay    time.Duration = 1 * time.Second
)

func isRetryableError(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code {
		case "08000", // connection_exception
			"08003", // connection_does_not_exist
			"08006", // connection_failure
			"08001", // sqlclient_unable_to_establish_sqlconnection
			"08004", // sqlserver_rejected_establishment_of_sqlconnection
			"08007", // transaction_resolution_unknown
			"40001", // serialization_failure
			"40P01", // deadlock_detected
			"55006", // object_in_use
			"55P03": // lock_not_available
			return true
		}
	}
	return false
}

func executeWithRetry(operation func() error) error {
	var lastErr error
	currentRetryDelay := retryDelay
	for i := 0; i < retriesAmount; i++ {
		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		if !isRetryableError(lastErr) {
			return lastErr
		}

		if i < retriesAmount-1 {
			timer := time.NewTimer(currentRetryDelay)
			<-timer.C
			currentRetryDelay *= 2
		}
	}
	return lastErr
}
