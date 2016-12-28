package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"time"

	"github.com/mtyurt/slack-bet/repo"
)

const availableCommands = "Available commands: save, start, end, list, info, whowins"

type betService struct {
	repo  repo.Repo
	conf  *Conf
	utils Utils
}

func betHandler(service *betService) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := service.parseRequestAndCheckToken(r)
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
			!strings.EqualFold("info", firstCommand) {
			writeResponseWithBadRequest(&w, availableCommands)
			return
		}
		var resp string
		if strings.EqualFold("start", firstCommand) {
			resp, err = service.startNewBet(user)
		} else if strings.EqualFold("list", firstCommand) {
			resp, err = service.listBets()
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
			resp, err = service.saveBet(user, number)
		} else if strings.EqualFold("end", firstCommand) {
			resp, err = service.endBet(user)
		} else if strings.EqualFold("info", firstCommand) {
			betID := -1
			if len(commands) > 1 {
				betID, err = strconv.Atoi(commands[1])
				if err != nil {
					writeResponseWithBadRequest(&w, "id is not a valid integer "+commands[1])
					return
				}
			}

			resp, err = service.getBetInfo(betID)
		} else if strings.EqualFold("whowins", firstCommand) {
			if len(commands) > 1 {
				referenceNumber, err := strconv.Atoi(commands[1])
				if err != nil {
					writeResponseWithBadRequest(&w, "reference number is not a valid integer "+commands[1])
					return
				}
				resp, err = service.calculateWhoWins(referenceNumber)
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
			resp, err = service.saveBet(user, number)
		} else if strings.EqualFold("listabsent", firstCommand) {
			resp, err = service.listAbsentUsers()
			if err != nil {
			}
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
			resp, err = service.saveWinner(betID, winner)
		}

		if err != nil {
			writeResponseWithBadRequest(&w, err.Error())
			return
		}
		fmt.Fprint(w, resp)
	}
}

type ByBet []repo.BetDetail

func (a ByBet) Len() int           { return len(a) }
func (a ByBet) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByBet) Less(i, j int) bool { return a[i].Number < a[j].Number }

func (service *betService) saveWinner(betID int, winner int) (string, error) {
	exists, err := service.repo.BetIDExists(betID)
	if err != nil || !exists {
		return "", errors.New("No such bet exists.")
	}

	err = service.repo.SetBetWinner(betID, winner)
	if err != nil {
		return "", err
	}
	return "winner " + strconv.Itoa(winner) + "for bet " + strconv.Itoa(betID) + " is saved successfully", err
}

func (service *betService) listAbsentUsers() (string, error) {
	openBetID, err := service.repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == -1 {
		return "there is no active bet.", err
	}
	betDetails, err := service.repo.GetBetDetails(openBetID)
	if err != nil {
		return "", err
	}
	go service.doListAbsentUsers(betDetails)
	return "ok", nil
}

func (service *betService) doListAbsentUsers(betDetails []repo.BetDetail) {
	channelMembers, err := service.utils.GetChannelMembers()
	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < len(betDetails); i++ {
		detail := betDetails[i]
		for j, member := range channelMembers {
			if strings.EqualFold(member, detail.User) {
				channelMembers = append(channelMembers[:j], channelMembers[j+1:]...)
				break
			}
		}
	}

	service.utils.SendCallback("Users who have not placed a bet yet: " + strings.Join(channelMembers, ", "))
}

func (service *betService) calculateWhoWins(reference int) (string, error) {
	betID, err := service.repo.GetLastBetID()
	if err != nil {
		return "", err
	}
	if betID == -1 {
		return "No bet exists", nil
	}
	openBetID, err := service.repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == betID {
		return "you cannot query who wins for an active bet! I'm telling mom", nil
	}
	details, err := service.repo.GetBetDetails(betID)
	if err != nil {
		return "", err
	}
	totalUser := len(details)
	details = service.getWinners(details, reference)
	summary := "bet " + strconv.Itoa(betID) + ", " + strconv.Itoa(totalUser) + " people joined, hypothetical " + strconv.Itoa(totalUser/2) + " winners for score " + strconv.Itoa(reference) + ": "
	responseStr := summary + "\n"
	for _, detail := range details {
		responseStr += "\t" + detail.User + "\t" + strconv.Itoa(detail.Number) + "\n"
	}
	return responseStr, nil
}

func (service *betService) getWinners(details []repo.BetDetail, score int) []repo.BetDetail {
	userBetMap := make(map[string]int)
	totalUser := len(details)

	for i := 0; i < totalUser; i++ {
		detail := details[i]
		userBetMap[detail.User] = detail.Number
		n := detail.Number - score
		if n < 0 {
			n = -n
		}
		details[i].Number = n
	}
	sort.Sort(ByBet(details))
	winners := details[0 : totalUser/2]

	for i := 0; i < len(winners); i++ {
		winners[i].Number = userBetMap[winners[i].User]
	}
	return winners
}

