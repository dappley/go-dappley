package core

type Event struct {
	topic string
	data  string
}

func NewEvent(topic, data string) *Event {
	return &Event{topic, data}
}

func (e *Event) GetTopic() string { return e.topic }
func (e *Event) GetData() string  { return e.data }
