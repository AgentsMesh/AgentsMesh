package blockstoreservice

import "time"

var timeNowUTC = func() time.Time {
	return time.Now().UTC()
}
