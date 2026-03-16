package common

import (
	"net"
	"os"
    "os/signal"
    "syscall"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	Bet 		  Bet
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

// StartClientLoop Send messages to the client until some time threshold is met
// ej5 update: Sends the bet to the server and waits for confirmation
func (c *Client) StartClientLoop() {
	// seteo un channel para escuchar SIGTERM y poder interrumpir el loop de envío de mensajes
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	// ej4: agrego para que escuche a SIGTERM sin importar si el main loop está bloqueado por ReadString por ej.
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

	// ej5 update: quito loop por cantidad de mensajes y envío un solo mensaje con bet
	if err := c.createClientSocket(); err != nil {
		return
	}

	if err := SendBet(c.conn, c.config.ID, c.config.Bet); err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		c.conn.Close()
		c.conn = nil
		return
	}

	confirmation, err := RecvConfirmation(c.conn)
	c.conn.Close()
	log.Infof("action: close_connection | result: success | client_id: %v", c.config.ID)
	c.conn = nil

	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | client_id: %v | error: %v",
			c.config.ID, err)
		return
	}

	if confirmation == "OK" {
		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v",
			c.config.Bet.Document, c.config.Bet.Number)
	}


	log.Infof("action: loop_finished | result: success | client_id: %v", c.config.ID)
}
