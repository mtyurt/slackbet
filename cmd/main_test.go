package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mtyurt/slack"
	"github.com/mtyurt/slackbet"
	"github.com/mtyurt/slackbet/bet"
	"github.com/mtyurt/slackbet/repo"
)

const slacktoken = "slacktoken"

func TestComplexBet(t *testing.T) {
	service := mockService()
	mockService := &MockService{}
	service.SlackService = mockService
	cli, err := openRedis()
	if err != nil {
		t.Fatal(err)
	}
	cli.Cmd("FLUSHALL")
	mux.Token = slacktoken
	populateMux(mux, service)
	ts := httptest.NewServer(http.HandlerFunc(mux.SlackHandler()))
	defer ts.Close()

	params := make(url.Values)
	params.Add("token", slacktoken)
	params.Add("user_name", "sezgin")
	params.Add("text", "start")
	if resp := betWithParams(params, service, mux); resp != "started bet[1] successfully" {
		t.Fatal(resp)
	}
	params.Set("user_name", "omer")
	params.Set("text", "save 100")
	if resp := betWithParams(params, service, mux); resp != "saved successfully" {
		t.Fatal(resp)
	}
	params.Set("user_name", "tarik")
	params.Set("text", "save 250")
	if resp := betWithParams(params, service, mux); resp != "saved successfully" {
		t.Fatal(resp)
	}
	params.Set("user_name", "tarik")
	params.Set("text", "save 75")
	if resp := betWithParams(params, service, mux); resp != "saved successfully" {
		t.Fatal(resp)
	}
	params.Set("text", "info 1")
	if resp := betWithParams(params, service, mux); strings.Contains(resp, "75") || strings.Contains(resp, "100") || !strings.Contains(resp, "open") {
		t.Fatal("response contains confidential info", resp)
	}
	params.Set("text", "end")
	params.Set("user_name", "sezgin")
	if resp := betWithParams(params, service, mux); resp != "ended bet[1] successfully" {
		t.Fatal(resp)
	}
	params.Set("text", "info 1")
	resp := betWithParams(params, service, mux)
	if strings.Contains(resp, "250") || !strings.Contains(resp, "100") || !strings.Contains(resp, "omer") || !strings.Contains(resp, "tarik") || !strings.Contains(resp, "end") {
		t.Fatal("response does not contain necessary info", resp)
	}
	body := mockService.sentCallback
	if strings.Contains(body, "250") || !strings.Contains(body, "100") || !strings.Contains(body, "omer") || !strings.Contains(body, "tarik") || !strings.Contains(body, "end") {
		t.Log(body)
		t.Fatal("body is wrong")
	}
}

func TestExampleConf(t *testing.T) {

	conf, err := parseConf("../conf.example.json")
	if err != nil || conf == nil {
		t.Fatal("error in initialization, err:", err, "conf:", conf)
	}
	if !reflect.DeepEqual(conf.Admins, []string{"tarik"}) {
		t.Fatal("admins is wrong:", conf.Admins)
	}
	if conf.PostToken != "chat-bot-token" {
		t.Error("token is wrong:", conf.PostToken)
	}
	if conf.Channel != "#general" {
		t.Fatal("channel is wrong:", conf.Channel)
	}
	if conf.ChannelID != "C9NMN9WVP" {
		t.Fatal("channel id is wrong:", conf.ChannelID)
	}
	if conf.SlashCommandToken != "8sLyRlhvsFwnZNOT1bpOxuocv1NnvZ1u" {
		t.Fatal("slash command token is wrong:", conf.SlashCommandToken)
	}
	if conf.RedisUrl != "http://localhost:6379" {
		t.Fatal("redis url is wrong:", conf.RedisUrl)
	}
	if conf.Port != "37564" {
		t.Fatal("port is wrong:", conf.Port)
	}
}

func betWithParams(params url.Values, service slackbet.BetService, mux *slack.SlackMux) string {
	recorder := httptest.NewRecorder()
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/bet"},
		Form:   params,
	}
	mux.SlackHandler()(recorder, req)
	return recorder.Body.String()
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
func mockService() *bet.BetService {
	c := &slackbet.Conf{SlashCommandToken: slacktoken, Admins: []string{"sezgin", "abdurrahim"}}
	mockService := bet.BetService{Conf: c, Repo: &repo.RedisRepo{Url: "localhost:37564"}, SlackService: &MockService{}}
	return &mockService
}
