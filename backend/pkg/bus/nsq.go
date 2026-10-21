package bus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nsqio/go-nsq"
)

const TopicCanonical = "iot.canonical.v1"
const TopicAlarm = "alarm.raised"

type Envelope struct {
	MsgID      string          `json:"msgId"`
	OccurredAt int64           `json:"occurredAt"`
	Producer   string          `json:"producer"`
	Payload    json.RawMessage `json:"payload"`
}

type Publisher struct {
	prod *nsq.Producer
}

func NewPublisher(nsqdTCP string) (*Publisher, error) {
	cfg := nsq.NewConfig()
	p, err := nsq.NewProducer(nsqdTCP, cfg)
	if err != nil {
		return nil, err
	}
	return &Publisher{prod: p}, nil
}

func (p *Publisher) Publish(topic string, producer string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	env := Envelope{
		MsgID: fmt.Sprintf("%d", time.Now().UnixNano()), OccurredAt: time.Now().UnixMilli(),
		Producer: producer, Payload: b,
	}
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return p.prod.Publish(topic, body)
}

func (p *Publisher) Stop() { p.prod.Stop() }

type HandlerFunc func(env Envelope) error

func Subscribe(nsqlookupdHTTP, topic, channel string, fn HandlerFunc) (*nsq.Consumer, error) {
	cfg := nsq.NewConfig()
	cfg.MaxInFlight = 32
	c, err := nsq.NewConsumer(topic, channel, cfg)
	if err != nil {
		return nil, err
	}
	c.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		var env Envelope
		if err := json.Unmarshal(msg.Body, &env); err != nil {
			return nil // drop bad
		}
		return fn(env)
	}))
	if err := c.ConnectToNSQLookupd(nsqlookupdHTTP); err != nil {
		return nil, err
	}
	return c, nil
}
