package slackbet

import "net/http"

const TimeFormat = "02-01-2006"

var Months = [...]string{"january", "february", "march", "april", "may", "june", "july", "august", "september", "october", "november", "december"}

type BetService interface {
	ParseRequestAndCheckToken(*http.Request) error
	StartNewBet(string) (string, error)
	EndBet(string) (string, error)
	SaveBet(string, int, string) (string, error)
	ListBets() (string, error)
	GetBetInfo(int) (string, error)
	GetBetInfoForMonth(int) (string, error)
	CalculateWhoWins(int) (string, error)
	SaveWinner(int, int) (string, error)
	GetLastEndedBetInfo() (string, error)
	ListAbsentUsers() (string, error)
	IsAuthorizedUser(string) bool
}
type SlackService interface {
	GetChannelMembers(string) ([]string, error)
	SendCallback(string, string)
}
type Conf struct {
	Admins            []string `json:admins`
	PostToken         string   `json:postToken`
	Channel           string   `json:channel`
	ChannelID         string   `json:channelId`
	SlashCommandToken string   `json:slashCommandToken`
	RedisUrl          string   `json:redisUrl`
	Port              string   `json:port`
	EncryptionKey     string   `json:encryptionKey`
}
