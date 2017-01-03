package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/mtyurt/slackbet"
)

type SlackService struct {
	Conf *slackbet.Conf
}

func (service *SlackService) SendCallback(text string) {
	conf := service.Conf
	uri := "https://slack.com/api/chat.postMessage?token=" + conf.PostToken + "&channel=" + url.QueryEscape(conf.Channel) + "&text=" + url.QueryEscape(text) + "&as_user=true"
	resp, err := http.Get(uri)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

type sluserinfo struct {
	Name    string `json:name`
	Deleted bool   `json:deleted`
	IsBot   bool   `json:is_bot`
}

type sluser struct {
	Ok   bool       `json:ok`
	User sluserinfo `json:user`
}

type slchannelinfo struct {
	Members []string `json:members`
}

type slchannel struct {
	Ok      bool          `json:ok`
	Channel slchannelinfo `json:channel`
}

func (service *SlackService) GetChannelMembers() ([]string, error) {
	conf := service.Conf
	resp, err := http.Get("https://slack.com/api/channels.info?token=" + conf.PostToken + "&channel=" + conf.ChannelID)

	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	var channelInfo slchannel
	err = json.Unmarshal(body, &channelInfo)
	if err != nil {
		return nil, err
	}
	memberIds := channelInfo.Channel.Members
	var userNames []string
	baseUserInfoReqUrl := "https://slack.com/api/users.info?token=" + conf.PostToken + "&user="
	for _, userId := range memberIds {
		resp, err = http.Get(baseUserInfoReqUrl + userId)
		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		var usrinfo sluser
		err = json.Unmarshal(body, &usrinfo)
		if err != nil {
			return nil, err
		}
		if usrinfo.User.Deleted || usrinfo.User.IsBot {
			continue
		}
		userNames = append(userNames, usrinfo.User.Name)
	}
	return userNames, nil
}
