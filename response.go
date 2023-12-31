package sse

import (
	"bytes"
	"io"
	"runtime"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var (
	ErrModelReturnWrong = errors.New("model return wrong")
)

type EnginesResInterface interface {
	Reset()
}

// 处理SSE数据流. 处理完毕后, 关闭streamData.
// output: 输出的结构体.
// onData: 数据处理中的, 回调函数. 其参数会完全和output的结构体一致. 如果onData返回错误, 则会中断处理.
// 此方法协程安全.
func StreamOnData(streamData io.ReadCloser, output EnginesResInterface, onData func(any) error) (err error) {
	defer func() {
		e := recover()
		if e != nil {
			stack := make([]byte, 1024)
			n := runtime.Stack(stack, false)
			if n > 0 {
				stack = stack[:n]
			}
			err = errors.Errorf("panic: %v\n%s", e, stack)
		}
	}()
	reader := newEventStreamReader(streamData, 1<<16)
	defer streamData.Close()

LOOP:
	for {
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				break LOOP
			}
			return errors.Wrap(err, "ReadEvent")
		}

		// If we get an error, ignore it.
		var msg *Event
		if msg, err = processEvent(event); err != nil {
			return errors.Wrap(err, "ProcessEvent")
		}
		if msg.Data == nil {
			continue
		}
		output.Reset()
		if bytes.Equal(msg.Data, doneSequence) {
			break LOOP
		}
		if err = jsoniter.Unmarshal(msg.Data, output); err != nil {
			return errors.Errorf("invalid json stream data: %v", err)
		}

		if err := onData(output); err != nil {
			return errors.Wrap(err, "onData")
		}
	}
	return nil
}
