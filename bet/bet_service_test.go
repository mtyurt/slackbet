package bet

import (
	"testing"
	"time"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mtyurt/slackbet"
	"github.com/mtyurt/slackbet/repo"
)

const slacktoken = "slacktoken"

func TestStartBet(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")

	startResp, err := service.StartNewBet("omer")
	if err == nil || err.Error() != "You are not authorized to start a bet." {
		t.Log(startResp)
		t.Fatal("start should fail, returned error:", err)
	}

	startResp, err = service.StartNewBet("sezgin")
	if err != nil || startResp != "started bet[1] successfully" {
		t.Fatal("start failed", err, startResp)
	}
	startResp, err = service.StartNewBet("sezgin")
	if err == nil || err.Error() != "There is a bet in progress, please finish it first." {
		t.Log(startResp)
		t.Fatal("start second bet should fail, returned error:", err)
	}
	allResp := client.Cmd("HGETALL", 1)

	betMap, err := allResp.Map()
	if err != nil {
		t.Fatal("bet entry doesn't exist")
	}
	if betMap["startDate"] != time.Now().Format(slackbet.TimeFormat) {
		t.Fatal("start date is wrong", betMap["startDate"])
	}
	if betMap["details"] != "[]" {
		t.Fatal("details is wrong", betMap["details"])
	}
	if betMap["status"] != "open" {
		t.Fatal("status is wrong", betMap["status"])
	}
	if openBetID, err := client.Cmd("GET", "OpenBet").Int(); err != nil || openBetID != 1 {
		t.Fatal("open bet id is wrong", openBetID, err)
	}
	if lastID, err := client.Cmd("GET", "LastID").Int(); err != nil || lastID != 1 {
		t.Fatal("last id is wrong", lastID, err)
	}
}
func TestSaveBet(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")

	saveResp, err := service.SaveBet("user1", 100)
	if err == nil || err.Error() != "There is no active bet right now." || saveResp != "" {
		t.Fatal("save bet should fail, returned error: ", err)
	}

	_, err = service.StartNewBet("sezgin")
	if err != nil {
		t.Fatal("start bet failed", err)
	}

	saveResp, err = service.SaveBet("user1", 100)
	if err != nil {
		t.Fatal("save failed", err, saveResp)
	}

	allResp := client.Cmd("HGETALL", 1)
	betMap, err := allResp.Map()
	if betMap["details"] != "[{\"User\":\"user1\",\"Number\":100}]" {
		t.Fatal("detail is wrong", betMap["details"])
	}
	//test second bet from same user
	saveResp, err = service.SaveBet("user1", 250)
	if err != nil {
		t.Fatal("save failed", err, saveResp)
	}

	allResp = client.Cmd("HGETALL", 1)
	betMap, err = allResp.Map()
	if betMap["details"] != "[{\"User\":\"user1\",\"Number\":250}]" {
		t.Fatal("detail is wrong", betMap["details"])
	}
	//test second user betting
	saveResp, err = service.SaveBet("user2", 300)
	if err != nil {
		t.Fatal("save failed", err, saveResp)
	}

	allResp = client.Cmd("HGETALL", 1)
	betMap, err = allResp.Map()
	if betMap["details"] != "[{\"User\":\"user1\",\"Number\":250},{\"User\":\"user2\",\"Number\":300}]" {
		t.Fatal("detail is wrong", betMap["details"])
	}
	saveResp, err = service.SaveBet("user2", 200)
	if err != nil {
		t.Fatal("save failed", err, saveResp)
	}
	allResp = client.Cmd("HGETALL", 1)
	betMap, err = allResp.Map()
	if betMap["details"] != "[{\"User\":\"user1\",\"Number\":250},{\"User\":\"user2\",\"Number\":200}]" {
		t.Fatal("detail is wrong", betMap["details"])
	}
	//set bet as closed
	client.Cmd("HSET", 1, "status", "closed")
	client.Cmd("DEL", "OpenBet")
	saveResp, err = service.SaveBet("user2", 300)
	if err == nil || err.Error() != "There is no active bet right now." {
		t.Fatal("save should fail with message", err, saveResp)
	}
}
func TestSaveBetForAnotherUser(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")

	saveResp, err := service.SaveBet("user1", 100)
	if err == nil || err.Error() != "There is no active bet right now." || saveResp != "" {
		t.Fatal("save bet should fail, returned error: ", err)
	}

	_, err = service.StartNewBet("sezgin")
	if err != nil {
		t.Fatal("start bet failed", err)
	}

	saveResp, err = service.SaveBet("user1", 100)
	if err != nil {
		t.Fatal("save failed", err, saveResp)
	}

	allResp := client.Cmd("HGETALL", 1)
	betMap, err := allResp.Map()
	if betMap["details"] != "[{\"User\":\"user1\",\"Number\":100}]" {
		t.Fatal("detail is wrong", betMap["details"])
	}
	//test second bet from same user
	saveResp, err = service.SaveBet("user1", 250)
}
func TestListBets(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")

	listResp, err := service.ListBets()
	if err != nil || listResp != "" {
		t.Fatal("list failed", err, listResp)
	}

	client.Cmd("HMSET", 1, "startDate", "01-02-2016", "endDate", "02-02-2016", "status", "closed")
	client.Cmd("HMSET", 2, "startDate", "01-02-2016", "endDate", "02-02-2016", "status", "closed")
	jsonStr := "[{\"User\":\"user1\",\"Number\":50},{\"User\":\"user2\",\"Number\":100}]"
	client.Cmd("HMSET", 3, "startDate", "01-02-2016", "status", "open", "details", jsonStr)
	client.Cmd("SET", "LastID", 3)
	expectedStr := "1\tstart: 01-02-2016\tend: 02-02-2016\n2\tstart: 01-02-2016\tend: 02-02-2016\n3\tstart: 01-02-2016\t(still open)\n"
	listResp, err = service.ListBets()
	if err != nil || listResp != expectedStr {
		t.Fatal("list failed", err, "expected\n", expectedStr, "but was\n", listResp)
	}
}
func TestEndBet(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")

	endResp, err := service.EndBet("sezgin")
	if err == nil || err.Error() != "There is no active bet right now." {
		t.Log(endResp)
		t.Fatal("end bet should fail", err)
	}
	_, err = service.EndBet("tarik")
	if err == nil || err.Error() != "You are not authorized to end a bet." {
		t.Fatal("end bet should fail", err)
	}
	client.Cmd("HMSET", 1, "startDate", "01-02-2016", "endDate", "02-02-2016", "status", "open")
	client.Cmd("SET", "OpenBet", 1)
	endResp, err = service.EndBet("sezgin")
	if err != nil && endResp != "ended bet[1] successfully" {
		t.Fatal("end bet failed", err, endResp)
	}
}
func TestGetBet(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")
	getResp, err := service.GetBetInfo(-1)
	if err != nil && getResp != "No bet exists" {
		t.Fatal("get bet failed", err, getResp)
	}
	jsonStr := "[{\"User\":\"user1\",\"Number\":100},{\"User\":\"user2\",\"Number\":75}]"
	client.Cmd("HMSET", 2, "startDate", "01-02-2016", "endDate", "02-02-2016", "status", "closed", "details", jsonStr)
	client.Cmd("HMSET", 3, "startDate", "01-03-2016", "status", "open", "details", "[]")
	client.Cmd("SET", "OpenBet", 3)
	client.Cmd("SET", "LastID", 3)

	getResp, err = service.GetBetInfo(2)
	if err != nil || getResp != "2\tstart: 01-02-2016\tend: 02-02-2016\n\n1.\tuser2\t75\n2.\tuser1\t100\n" {
		t.Fatal("get bet failed", err, "response:", getResp)
	}
	getResp, err = service.GetBetInfoForMonth(1)
	if err != nil || getResp != "2\tstart: 01-02-2016\tend: 02-02-2016\n\n1.\tuser2\t75\n2.\tuser1\t100\n" {
		t.Fatal("get bet failed", err, "response:", getResp)
	}
	getResp, err = service.GetBetInfo(3)
	if err != nil || getResp != "3\tstart: 01-03-2016\t(still open)" {
		t.Fatal("get bet failed", err, getResp)
	}
	getResp, err = service.GetBetInfoForMonth(2)
	if err != nil || getResp != "3\tstart: 01-03-2016\t(still open)" {
		t.Fatal("get bet failed", err, getResp)
	}
	getResp, err = service.GetBetInfo(4)
	if err == nil || err.Error() != "No such bet exists." {
		t.Fatal("bet should fail", err, getResp)
	}

	client.Cmd("SET", "LastID", 2)
	getResp, err = service.GetBetInfo(-1)
	if err != nil || getResp != "2\tstart: 01-02-2016\tend: 02-02-2016\n\n1.\tuser2\t75\n2.\tuser1\t100\n" {
		t.Fatal("get bet failed", err, getResp)
	}
}
func TestWhoWins(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")
	getResp, err := service.GetBetInfo(-1)
	if err != nil && getResp != "No bet exists" {
		t.Fatal("who wins failed", err, getResp)
	}
	jsonStr := "[{\"User\":\"user1\",\"Number\":100},{\"User\":\"user2\",\"Number\":75},{\"User\":\"user3\",\"Number\":175},{\"User\":\"user4\",\"Number\":275},{\"User\":\"user5\",\"Number\":120}]"
	client.Cmd("HMSET", 2, "startDate", "01-02-2016", "endDate", "02-02-2016", "status", "closed", "details", jsonStr)
	client.Cmd("HMSET", 3, "startDate", "01-02-2016", "status", "open", "details", "[]")
	client.Cmd("SET", "OpenBet", 3)

	getResp, err = service.CalculateWhoWins(100)
	if err != nil && getResp != "you cannot query who wins for an active bet! I'm telling mom" {
		t.Fatal("who wins failed", err, getResp)
	}
	client.Cmd("DEL", 3)
	client.Cmd("SET", "OpenBet", -1)
	client.Cmd("SET", "LastID", 2)

	getResp, err = service.CalculateWhoWins(130)
	if err != nil || getResp != "bet 2, 5 people joined, hypothetical 2 winners for score 130: \n\tuser5\t120\n\tuser1\t100\n" {
		t.Fatal("who wins failed", err, getResp)
	}
}
func TestListAbsentUsers(t *testing.T) {
	service := mockService()
	mockService := &MockService{}
	service.SlackService = mockService
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")
	jsonStr := "[{\"User\":\"user1\",\"Number\":100},{\"User\":\"user2\",\"Number\":75},{\"User\":\"user3\",\"Number\":175},{\"User\":\"user4\",\"Number\":275},{\"User\":\"user5\",\"Number\":120}]"
	client.Cmd("HMSET", 2, "startDate", "01-02-2016", "status", "open", "details", jsonStr)
	client.Cmd("SET", "OpenBet", 2)

	mockService.channelMembers = []string{"user1", "user2", "user3", "user4", "user5", "user6", "user7"}
	resp, err := service.ListAbsentUsers()
	if err != nil || resp != "ok" {
		t.Fatal("list absent users failed, err:", err, "response: ", resp)
	}
}
func TestSaveWinner(t *testing.T) {
	service := mockService()
	client, err := openRedis()
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}
	client.Cmd("FLUSHALL")
	jsonStr := "[{\"User\":\"user1\",\"Number\":100},{\"User\":\"user2\",\"Number\":75},{\"User\":\"user3\",\"Number\":500},{\"User\":\"user4\",\"Number\":200}]"
	client.Cmd("HMSET", 2, "startDate", "01-02-2016", "endDate", "02-02-2016", "status", "closed", "details", jsonStr)

	getResp, err := service.SaveWinner(2, 250)
	if err != nil {
		t.Fatal("save winner failed with error", err)
	}
	getResp, err = service.GetBetInfo(2)
	if err != nil || getResp != "2\tstart: 01-02-2016\tend: 02-02-2016\twinner score: 250\n\n1.\tuser2\t75\n*2.\tuser1\t100 (WINNER!)*\n*3.\tuser4\t200 (WINNER!)*\n4.\tuser3\t500\n" {
		t.Fatal("save winner failed", err, getResp)
	}
}

type MockService struct {
	channelMembers []string
	sentCallback   string
}

func openRedis() (*redis.Client, error) {
	client, err := redis.Dial("tcp", "localhost:37564")
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (service *MockService) GetChannelMembers(channelID string) ([]string, error) {
	return service.channelMembers, nil
}

func (service *MockService) SendCallback(text string, channel string) {
	service.sentCallback = text
}
func mockService() *BetService {
	c := &slackbet.Conf{SlashCommandToken: slacktoken, Admins: []string{"sezgin", "abdurrahim"}}
	mockService := BetService{Conf: c, Repo: &repo.RedisRepo{Url: "localhost:37564"}, SlackService: &MockService{}}
	return &mockService
}
