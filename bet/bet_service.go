package bet

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtyurt/slackbet"
	"github.com/mtyurt/slackbet/repo"
)

type BetService struct {
	Repo         repo.Repo
	Conf         *slackbet.Conf
	SlackService slackbet.SlackService
}
type ByBet []repo.BetDetail

func (a ByBet) Len() int           { return len(a) }
func (a ByBet) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByBet) Less(i, j int) bool { return a[i].Number < a[j].Number }

func (service *BetService) SaveWinner(betID int, winner int) (string, error) {
	exists, err := service.Repo.BetIDExists(betID)
	if err != nil || !exists {
		return "", errors.New("No such bet exists.")
	}

	err = service.Repo.SetBetWinner(betID, winner)
	if err != nil {
		return "", err
	}
	return "winner " + strconv.Itoa(winner) + "for bet " + strconv.Itoa(betID) + " is saved successfully", err
}

func (service *BetService) ListAbsentUsers() (string, error) {
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == -1 {
		return "there is no active bet.", err
	}
	betDetails, err := service.Repo.GetBetDetails(openBetID)
	if err != nil {
		return "", err
	}
	go service.doListAbsentUsers(betDetails)
	return "ok", nil
}

func (service *BetService) doListAbsentUsers(betDetails []repo.BetDetail) {
	channelMembers, err := service.SlackService.GetChannelMembers(service.Conf.ChannelID)
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

	service.SlackService.SendCallback("Users who have not placed a bet yet: "+strings.Join(channelMembers, ", "), service.Conf.Channel)
}

