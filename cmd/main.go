package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mtyurt/slack"
	"github.com/mtyurt/slackbet"
	"github.com/mtyurt/slackbet/bet"
	"github.com/mtyurt/slackbet/repo"
	oldslack "github.com/mtyurt/slackbet/slack"
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
		if len(commands) != 2 {
			return "", errors.New("save command format: save <number>")
		}
		number, err := strconv.Atoi(commands[1])
		if err != nil {
			return "", errors.New("number is not a valid integer " + commands[1])
		}
		return service.SaveBet(user, number)
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
		return service.SaveBet(user, number)
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
		var handler oldslack.SlackCommandHandler
		if strings.EqualFold("start", firstCommand) {
			handler = startHandler(service)
		} else if strings.EqualFold("list", firstCommand) {
			handler = listHandler(service)
		} else if strings.EqualFold("save", firstCommand) {
			handler = saveBetHandler(service)
		} else if strings.EqualFold("end", firstCommand) {
			handler = endBetHandler(service)
		} else if strings.EqualFold("info", firstCommand) {
			handler = betInfoHandler(service)
		} else if strings.EqualFold("whowins", firstCommand) {
			handler = whoWinsHandler(service)
		} else if strings.EqualFold("savefor", firstCommand) {
			handler = saveForHandler(service)
		} else if strings.EqualFold("listabsent", firstCommand) {
			handler = listAbentUsersHandler(service)
		} else if strings.EqualFold("savewinner", firstCommand) {
			handler = saveWinnerHandler(service)
		} else if strings.EqualFold("last", firstCommand) {
			handler = lastInfoHandler(service)
		}
		resp, err = handler(user, commands)

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

//func populateMux(mux *oldslack.SlackMux) {
//	if strings.EqualFold("start", firstCommand) {
//		handler = startHandler(service)
//	} else if strings.EqualFold("list", firstCommand) {
//		handler = listHandler(service)
//	} else if strings.EqualFold("save", firstCommand) {
//		handler = saveBetHandler(service)
//	} else if strings.EqualFold("end", firstCommand) {
//		handler = endBetHandler(service)
//	} else if strings.EqualFold("info", firstCommand) {
//		handler = betInfoHandler(service)
//	} else if strings.EqualFold("whowins", firstCommand) {
//		handler = whoWinsHandler(service)
//	} else if strings.EqualFold("savefor", firstCommand) {
//		handler = saveForHandler(service)
//	} else if strings.EqualFold("listabsent", firstCommand) {
//		handler = listAbentUsersHandler(service)
//	} else if strings.EqualFold("savewinner", firstCommand) {
//		handler = saveWinnerHandler(service)
//	} else if strings.EqualFold("last", firstCommand) {
//		handler = lastInfoHandler(service)
//	}
//
//}
func main() {
	conf, err := parseConf("conf.json")
	if err != nil {
		fmt.Println("conf cannot be read", err)
		return
	}
	slackService := &slack.SlackService{PostToken: conf.PostToken}
	service := &bet.BetService{Repo: &repo.RedisRepo{Url: conf.RedisUrl}, Conf: conf, SlackService: slackService}
	mux := &oldslack.SlackMux{}
	populateMux()
	http.HandleFunc("/bet", betHandler(service))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello mate.")
	})
	http.ListenAndServe(":"+conf.Port, nil)
}
