package slackbet

import "net/http"

const TimeFormat = "01-02-2006"

type BetService interface {
	ParseRequestAndCheckToken(*http.Request) error
	StartNewBet(string) (string, error)
	EndBet(string) (string, error)
	SaveBet(string, int) (string, error)
	ListBets() (string, error)
	GetBetInfo(int) (string, error)
	CalculateWhoWins(int) (string, error)
	SaveWinner(int, int) (string, error)
	GetLastEndedBetInfo() (string, error)
	ListAbsentUsers() (string, error)
}
type SlackService interface {
	GetChannelMembers() ([]string, error)
	SendCallback(string)
}
type Conf struct {
	Admins            []string `json:admins`
	PostToken         string   `json:postToken`
	Channel           string   `json:channel`
	ChannelID         string   `json:channelId`
	SlashCommandToken string   `json:slashCommandToken`
	RedisUrl          string   `json:redisUrl`
	Port              string   `json:port`
}
