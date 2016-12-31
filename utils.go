package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/mtyurt/slack-bet/repo"
)

const TimeFormat = "01-02-2006"

type Utility struct {
	conf         *Conf
	ConfFileName string
	Repo         repo.Repo
}
type Utils interface {
	GetAuthorizedUsers() []string
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

func (utils *Utility) SendCallback(text string) {
	conf, err := utils.GetConf()
	if err != nil {
		return
	}
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
func (util *Utility) GetAuthorizedUsers() []string {
	conf, err := util.GetConf()
	if err != nil {
		return []string{}
	}
	return conf.Admins
}
func (util *Utility) GetConf() (*Conf, error) {
	if util.conf != nil {
		return util.conf, nil
	}
	file, err := os.Open(util.ConfFileName)
	if err != nil {
		return nil, err
	}
	c := &Conf{}
	err = json.NewDecoder(file).Decode(c)
	if err != nil {
		return nil, err
	}
	util.conf = c
	return c, nil
}

type sluserinfo struct {
	Name    string `json:name`
	Deleted bool   `json:deleted`
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

func (util *Utility) GetChannelMembers() ([]string, error) {
	conf, err := util.GetConf()
	if err != nil {
		return nil, err
	}
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
		if usrinfo.User.Deleted {
			continue
		}
		userNames = append(userNames, usrinfo.User.Name)
	}
	return userNames, nil
}
