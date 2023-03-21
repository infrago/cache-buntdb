package cache_buntdb

import (
	"github.com/infrago/cache"
	"github.com/infrago/infra"
)

func Driver() cache.Driver {
	return &buntdbDriver{}
}

func init() {
	infra.Register("buntdb", Driver())
}
