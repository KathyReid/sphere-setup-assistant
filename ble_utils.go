package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ninjasphere/gatt"
)

func EnsureBLEIsUp(timeout time.Duration) error {

	success := make(chan bool, 1)

	go func() {
		for {
			cmd := exec.Command("hciconfig", "hci0")

			output, err := cmd.Output()
			logger.Infof("Output from hciconfig: %s", output)

			if err != nil {
				logger.Warningf("Failed to run hciconfig. Sleeping for 3 sec.")
				time.Sleep(time.Second * 3)
			} else {
				if strings.Contains(string(output), "UP") {
					success <- true
					break
				}

				logger.Warningf("BLE is down. Attempting to bring back up")

				cmd := exec.Command("hciconfig", "hci0", "down")

				output, err = cmd.Output()
				logger.Infof("Output from hciconfig down: %s", output)

				cmd = exec.Command("hciconfig", "hci0", "up")

				output, err = cmd.Output()
				logger.Infof("Output from hciconfig up: %s", output)

				time.Sleep(time.Second * 5)

			}
		}
	}()

	select {
	case <-time.After(timeout):
		return fmt.Errorf("Timed out after %s waiting for BLE to be up.", timeout.String())
	case <-success:
		return nil
	}

}

func ChunkWrite(req *gatt.ReadRequest, resp gatt.ReadResponseWriter, out []byte) {
	end := req.Offset + req.Cap
	if end > len(out) {
		end = len(out)
	}

	if req.Offset >= len(out) {
		resp.SetStatus(gatt.StatusUnexpectedError)
		return
	}

	resp.Write(out[req.Offset:end])

	resp.SetStatus(gatt.StatusSuccess)
}

type MultiBufferWrittenFunc func(data []byte) byte

func MultiWritableCharacteristic(char *gatt.Characteristic, maxBytes uint64, writeCompleteFunc MultiBufferWrittenFunc) {
	var offsetReqChan chan uint16
	bytesWritten := make([]byte, maxBytes)
	var expectedOffset uint16 = 0
	char.HandleNotifyFunc(
		func(r gatt.Request, n gatt.Notifier) {
			offsetReqChan = make(chan uint16, 1)
			offsetReqChan <- 0
			go func() {
				var nextOffset uint16
				count := 0
				for !n.Done() {
					nextOffset = <-offsetReqChan
					if nextOffset != 0xffff {
						buf := new(bytes.Buffer)
						err := binary.Write(buf, binary.LittleEndian, nextOffset)
						if err != nil {
							fmt.Printf("Error: %v\n", err)
						} else {
							n.Write(buf.Bytes())
						}

						fmt.Printf("Notify Count: %d, Offset: %04x\n", count, nextOffset)
						count++
					}
				}
				offsetReqChan = nil
			}()
		})
	char.HandleWriteFunc(
		func(r gatt.Request, data []byte) (status byte) {
			var offset uint16
			finalMessage := false

			buf := bytes.NewReader(data)
			err := binary.Read(buf, binary.LittleEndian, &offset)
			if err != nil {
				fmt.Println("binary.Read failed:", err)
				return gatt.StatusUnexpectedError
			}

			if offset&0x8000 != 0 {
				finalMessage = true
				offset &= 0x7fff
			}

			if offset < 0 || offset >= uint16(len(bytesWritten)) {
				fmt.Println("Invalid offset specified")
				return gatt.StatusUnexpectedError
			}

			payload := data[2:]
			copy(bytesWritten[offset:], payload)
			nextOffset := offset + uint16(len(payload))
			if expectedOffset == offset { // if we just received the last packet
				expectedOffset = nextOffset
			}

			if finalMessage && expectedOffset == nextOffset {
				if offsetReqChan != nil {
					offsetReqChan <- 0xffff
				}

				finalData := bytesWritten[:expectedOffset]
				expectedOffset = 0
				return writeCompleteFunc(finalData)
			} else {
				if offsetReqChan != nil {
					offsetReqChan <- expectedOffset
				}

				return gatt.StatusSuccess
			}
		})
}
