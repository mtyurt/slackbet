package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mtyurt/slackbet"
	"github.com/mtyurt/slackbet/bet"
	"github.com/mtyurt/slackbet/repo"
	"github.com/mtyurt/slackcommander"
)

const availableCommands = "Available commands: save, list, info, last, whowins "

func startHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, args []string) (string, error) {
		return service.StartNewBet(user)
	}
}
func listHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, args []string) (string, error) {
		return service.ListBets()
	}
}
func saveBetHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {
		if len(commands) < 2 {
			return "", errors.New("save command format: save <number> <extra info>")
		}
		number, err := strconv.Atoi(commands[1])
		if err != nil {
			return "", errors.New("number is not a valid integer " + commands[1])
		}
		extraInfo := ""
		if len(commands) > 2 {
			extraInfo = strings.Join(commands[2:], " ")
		}
		return service.SaveBet(user, number, extraInfo)
	}
}
func endBetHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, args []string) (string, error) {
		return service.EndBet(user)
	}
}

func betInfoHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {
		if len(commands) < 2 {
			return "", errors.New("usage: /bet info <month or id of bet>")
		}
		secondArg := commands[1]
		if isAllInteger(secondArg) {
			betID, err := strconv.Atoi(secondArg)
			if err != nil {
				return "", errors.New("id is not a valid integer " + commands[1])
			}
			return service.GetBetInfo(betID)
		} else {
			monthIndex := getMonthIndex(secondArg)
			if monthIndex == -1 {
				return "", errors.New(secondArg + " is not a valid month.")
			}
			return service.GetBetInfoForMonth(monthIndex)
		}
	}
}
func whoWinsHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {
		if len(commands) > 1 {
			referenceNumber, err := strconv.Atoi(commands[1])
			if err != nil {
				return "", errors.New("reference number is not a valid integer " + commands[1])
			}
			return service.CalculateWhoWins(referenceNumber)
		} else {
			return "", errors.New("usage: /bet whowins <number>")
		}
	}
}
func saveForHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {
		if !service.IsAuthorizedUser(user) || len(commands) != 3 {
			return "", errors.New("savefor is not a valid command.")
		}
		user = commands[1]
		number, err := strconv.Atoi(commands[2])
		if err != nil {
			return "", errors.New("number is not a valid integer " + commands[2])
		}
		return service.SaveBet(user, number, "")
	}
}
func listAbentUsersHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {
		return service.ListAbsentUsers()
	}
}
func saveWinnerHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {

		if !service.IsAuthorizedUser(user) || len(commands) != 3 {
			return "", errors.New("savefor is not a valid command.")
		}
		winner, err := strconv.Atoi(commands[2])
		if err != nil {
			return "", errors.New("winner number is not a valid integer " + commands[2])
		}
		betID, err := strconv.Atoi(commands[1])
		if err != nil {
			return "", errors.New("betID is not a valid integer " + commands[1])
		}
		return service.SaveWinner(betID, winner)
	}
}
func lastInfoHandler(service slackbet.BetService) func(string, []string) (string, error) {
	return func(user string, commands []string) (string, error) {
		return service.GetLastEndedBetInfo()
	}
}
func getMonthIndex(month string) int {
	for i, m := range slackbet.Months {
		if m == month {
			return i
		}
	}
	return -1
}

func isAllInteger(s string) bool {
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func writeResponseWithBadRequest(w *http.ResponseWriter, text string) {
	(*w).WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(*w, text)
}
func parseConf(confFileName string) (*slackbet.Conf, error) {
	file, err := os.Open(confFileName)
	if err != nil {
		return nil, err
	}
	c := &slackbet.Conf{}
	err = json.NewDecoder(file).Decode(c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func populateMux(mux *slackcommander.SlackMux, service slackbet.BetService) {
	mux.RegisterCommand("start", startHandler(service))
	mux.RegisterCommand("list", listHandler(service))
	mux.RegisterCommand("save", saveBetHandler(service))
	mux.RegisterCommand("end", endBetHandler(service))
	mux.RegisterCommand("info", betInfoHandler(service))
	mux.RegisterCommand("whowins", whoWinsHandler(service))
	mux.RegisterCommand("savefor", saveForHandler(service))
	mux.RegisterCommand("listabsent", listAbentUsersHandler(service))
	mux.RegisterCommand("savewinner", saveWinnerHandler(service))
	mux.RegisterCommand("last", lastInfoHandler(service))
}

var mux *slackcommander.SlackMux = &slackcommander.SlackMux{}

func main() {
	conf, err := parseConf("conf.json")
	if err != nil {
		fmt.Println("conf cannot be read", err)
		return
	}
	slackService := &slackcommander.SlackService{PostToken: conf.PostToken}
	service := &bet.BetService{Repo: &repo.RedisRepo{Url: conf.RedisUrl}, Conf: conf, SlackService: slackService}
	mux.Token = service.Conf.SlashCommandToken
	populateMux(mux, service)
	http.HandleFunc("/bet", mux.SlackHandler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello mate.")
	})
	http.ListenAndServe(":"+conf.Port, nil)
}
