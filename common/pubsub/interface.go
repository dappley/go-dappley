package pubsub

type Subscriber interface {
	GetSubscribedTopics() []string
	GetTopicHandler(topic string) TopicHandler
}
