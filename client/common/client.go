package common

import (
	"bufio"
	"net"
	"os"
    "os/signal"
	"strings"
    "syscall"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID             string
	ServerAddress  string
	MaxBatchAmount int
	DataFilePath   string
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	conn   net.Conn
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	return &Client{config: config}
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
    if err != nil {
        log.Criticalf(
            "action: connect | result: fail | client_id: %v | error: %v",
            c.config.ID, err,
        )
        return err
    }
    c.conn = conn
    return nil
}

// sendBatch sends a batch and waits for ACK, logs each bet on success
func (c *Client) sendBatch(batch []Bet) error {
	if err := SendBatch(c.conn, c.config.ID, batch); err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		c.conn.Close()
		c.conn = nil
		return err
	}
	ok, err := RecvBatchAck(c.conn)
	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		c.conn.Close()
		c.conn = nil
		return err
	}
	if ok {
		for _, bet := range batch {
			log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
				bet.Document, bet.Number)
		}
	}
	return nil
}

// ej6 update: createClientSocket now returns an error instead of exiting the program, so the caller can decide how to handle it (e.g. retry, log and exit, etc.)
// StartClientLoop reads bets from CSV and sends them in batches to the server
func (c *Client) StartClientLoop() {
	// seteo un channel para escuchar SIGTERM y poder interrumpir el loop de envío de mensajes
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	// ej4: agrego para que escuche a SIGTERM sin importar si el main loop está bloqueado
	go func() {
		<-sigs
		log.Infof("action: receive_sigterm | result: success | client_id: %v", c.config.ID)
		if c.conn != nil {
			c.conn.Close()
			log.Infof("action: close_connection | result: success | client_id: %v", c.config.ID)
		}
		log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
		os.Exit(0)
	}()

	if err := c.createClientSocket(); err != nil {
		return
	}

	file, err := os.Open(c.config.DataFilePath)
	if err != nil {
		log.Errorf("action: open_file | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		c.conn.Close()
		c.conn = nil
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	batch := make([]Bet, 0, c.config.MaxBatchAmount)

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ",")
		if len(fields) != 5 {
			continue
		}
		batch = append(batch, Bet{
			FirstName: fields[0],
			LastName:  fields[1],
			Document:  fields[2],
			Birthdate: fields[3],
			Number:    fields[4],
		})

		if len(batch) >= c.config.MaxBatchAmount {
			if err := c.sendBatch(batch); err != nil {
				return
			}
			batch = batch[:0]
		}
	}

	// enviar el último batch si quedaron apuestas
	if len(batch) > 0 {
		if err := c.sendBatch(batch); err != nil {
			return
		}
	}

	c.conn.Close()
	log.Infof("action: close_connection | result: success | client_id: %v", c.config.ID)
	c.conn = nil
	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