func (service *BetService) CalculateWhoWins(reference int) (string, error) {
	betID, err := service.Repo.GetLastBetID()
	if err != nil {
		return "", err
	}
	if betID == -1 {
		return "No bet exists", nil
	}
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == betID {
		return "you cannot query who wins for an active bet! I'm telling mom", nil
	}
	details, err := service.Repo.GetBetDetails(betID)
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
func (service *BetService) getWinners(details []repo.BetDetail, score int) []repo.BetDetail {
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
func (service *BetService) GetLastEndedBetInfo() (string, error) {
	betID, err := service.Repo.GetLastBetID()
	if err != nil {
		return "", err
	}
	if betID == -1 {
		return "No bet exists", nil
	}
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "", nil
	}
	if betID == openBetID {
		betID = betID - 1
	}
	exists, err := service.Repo.BetIDExists(betID)
	if err != nil || !exists {
		return "", errors.New("No such bet exists.")
	}
	summary, err := service.Repo.GetBetSummary(betID)
	if err != nil {
		return "", err
	}
	return service.generateBetDetails(betID, summary.String())
}

func (service *BetService) GetBetInfo(id int) (string, error) {
	var err error
	betID := id
	if betID == -1 {
		betID, err = service.Repo.GetLastBetID()
		if err != nil {
			return "", err
		}
		if betID == -1 {
			return "No bet exists", nil
		}
	}
	exists, err := service.Repo.BetIDExists(betID)
	if err != nil || !exists {
		return "", errors.New("No such bet exists.")
	}
	summary, err := service.Repo.GetBetSummary(betID)
	if err != nil {
		return "", err
	}
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == betID {
		return summary.String(), nil
	}
	return service.generateBetDetails(betID, summary.String())
}
func (service *BetService) GetBetInfoForMonth(monthIndex int) (string, error) {
	summaries, err := service.getBetSummaryList(12)
	if err != nil {
		return "", err
	}
	reverse(summaries)
	var summary *repo.BetSummary
	for i := 0; i < len(summaries); i++ {
		s := summaries[i]
		date := s.EndDate
		if s.EndDate == "" {
			date = s.StartDate
		}
		month, err := strconv.Atoi(date[3:5])
		if err != nil {
			return "", err
		}
		if month == monthIndex+1 {
			summary = &s
			break
		}
	}
	if summary == nil {
		return "", errors.New("bet for month " + slackbet.Months[monthIndex] + " not found.")
	}
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == summary.ID {
		return summary.String(), nil
	}
	return service.generateBetDetails(summary.ID, summary.String())
}
func (service *BetService) generateBetDetails(betID int, summary string) (string, error) {
	details, err := service.Repo.GetBetDetails(betID)
	if err != nil {
		return "", err
	}
	sort.Sort(ByBet(details))
	winnerScore, err := service.Repo.GetWinnerScore(betID)
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

func (service *BetService) EndBet(user string) (string, error) {
	if !service.IsAuthorizedUser(user) {
		return "", errors.New("You are not authorized to end a bet.")
	}
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "", err
	}
	if openBetID == -1 {
		return "", errors.New("There is no active bet right now.")
	}
	date := time.Now().Format(slackbet.TimeFormat)
	err = service.Repo.SetBetAsEnded(openBetID, date)
	if err != nil {
		return "", err
	}
	go service.sendBetEndedCallback(openBetID)
	return "ended bet[" + strconv.Itoa(openBetID) + "] successfully", nil
}
func (service *BetService) IsAuthorizedUser(user string) bool {
	for _, n := range service.Conf.Admins {
		if strings.EqualFold(n, user) {
			return true
		}
	}
	return false
}

func (service *BetService) sendBetEndedCallback(betID int) {
	betInfo, err := service.GetBetInfo(betID)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	service.SlackService.SendCallback(betInfo, service.Conf.Channel)
}

func (service *BetService) SaveBet(user string, number int, extraInfo string) (string, error) {
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if openBetID == -1 {
		return "", errors.New("There is no active bet right now.")
	}

	details, err := service.Repo.GetBetDetails(openBetID)
	if err != nil {
		return "", err
	}
	details = appendBetToList(details, user, number)
	err = service.Repo.SetBetDetail(openBetID, details)
	if err != nil {
		return "", err
	}
	go service.SlackService.SendCallback(user+" has placed a bet. Have you?", service.Conf.Channel)
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

func (service *BetService) StartNewBet(user string) (string, error) {
	if !service.IsAuthorizedUser(user) {
		return "", errors.New("You are not authorized to start a bet.")
	}
	openBetID, err := service.Repo.GetIDOfOpenBet()
	if err != nil {
		return "openbetid", err
	}
	if openBetID != -1 {
		return "", errors.New("There is a bet in progress, please finish it first.")
	}
	lastBetID, err := service.Repo.GetLastBetID()
	if err != nil {
		return "", err
	}
	if lastBetID == -1 {
		lastBetID = 0
	}
	newID := lastBetID + 1
	err = service.Repo.AddNewBet(newID, time.Now().Format(slackbet.TimeFormat))
	if err != nil {
		return "", err
	}

	go service.SlackService.SendCallback("A new bet has started!", service.Conf.Channel)
	return "started bet[" + strconv.Itoa(newID) + "] successfully", nil
}

func (service *BetService) ListBets() (string, error) {
	summaries, err := service.getBetSummaryList(5)
	if err != nil {
		return "", err
	}
	response := ""
	for _, summary := range summaries {
		response += summary.String() + "\n"
	}
	return response, nil
}

func (service *BetService) getBetSummaryList(count int) ([]repo.BetSummary, error) {
	lastID, err := service.Repo.GetLastBetID()
	if err != nil {
		return nil, err
	}
	if lastID < 1 {
		return nil, nil
	}
	i := lastID - count
	length := count
	if i < 1 {
		i = 1
		length = lastID
	}

	list := make([]repo.BetSummary, length)
	for ; i <= lastID; i++ {
		summary, err := service.Repo.GetBetSummary(i)
		if err != nil {
			return nil, err
		} else {
			list[i-1] = *summary
		}
	}
	return list, nil
}
func reverse(ss []repo.BetSummary) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}
func (service *BetService) ParseRequestAndCheckToken(r *http.Request) error {
	r.ParseForm()

	if r.FormValue("token") != service.Conf.SlashCommandToken {
		return errors.New("Token invalid, contact an admin")
	}
	return nil
}
