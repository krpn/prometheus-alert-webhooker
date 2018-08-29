package utils

import (
	"crypto/md5" // #nosec
	"encoding/hex"
	"strconv"
	"time"
)

// MD5Hash return MD5 hash of given string.
func MD5Hash(text string) string {
	hash := md5.Sum([]byte(text)) // #nosec
	return hex.EncodeToString(hash[:])
}

// MD5HashFromTime return MD5 hash of given time.
// It used for generate event ID.
func MD5HashFromTime(t time.Time) string {
	return MD5Hash(strconv.FormatInt(t.UnixNano(), 10))
}
