package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"strconv"
	"strings"
)

func toHex(sb *strings.Builder, b []byte) *strings.Builder {
	he := hex.NewEncoder(sb)
	for i := range b {
		_, _ = he.Write(b[i : i+1])
		if i == len(b)-1 {
			break
		}
		if i&63 == 63 {
			sb.WriteByte('\n')
		} else {
			sb.WriteByte(' ')
		}
	}
	return sb
}

func splitBySpace(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, " \t\n"); i >= 0 {
		// We have a full hex token.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated token. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func main() {
	// use with socat, e.g. `socat exec:hexio,fdin=3 tcp4-connect:localhost:11264`
	userIn := os.Stdin
	socatOut := os.Stdout
	userOut := os.Stderr
	socatIn := os.NewFile(3, "socat-in")

	go func() {
		lineScanner := bufio.NewScanner(userIn)
		for lineScanner.Scan() {
			line := lineScanner.Text()
			lineReader := strings.NewReader(line)

			outBytes := [65536]byte{0}
			outSlice := outBytes[0:0:65536]

			scanner := bufio.NewScanner(lineReader)
			scanner.Split(splitBySpace)
			for scanner.Scan() {
				hexToken := scanner.Text()
				if hexToken == "" {
					continue
				}

				c, err := strconv.ParseUint(hexToken, 16, 8)
				if err != nil {
					panic(err)
				}

				outSlice = append(outSlice, byte(c))
			}

			if len(outSlice) == 0 {
				continue
			}

			_, err := socatOut.Write(outSlice)
			if err != nil {
				panic(err)
			}

			// echo to userOut:
			sb := strings.Builder{}
			sb.WriteString("\033[1;35mOUT: ")
			toHex(&sb, outSlice)
			sb.WriteString("\033[0m\n")
			_, err = userOut.WriteString(sb.String())
			if err != nil {
				panic(err)
			}
		}
	}()

	// read from socatIn and echo as hex to userOut:
	for {
		inpBytes := [65536]byte{}

		// read from socatIn:
		n, err := socatIn.Read(inpBytes[:])
		if err == io.EOF {
			break
		}
		inpSlice := inpBytes[0:n]

		// echo to userOut:
		sb := strings.Builder{}
		sb.WriteString("\033[1;36mIN:  ")
		toHex(&sb, inpSlice)
		sb.WriteString("\033[0m\n")
		_, err = userOut.WriteString(sb.String())
		if err != nil {
			panic(err)
		}
	}
}
