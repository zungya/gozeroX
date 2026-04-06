package idgen

import (
	"errors"
	"sync"
	"time"
)

// 核心配置（毕设单机部署足够用）
const (
	// 起始时间戳（2024-01-01），缩短ID长度
	epoch = 1704067200000
	// 机器ID位数（最多支持1024台机器）
	workerIDBits = 10
	// 序列号位数（每毫秒最多生成4096个ID）
	sequenceBits = 12

	// 计算最大值
	maxWorkerID = -1 ^ (-1 << workerIDBits)
	maxSequence = -1 ^ (-1 << sequenceBits)

	// 位移数
	workerIDShift  = sequenceBits
	timestampShift = sequenceBits + workerIDBits
)

// Snowflake 雪花ID生成器实例
type Snowflake struct {
	mu        sync.Mutex // 并发安全锁
	workerID  int64      // 机器ID
	timestamp int64      // 上次生成ID的时间戳
	sequence  int64      // 毫秒内序列号
}

var (
	instance *Snowflake
	once     sync.Once
)

// Init 初始化生成器（单机部署workerID填0）
func Init(workerID int64) error {
	var err error
	once.Do(func() {
		if workerID < 0 || workerID > maxWorkerID {
			err = errors.New("workerID必须在0-1023之间")
			return
		}
		instance = &Snowflake{
			workerID:  workerID,
			timestamp: 0,
			sequence:  0,
		}
	})
	return err
}

// GenID 生成雪花ID（int64类型，存数据库）
func GenID() (int64, error) {
	if instance == nil {
		return 0, errors.New("请先调用Init初始化雪花ID生成器")
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	// 获取当前毫秒时间戳
	now := time.Now().UnixMilli()

	// 处理时钟回拨
	if now < instance.timestamp {
		return 0, errors.New("系统时钟回拨，无法生成ID")
	}

	// 同一毫秒，序列号自增
	if now == instance.timestamp {
		instance.sequence = (instance.sequence + 1) & maxSequence
		// 序列号溢出，等待下一毫秒
		if instance.sequence == 0 {
			for now <= instance.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		// 新毫秒，重置序列号
		instance.sequence = 0
	}

	instance.timestamp = now

	// 拼接ID：时间戳 + 机器ID + 序列号
	id := ((now - epoch) << timestampShift) | (instance.workerID << workerIDShift) | instance.sequence
	return id, nil
}

// GenIDStr 生成雪花ID并转为字符串（返回给前端，避免精度丢失）
func GenIDStr() (string, error) {
	id, err := GenID()
	if err != nil {
		return "", err
	}
	return string(rune(id)), err
}
