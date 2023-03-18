package cache_buntdb

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/infrago/cache"
	"github.com/tidwall/buntdb"
)

var (
	errInvalidCacheConnection = errors.New("Invalid cache connection.")
	errEmptyData              = errors.New("Empty cache data.")
)

type (
	buntdbDriver struct {
		store string
	}
	buntdbConnect struct {
		mutex sync.RWMutex

		name    string
		config  cache.Config
		setting buntdbSetting

		db *buntdb.DB
	}
	buntdbSetting struct {
		Store string
	}
)

// 连接
func (driver *buntdbDriver) Connect(name string, config cache.Config) (cache.Connect, error) {
	//获取配置信息
	setting := buntdbSetting{
		Store: driver.store,
	}

	//创建目录，如果不存在
	dir := path.Dir(setting.Store)
	_, e := os.Stat(dir)
	if e != nil {
		os.MkdirAll(dir, 0700)
	}

	if vv, ok := config.Setting["file"].(string); ok && vv != "" {
		setting.Store = vv
	}
	if vv, ok := config.Setting["store"].(string); ok && vv != "" {
		setting.Store = vv
	}

	return &buntdbConnect{
		name: name, config: config, setting: setting,
	}, nil
}

// 打开连接
func (this *buntdbConnect) Open() error {
	if this.setting.Store == "" {
		return errors.New("无效缓存存储")
	}
	db, err := buntdb.Open(this.setting.Store)
	if err != nil {
		return err
	}
	this.db = db
	return nil
}

// 关闭连接
func (this *buntdbConnect) Close() error {
	if this.db != nil {
		if err := this.db.Close(); err != nil {
			return err
		}
	}
	return nil
}

// 查询缓存，
func (this *buntdbConnect) Read(key string) ([]byte, error) {
	if this.db == nil {
		return nil, errInvalidCacheConnection
	}

	value := ""
	err := this.db.View(func(tx *buntdb.Tx) error {
		vvv, err := tx.Get(key)
		if err != nil {
			return err
		}
		value = vvv
		return nil
	})
	if err == buntdb.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 使用base64转换
	return base64.StdEncoding.DecodeString(value)
}

// 更新缓存
func (this *buntdbConnect) Write(key string, data []byte, expiry time.Duration) error {
	if this.db == nil {
		return errInvalidCacheConnection
	}

	value := base64.StdEncoding.EncodeToString(data)
	if value == "" {
		return errEmptyData
	}

	return this.db.Update(func(tx *buntdb.Tx) error {
		opts := &buntdb.SetOptions{Expires: false}
		if expiry > 0 {
			opts.Expires = true
			opts.TTL = expiry
		}
		_, _, err := tx.Set(key, value, opts)
		return err
	})
}

// 查询缓存，
func (this *buntdbConnect) Exists(key string) (bool, error) {
	if this.db == nil {
		return false, errInvalidCacheConnection
	}

	err := this.db.View(func(tx *buntdb.Tx) error {
		_, err := tx.Get(key)
		return err
	})
	if err != nil {
		if err == buntdb.ErrNotFound {
			return true, nil
		}
	}
	return false, nil
}

// 删除缓存
func (this *buntdbConnect) Delete(key string) error {
	if this.db == nil {
		return errInvalidCacheConnection
	}

	return this.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(key)
		return err
	})
}

func (this *buntdbConnect) Serial(key string, start, step int64, expiry time.Duration) (int64, error) {
	value := start

	if data, err := this.Read(key); err == nil {
		num, err := strconv.ParseInt(string(data), 10, 64)
		if err == nil {
			value = num
		}
	}
	//加数字
	value += step

	//写入值
	data := []byte(fmt.Sprintf("%v", value))
	err := this.Write(key, data, expiry)
	if err != nil {
		return int64(0), err
	}

	return value, nil
}

func (this *buntdbConnect) Clear(prefix string) error {
	if this.db == nil {
		return errors.New("连接失败")
	}

	keys, err := this.Keys(prefix)
	if err != nil {
		return err
	}

	return this.db.Update(func(tx *buntdb.Tx) error {
		for _, key := range keys {
			_, err := tx.Delete(key)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
func (this *buntdbConnect) Keys(prefix string) ([]string, error) {
	if this.db == nil {
		return nil, errors.New("连接失败")
	}

	keys := []string{}
	err := this.db.View(func(tx *buntdb.Tx) error {
		tx.AscendKeys(prefix+"*", func(k, v string) bool {
			keys = append(keys, k)
			return true
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return keys, nil
}
