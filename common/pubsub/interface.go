package pubsub

type Subscriber interface {
	GetSubscribedTopics() []string
	GetCommandHandler(commandName string) CommandHandler
}
