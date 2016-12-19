package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mtyurt/bet/repo"
)

type DB struct {
	*sql.DB
}

const TimeFormat = "01-02-2006"

type Utility struct {
}
type Utils interface {
	OpenRedis() (*redis.Client, error)
	PostHTTP(string, string) error
	GetAuthorizedUsers() []string
	GetChannelMembers() ([]string, error)
	GetRepo() repo.Repo
	GetConf() (*Conf, error)
}

func (util *Utility) GetRepo() repo.Repo {
	return nil
}
func (util *Utility) OpenRedis() (*redis.Client, error) {
	client, err := redis.Dial("tcp", "localhost:37564")
	if err != nil {
		return nil, err
	}
	return client, nil
}
func (util *Utility) PostHTTP(url string, body string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	if _, err := client.Do(req); err != nil {
		return err
	}
	return nil
}

type Conf struct {
	Admins       []string `json:admins`
	Token        string   `json:readToken`
	Channel      string   `json:channel`
	ChannelID    string   `json:channelId`
	WebhookToken string   `json:webhookToken`
}

func (util *Utility) GetAuthorizedUsers() []string {
	conf, err := util.GetConf()
	if err != nil {
		return []string{}
	}
	return conf.Admins
}
func (util *Utility) GetConf() (*Conf, error) {
	file, err := os.Open("conf.json")
	if err != nil {
		return nil, err
	}
	c := &Conf{}
	err = json.NewDecoder(file).Decode(c)
	if err != nil {
		return nil, err
	}
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
	resp, err := http.Get("https://slack.com/api/channels.info?token=" + conf.Token + "&channel=" + conf.ChannelID)

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
	baseUserInfoReqUrl := "https://slack.com/api/users.info?token=" + conf.Token + "&user="
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
