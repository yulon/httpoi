package httpoi

import (
	"net"
	"fmt"
	"time"
	"errors"
)

type Handler func(*Respond, *Requested)

func Sever(laddr string, h Handler) error {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return errors.New("Failed to listen")
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return errors.New("Failed to accept")
		}
		go saw(conn, h)
	}
	return nil
}

func saw(c net.Conn, h Handler) {
	rawReq := make([]byte, 512)
	rawReqLen, err := c.Read(rawReq)
	if err == nil {
		reqLine := &RequestLine{}

		// Parse Request Line
		i, sp1, sp2 := 0, 0, 0
		prl:
		for ; i < rawReqLen; i++ {
			switch rawReq[i] {
				case ' ':
					if sp1 == 0 {
						sp1 = i
					}else{
						sp2 = i
					}

				case '\r':
					reqLine.HTTPVersion = string(rawReq[sp2+1:i])

				case '\n':
					break prl
			}
		}

		if reqLine.HTTPVersion == "HTTP/1.1" { // Is HTTP
			reqLine.Method = string(rawReq[:sp1])
			reqLine.URI = string(rawReq[sp1+1:sp2])
			fmt.Println(reqLine)

			// Parse Request Header Fields
			reqHeader := &RequestHeader{
				RequestLine: reqLine,
				Fields: map[string]string{},
			}

			prh:
			for keyStart, keyEnd, valStart := i + 1, 0, 0; i < rawReqLen; i++ {
				switch rawReq[i] {
					case ':':
						if keyEnd == 0 {
							keyEnd = i
							for y := 1; i+1 < rawReqLen; y++ {
								if rawReq[i+y] != ' ' && rawReq[i+y] != '\t' {
									i = i + y - 1
									valStart = i + 1
									continue prh
								}
							}
						}

					case '\r':
						if keyEnd - keyStart > 0 {
							for y := 1; i+1 < rawReqLen; y++ {
								if rawReq[i-y] != ' ' && rawReq[i-y] != '\t' {
									reqHeader.Fields[string(rawReq[keyStart:keyEnd])] = string(rawReq[valStart:i-y+1])
									continue prh
								}
							}
						}

					case '\n':
						keyStart = i + 1
						keyEnd = 0
				}
			}

			rqd := &Requested{
				RequestHeader: reqHeader,
			}

			rsd := &Respond{
				ResponseHeader: &ResponseHeader{
					StatusLine: &StatusLine{
						HTTPVersion: rqd.HTTPVersion,
					},
					Fields: map[string]string{
						"Date": time.Now().Format(time.RFC1123),
						"Server": "HTTPOI",
						"X-Powered-By": langVer,
					},
				},
				w: c,
				end: false,
			}

			h(rsd, rqd)
			rsd.writeEnd()
		}
	}
	c.Close()
}