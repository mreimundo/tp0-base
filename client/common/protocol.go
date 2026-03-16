package common

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

const separator = "|" // it could be any char but this one is not likely to be expected

// SendAll sends all bytes avoiding short-write
func SendAll(conn net.Conn, data []byte) error {
	total := 0
	for total < len(data) {
		n, err := conn.Write(data[total:])
		if err != nil {
			return err
		}
		total += n
	}
	return nil
}

// RecvAll reads exactly n bytes avoiding short-read
func RecvAll(conn net.Conn, n int) ([]byte, error) {
	buf := make([]byte, n)
	total := 0
	for total < n {
		read, err := conn.Read(buf[total:])
		if err != nil {
			return nil, err
		}
		total += read
	}
	return buf, nil
}

// SendBet serializes and sends a bet using length-prefix framing:
// [ 2-byte big-endian length ][ agency|nombre|apellido|documento|nacimiento|numero ]
func SendBet(conn net.Conn, agencyID string, bet Bet) error {
	payload := fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s",
		agencyID, separator,
		bet.FirstName, separator,
		bet.LastName, separator,
		bet.Document, separator,
		bet.Birthdate, separator,
		bet.Number,
	)
	data := []byte(payload)
	header := make([]byte, 2)
	binary.BigEndian.PutUint16(header, uint16(len(data)))
	if err := SendAll(conn, header); err != nil {
		return err
	}
	return SendAll(conn, data)
}

// RecvConfirmation reads the server's response (length-prefix framing)
func RecvConfirmation(conn net.Conn) (string, error) {
	header, err := RecvAll(conn, 2)
	if err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint16(header)
	payload, err := RecvAll(conn, int(length))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(payload)), nil
}