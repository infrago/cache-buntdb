package cache_buntdb

import (
	"github.com/infrago/cache"
)

func Driver() cache.Driver {
	return &buntdbDriver{}
}

func init() {
	cache.Register("buntdb", Driver())
	// cache.Register("memory", Driver(":memory:"))
}
