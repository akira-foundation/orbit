package proxy

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	fcgiVersion1 = 1

	typeBeginRequest = 1
	typeAbortRequest = 2
	typeEndRequest   = 3
	typeParams       = 4
	typeStdin        = 5
	typeStdout       = 6
	typeStderr       = 7

	roleResponder = 1

	maxWrite = 65535
)

type fcgiHeader struct {
	version       uint8
	reqType       uint8
	requestID     uint16
	contentLength uint16
	paddingLength uint8
}

func writeHeader(w io.Writer, h fcgiHeader) error {
	buf := [8]byte{
		h.version, h.reqType,
		byte(h.requestID >> 8), byte(h.requestID),
		byte(h.contentLength >> 8), byte(h.contentLength),
		h.paddingLength, 0,
	}
	_, err := w.Write(buf[:])
	return err
}

func writeRecord(w io.Writer, reqType uint8, requestID uint16, content []byte) error {
	for len(content) > 0 || content == nil {
		chunk := content
		if len(chunk) > maxWrite {
			chunk = chunk[:maxWrite]
		}
		pad := (8 - len(chunk)%8) % 8
		if err := writeHeader(w, fcgiHeader{
			version: fcgiVersion1, reqType: reqType, requestID: requestID,
			contentLength: uint16(len(chunk)), paddingLength: uint8(pad),
		}); err != nil {
			return err
		}
		if len(chunk) > 0 {
			if _, err := w.Write(chunk); err != nil {
				return err
			}
		}
		if pad > 0 {
			if _, err := w.Write(make([]byte, pad)); err != nil {
				return err
			}
		}
		content = content[len(chunk):]
		if len(chunk) == 0 {
			break
		}
	}
	return nil
}

func writeBeginRequest(w io.Writer, requestID uint16) error {
	body := [8]byte{0, roleResponder, 0, 0, 0, 0, 0, 0}
	return writeRecord(w, typeBeginRequest, requestID, body[:])
}

func encodeParamLen(buf *[]byte, n int) {
	if n < 128 {
		*buf = append(*buf, byte(n))
		return
	}
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(n)|0x80000000)
	*buf = append(*buf, b[:]...)
}

func encodeParams(pairs [][2]string) []byte {
	var buf []byte
	for _, kv := range pairs {
		k, v := kv[0], kv[1]
		encodeParamLen(&buf, len(k))
		encodeParamLen(&buf, len(v))
		buf = append(buf, k...)
		buf = append(buf, v...)
	}
	return buf
}

func writeParams(w io.Writer, requestID uint16, pairs [][2]string) error {
	body := encodeParams(pairs)
	if err := writeRecord(w, typeParams, requestID, body); err != nil {
		return err
	}
	return writeRecord(w, typeParams, requestID, nil)
}

func writeStdin(w io.Writer, requestID uint16, body io.Reader) error {
	if body == nil {
		return writeRecord(w, typeStdin, requestID, nil)
	}
	buf := make([]byte, maxWrite)
	for {
		n, err := body.Read(buf)
		if n > 0 {
			if werr := writeRecord(w, typeStdin, requestID, buf[:n]); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return writeRecord(w, typeStdin, requestID, nil)
}

type fcgiResponse struct {
	stdout []byte
	stderr []byte
}

func readResponse(r io.Reader) (*fcgiResponse, error) {
	br := bufio.NewReaderSize(r, 32*1024)
	resp := &fcgiResponse{}
	for {
		var hdr [8]byte
		if _, err := io.ReadFull(br, hdr[:]); err != nil {
			if err == io.EOF {
				return resp, nil
			}
			return nil, err
		}
		reqType := hdr[1]
		contentLen := int(hdr[4])<<8 | int(hdr[5])
		padLen := int(hdr[6])
		content := make([]byte, contentLen)
		if contentLen > 0 {
			if _, err := io.ReadFull(br, content); err != nil {
				return nil, err
			}
		}
		if padLen > 0 {
			if _, err := io.CopyN(io.Discard, br, int64(padLen)); err != nil {
				return nil, err
			}
		}
		switch reqType {
		case typeStdout:
			resp.stdout = append(resp.stdout, content...)
		case typeStderr:
			resp.stderr = append(resp.stderr, content...)
		case typeEndRequest:
			return resp, nil
		default:
			return nil, fmt.Errorf("fastcgi: unexpected record type %d", reqType)
		}
	}
}

func doFastCGI(sockPath string, pairs [][2]string, body io.Reader) (*fcgiResponse, error) {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("fastcgi: dial %s: %w", sockPath, err)
	}
	defer conn.Close()

	const requestID = 1
	if err := writeBeginRequest(conn, requestID); err != nil {
		return nil, err
	}
	if err := writeParams(conn, requestID, pairs); err != nil {
		return nil, err
	}
	if err := writeStdin(conn, requestID, body); err != nil {
		return nil, err
	}
	return readResponse(conn)
}
