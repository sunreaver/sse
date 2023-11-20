package util

import (
	"bytes"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

var (
	ErrModelReturnWrong = errors.New("model return wrong")
)

type EnginesResInterface interface {
	Reset()
}

func StreamOnData(streamData io.ReadCloser, output EnginesResInterface, onData func(EnginesResInterface)) error {
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

		if onData != nil {
			onData(output)
		}
	}
	return io.EOF
}
