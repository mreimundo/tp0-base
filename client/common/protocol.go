package common

import (
	"encoding/binary"
	"net"
	"strconv"
	"time"
)

const (
	StatusOK    = byte(0x00)
	StatusError = byte(0x01)
    MsgBatch 	= byte(0x01)
    MsgDone  	= byte(0x02)
    MsgQuery 	= byte(0x03)
)

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

// encodeBet serializes a single bet using mixed TLV encoding:
// [1B agency][4B doc][4B birthdate YYYYMMDD][2B number][1B fname_len][fname][1B lname_len][lname]
func encodeBet(agencyID string, bet Bet) ([]byte, error) {
	agency, err := strconv.ParseUint(agencyID, 10, 8)
	if err != nil {
		return nil, err
	}
	doc, err := strconv.ParseUint(bet.Document, 10, 32)
	if err != nil {
		return nil, err
	}
	number, err := strconv.ParseUint(bet.Number, 10, 16)
	if err != nil {
		return nil, err
	}
	t, err := time.Parse("2006-01-02", bet.Birthdate)
	if err != nil {
		return nil, err
	}
	birthdate := uint32(t.Year())*10000 + uint32(t.Month())*100 + uint32(t.Day())

	fname := []byte(bet.FirstName)
	lname := []byte(bet.LastName)

	// fixed OH: 1+4+4+2+1+1 = 13B variable
	buf := make([]byte, 13+len(fname)+len(lname))
	buf[0] = byte(agency)
	binary.BigEndian.PutUint32(buf[1:5], uint32(doc))
	binary.BigEndian.PutUint32(buf[5:9], birthdate)
	binary.BigEndian.PutUint16(buf[9:11], uint16(number))
	buf[11] = byte(len(fname))
	copy(buf[12:12+len(fname)], fname)
	buf[12+len(fname)] = byte(len(lname))
	copy(buf[13+len(fname):], lname)
	return buf, nil
}


// RecvBatchAck reads the 1-byte status response from the server
func RecvBatchAck(conn net.Conn) (bool, error) {
	buf, err := RecvAll(conn, 1)
	if err != nil {
		return false, err
	}
	return buf[0] == StatusOK, nil
}

// SendBatch ahora incluye el tipo de mensaje como primer byte
func SendBatch(conn net.Conn, agencyID string, bets []Bet) error {
    countBuf := make([]byte, 2)
    binary.BigEndian.PutUint16(countBuf, uint16(len(bets)))
    payload := countBuf

    for _, bet := range bets {
        encoded, err := encodeBet(agencyID, bet)
        if err != nil {
            return err
        }
        payload = append(payload, encoded...)
    }

    header := make([]byte, 2)
    binary.BigEndian.PutUint16(header, uint16(len(payload)))

    frame := append([]byte{MsgBatch}, header...)
    frame  = append(frame, payload...)
    return SendAll(conn, frame)
}

// SendDone notifica al servidor que la agencia terminó de enviar apuestas
func SendDone(conn net.Conn, agencyID string) error {
    agency, _ := strconv.ParseUint(agencyID, 10, 8)
    return SendAll(conn, []byte{MsgDone, byte(agency)})
}

// SendQueryWinners consulta los ganadores de la agencia.
// Retorna (ready, []dni, error). Si ready=false, el sorteo no ocurrió aún.
func SendQueryWinners(conn net.Conn, agencyID string) (bool, []string, error) {
    agency, _ := strconv.ParseUint(agencyID, 10, 8)
    if err := SendAll(conn, []byte{MsgQuery, byte(agency)}); err != nil {
        return false, nil, err
    }

    status, err := RecvAll(conn, 1)
    if err != nil {
        return false, nil, err
    }
    if status[0] != 0x00 { // QUERY_NOT_READY
        return false, nil, nil
    }

    countBuf, err := RecvAll(conn, 2)
    if err != nil {
        return false, nil, err
    }
    count := binary.BigEndian.Uint16(countBuf)

    winners := make([]string, 0, count)
    for i := 0; i < int(count); i++ {
        docBuf, err := RecvAll(conn, 4)
        if err != nil {
            return false, nil, err
        }
        doc := binary.BigEndian.Uint32(docBuf)
        winners = append(winners, strconv.FormatUint(uint64(doc), 10))
    }
    return true, winners, nil
}