func (service *betService) getBetInfo(id int) (string, error) {
	var err error
	betID := id
	if betID == -1 {
		betID, err = service.repo.GetLastBetID()
		if err != nil {
			return "", err
		}
		if betID == -1 {
			return "No bet exists", nil
		}
	}
	exists, err := service.repo.BetIDExists(betID)
	if err != nil || !exists {
		return "", errors.New("No such bet exists.")
	}
	summary, err := service.betSummary(betID)
	if err != nil {
		return "", err
	}
	openBetID, err := service.repo.GetIDOfOpenBet()
	if err != nil {
		return "", nil
	}
	if openBetID == betID {
		return summary, nil
	}
	details, err := service.repo.GetBetDetails(betID)
	if err != nil {
		return "", err
	}
	sort.Sort(ByBet(details))
	winnerScore, err := service.repo.GetWinnerScore(betID)
	winners := make(map[string]int)
	if winnerScore != -1 {
		winnerUsers := make([]repo.BetDetail, len(details))
		copy(winnerUsers, details)
		winnerUsers = service.getWinners(winnerUsers, winnerScore)
		for _, detail := range winnerUsers {
			winners[detail.User] = detail.Number
		}
	}
	responseStr := summary + "\n\n"
	for i, detail := range details {
		userSummary := strconv.Itoa(i+1) + ".\t" + detail.User + "\t" + strconv.Itoa(detail.Number)
		if _, ok := winners[detail.User]; ok {
			userSummary = "*" + userSummary + " (WINNER!)*"
		}
		responseStr += userSummary + "\n"
	}
	return responseStr, nil
}

func (service *betService) endBet(user string) (string, error) {
	if !service.isUserAuthorized(user) {
		return "", errors.New("You are not authorized to end a bet.")
	}
	openBetID, err := service.repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == -1 {
		return "", errors.New("There is no active bet right now.")
	}
	date := time.Now().Format(TimeFormat)
	err = service.repo.SetBetAsEnded(openBetID, date)
	if err != nil {
		return "", err
	}
	go service.sendBetEndedCallback(openBetID)
	return "ended bet[" + strconv.Itoa(openBetID) + "] successfully", nil
}

func (service *betService) isUserAuthorized(user string) bool {
	authorizedUsers := service.utils.GetAuthorizedUsers()
	for _, n := range authorizedUsers {
		if strings.EqualFold(n, user) {
			return true
		}
	}
	return false
}

func (service *betService) sendBetEndedCallback(betID int) {
	betInfo, err := service.getBetInfo(betID)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	service.utils.SendCallback(betInfo)
}

func (service *betService) saveBet(user string, number int) (string, error) {
	openBetID, err := service.repo.GetIDOfOpenBet()
	if openBetID == -1 {
		return "", errors.New("There is no active bet right now.")
	}

	details, err := service.repo.GetBetDetails(openBetID)
	if err != nil {
		return "", err
	}
	details = appendBetToList(details, user, number)
	err = service.repo.SetBetDetail(openBetID, details)
	if err != nil {
		return "", err
	}
	go service.utils.SendCallback(user + " has placed a bet. Have you?")
	return "saved successfully", nil
}

func appendBetToList(list []repo.BetDetail, user string, number int) []repo.BetDetail {
	found := false
	newList := make([]repo.BetDetail, len(list))

	for i, elem := range list {
		newElem := repo.BetDetail{User: elem.User, Number: elem.Number}
		if elem.User == user {
			newElem.Number = number
			found = true
		}
		newList[i] = newElem
	}
	if !found {
		newList = append(newList, repo.BetDetail{User: user, Number: number})
	}
	return newList
}

func (service *betService) startNewBet(user string) (string, error) {
	if !service.isUserAuthorized(user) {
		return "", errors.New("You are not authorized to start a bet.")
	}
	openBetID, err := service.repo.GetIDOfOpenBet()
	if err != nil {
		return "openbetid", err
	}
	if openBetID != -1 {
		return "", errors.New("There is a bet in progress, please finish it first.")
	}
	lastBetID, err := service.repo.GetLastBetID()
	if err != nil {
		return "", err
	}
	if lastBetID == -1 {
		lastBetID = 0
	}
	newID := lastBetID + 1
	err = service.repo.AddNewBet(newID, time.Now().Format(TimeFormat))
	if err != nil {
		return "", err
	}

	go service.utils.SendCallback("A new bet has started!")
	return "started bet[" + strconv.Itoa(newID) + "] successfully", nil
}

func (service *betService) listBets() (string, error) {
	lastID, err := service.repo.GetLastBetID()
	if err != nil {
		return "", nil
	}
	if lastID < 1 {
		return "empty", nil
	}
	responseStr := ""
	i := lastID - 5
	if i < 1 {
		i = 1
	}
	for ; i <= lastID; i++ {
		summary, err := service.betSummary(i)
		if err == nil {
			responseStr += summary + "\n"
		} else {
			return "", err
		}
	}
	return responseStr, nil
}

func (service *betService) betSummary(betID int) (string, error) {
	summary, err := service.repo.GetBetSummary(betID)
	if err != nil {
		return "", err
	}
	str := strconv.Itoa(summary.ID) + "\tstart: " + summary.StartDate
	if summary.Status == "open" {
		str += "\t(still open)"
	} else if summary.EndDate != "" {
		str += "\tend: " + summary.EndDate
	}
	if summary.WinnerNumber != -1 {
		str += "\twinner score: " + strconv.Itoa(summary.WinnerNumber)
	}
	return str, nil
}

func writeResponseWithBadRequest(w *http.ResponseWriter, text string) {
	(*w).WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(*w, text)
}

func (service *betService) parseRequestAndCheckToken(r *http.Request) error {
	r.ParseForm()

	if r.FormValue("token") != service.conf.SlashCommandToken {
		return errors.New("Token invalid, contact an admin")
	}
	return nil
}

func main() {
	utils := &Utility{ConfFileName: "conf.json"}
	conf, err := utils.GetConf()
	if err != nil {
		fmt.Println("conf cannot be read", err)
		return
	}
	service := &betService{repo: &repo.RedisRepo{Url: conf.RedisUrl}, conf: conf, utils: utils}
	http.HandleFunc("/bet", betHandler(service))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello mate.")
	})
	http.ListenAndServe(":"+conf.Port, nil)
}
