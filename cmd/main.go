package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mtyurt/slack"
	"github.com/mtyurt/slackbet"
	"github.com/mtyurt/slackbet/bet"
	"github.com/mtyurt/slackbet/repo"
)

const availableCommands = "Available commands: save, list, info, last, whowins "

func betHandler(service slackbet.BetService) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := service.ParseRequestAndCheckToken(r)
		if err != nil {
			writeResponseWithBadRequest(&w, err.Error())
			return
		}
		user := r.FormValue("user_name")
		text := r.FormValue("text")
		commands := strings.Split(text, " ")
		if commands == nil || len(commands) < 1 {
			writeResponseWithBadRequest(&w, availableCommands)
			return
		}
		firstCommand := commands[0]
		if !strings.EqualFold("save", firstCommand) &&
			!strings.EqualFold("start", firstCommand) &&
			!strings.EqualFold("end", firstCommand) &&
			!strings.EqualFold("whowins", firstCommand) &&
			!strings.EqualFold("savefor", firstCommand) &&
			!strings.EqualFold("listabsent", firstCommand) &&
			!strings.EqualFold("savewinner", firstCommand) &&
			!strings.EqualFold("list", firstCommand) &&
			!strings.EqualFold("last", firstCommand) &&
			!strings.EqualFold("info", firstCommand) {
			writeResponseWithBadRequest(&w, availableCommands)
			return
		}
		var resp string
		if strings.EqualFold("start", firstCommand) {
			resp, err = service.StartNewBet(user)
		} else if strings.EqualFold("list", firstCommand) {
			resp, err = service.ListBets()
		} else if strings.EqualFold("save", firstCommand) {
			if len(commands) != 2 {
				writeResponseWithBadRequest(&w, "save command format: save <number>")
				return
			}
			number, err := strconv.Atoi(commands[1])
			if err != nil {
				writeResponseWithBadRequest(&w, "number is not a valid integer "+commands[1])
				return
			}
			resp, err = service.SaveBet(user, number)
		} else if strings.EqualFold("end", firstCommand) {
			resp, err = service.EndBet(user)
		} else if strings.EqualFold("info", firstCommand) {
			betID := -1
			if len(commands) < 2 {
				writeResponseWithBadRequest(&w, "usage: /bet info <month or id of bet>")
				return
			}
			secondArg := commands[1]
			if isAllInteger(secondArg) {
				betID, err = strconv.Atoi(secondArg)
				if err != nil {
					writeResponseWithBadRequest(&w, "id is not a valid integer "+commands[1])
					return
				}
				resp, err = service.GetBetInfo(betID)
			} else {
				monthIndex := getMonthIndex(secondArg)
				if monthIndex == -1 {
					writeResponseWithBadRequest(&w, secondArg+" is not a valid month.")
					return
				}
				resp, err = service.GetBetInfoForMonth(monthIndex)
			}
		} else if strings.EqualFold("whowins", firstCommand) {
			if len(commands) > 1 {
				referenceNumber, err := strconv.Atoi(commands[1])
				if err != nil {
					writeResponseWithBadRequest(&w, "reference number is not a valid integer "+commands[1])
					return
				}
				resp, err = service.CalculateWhoWins(referenceNumber)
			} else {
				writeResponseWithBadRequest(&w, "usage: /bet whowins <number>")
				return
			}
		} else if strings.EqualFold("savefor", firstCommand) && user == "tarik" {
			if len(commands) != 3 {
				writeResponseWithBadRequest(&w, availableCommands)
				return
			}
			user = commands[1]
			number, err := strconv.Atoi(commands[2])
			if err != nil {
				writeResponseWithBadRequest(&w, "number is not a valid integer "+commands[2])
				return
			}
			resp, err = service.SaveBet(user, number)
		} else if strings.EqualFold("listabsent", firstCommand) {
			resp, err = service.ListAbsentUsers()
		} else if strings.EqualFold("savewinner", firstCommand) && user == "tarik" {

			if len(commands) != 3 {
				writeResponseWithBadRequest(&w, availableCommands)
			}
			winner, err := strconv.Atoi(commands[2])
			if err != nil {
				writeResponseWithBadRequest(&w, "number is not a valid integer "+commands[2])
				return
			}

			betID, err := strconv.Atoi(commands[1])
			if err != nil {
				writeResponseWithBadRequest(&w, "number is not a valid integer "+commands[1])
				return
			}
			resp, err = service.SaveWinner(betID, winner)
		} else if strings.EqualFold("last", firstCommand) {
			resp, err = service.GetLastEndedBetInfo()
		}

		if err != nil {
			writeResponseWithBadRequest(&w, err.Error())
			return
		}
		fmt.Fprint(w, resp)
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
func main() {
	conf, err := parseConf("conf.json")
	if err != nil {
		fmt.Println("conf cannot be read", err)
		return
	}
	slackService := &slack.SlackService{PostToken: conf.PostToken}
	service := &bet.BetService{Repo: &repo.RedisRepo{Url: conf.RedisUrl}, Conf: conf, SlackService: slackService}
	http.HandleFunc("/bet", betHandler(service))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello mate.")
	})
	http.ListenAndServe(":"+conf.Port, nil)
}
