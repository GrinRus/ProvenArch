package providercommon

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
)

func CaptureCommandStream(reader io.Reader, sink *bytes.Buffer, task acpruntime.Task, stream acpruntime.OutputStream) error {
	if sink == nil {
		return errors.New("capture sink is nil")
	}
	bufReader := bufio.NewReader(reader)
	for {
		part, err := bufReader.ReadString('\n')
		if len(part) > 0 {
			sink.WriteString(part)
			if task.OnOutput != nil {
				task.OnOutput(acpruntime.OutputChunk{
					Stream: stream,
					Text:   strings.TrimRight(part, "\r\n"),
				})
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
