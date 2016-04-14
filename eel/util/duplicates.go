/**
 * Copyright 2015 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"crypto/md5"
	"encoding/hex"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

type (
	// DuplicateChecker simple interface for duplicate checker.
	DuplicateChecker interface {
		IsDuplicate(Context, []byte) bool
		GetTtl() int
	}
	LocalInMemoryDupChecker struct {
		hashMap *lru.Cache
		ttl     int
		size    int
	}
)

// NewLocalInMemoryDupChecker creates a simple local in-memory de-duplication cache with optional ttl support.
func NewLocalInMemoryDupChecker(ttl int, size int) DuplicateChecker {
	dc := new(LocalInMemoryDupChecker)
	dc.hashMap, _ = lru.New(size)
	dc.ttl = ttl
	dc.size = size
	return dc
}

// getMD5Hash gets md5 hash for buf.
func (d *LocalInMemoryDupChecker) getMD5Hash(buf []byte) string {
	hasher := md5.New()
	hasher.Write(buf)
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetTtl gets ttl setting for cache.
func (d *LocalInMemoryDupChecker) GetTtl() int {
	return d.ttl
}

// IsDuplicate checks if payload was seen in past ttl ms.
func (d *LocalInMemoryDupChecker) IsDuplicate(ctx Context, payload []byte) bool {
	// if identical payload has been sent in the past ttl ms drop it
	hash := d.getMD5Hash(payload)
	if t, ok := d.hashMap.Get(hash); ok {
		if time.Now().UnixNano()-t.(int64) < int64(d.ttl*1e6) { //20000000000
			return true
		}
	}
	d.hashMap.Add(hash, time.Now().UnixNano())
	return false
}
