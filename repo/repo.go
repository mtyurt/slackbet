package repo

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"strconv"

	"github.com/mediocregopher/radix.v2/redis"
)

var commonIV = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}

type Repo interface {
	AddNewBet(int, string) error
	BetIDExists(betID int) (bool, error)
	GetBetDetails(int) ([]BetDetail, error)
	GetIDOfOpenBet() (int, error)
	GetLastBetID() (int, error)
	GetWinnerScore(int) (int, error)
	SetBetAsEnded(int, string) error
	SetBetDetail(int, []BetDetail) error
	SetBetWinner(int, int) error
	GetBetSummary(betID int) (*BetSummary, error)
}
type RedisRepo struct {
	Url           string
	EncryptionKey string
}
type BetSummary struct {
	ID           int
	Status       string
	StartDate    string
	EndDate      string
	WinnerNumber int
}

func (b *BetSummary) String() string {
	str := strconv.Itoa(b.ID) + "\tstart: " + b.StartDate
	if b.Status == "open" {
		str += "\t(still open)"
	} else if b.EndDate != "" {
		str += "\tend: " + b.EndDate
	}
	if b.WinnerNumber != -1 {
		str += "\twinner score: " + strconv.Itoa(b.WinnerNumber)
	}
	return str
}

type BetDetail struct {
	User      string
	Number    int
	ExtraInfo string
}

// SetBetWinner sets the winner field of the bet.
// returns error in case of a connection error.
func (repo *RedisRepo) SetBetWinner(betID int, winner int) error {
	client, err := repo.openRedisClient()
	if err != nil {
		return err
	}
	err = client.Cmd("HSET", betID, "winner", winner).Err
	if err != nil {
		return err
	}
	return nil
}

// BetIDExists returns true if a bet with given id exists
// returns error for any connection error
func (repo *RedisRepo) BetIDExists(betID int) (bool, error) {
	client, err := repo.openRedisClient()
	if err != nil {
		return false, err
	}
	defer client.Close()
	exists, err := client.Cmd("EXISTS", betID).Int()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// GetBetSummary returns summary of bet with ID
// return error for any connection error
func (repo *RedisRepo) GetBetSummary(betID int) (*BetSummary, error) {
	client, err := repo.openRedisClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	entry, err := client.Cmd("HGETALL", betID).Map()
	if err != nil {
		return nil, err
	}
	winnerNumber := -1
	if winnerStr, ok := entry["winner"]; ok {
		winnerNumber, err = strconv.Atoi(winnerStr)
		if err != nil {
			return nil, err
		}
	}
	return &BetSummary{Status: entry["status"],
		StartDate:    entry["startDate"],
		EndDate:      entry["endDate"],
		ID:           betID,
		WinnerNumber: winnerNumber}, nil
}

// GetBetDetails finds and returns details list of the bet.
// returns error in case of a connection error.
func (repo *RedisRepo) GetBetDetails(betID int) ([]BetDetail, error) {
	client, err := repo.openRedisClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()
	var details []BetDetail
	encodedDetails, err := client.Cmd("HGET", betID, "details").Str()
	if err != nil {
		return nil, err
	}
	detailsStr, err := decodeString([]byte(encodedDetails), repo.EncryptionKey)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(detailsStr, &details)
	if err != nil {
		return nil, err
	}
	return details, nil

}

func decodeString(cipheredtext []byte, key string) ([]byte, error) {
	if key == "" {
		return cipheredtext, nil
	}
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBDecrypter(c, commonIV)
	plaintext := make([]byte, len(cipheredtext))
	cfb.XORKeyStream(plaintext, cipheredtext)
	return plaintext, nil
}

// GetLastBetID returns the last inserted bet id into the system
// returns error in case of a connection error.
func (repo *RedisRepo) GetLastBetID() (int, error) {
	client, err := repo.openRedisClient()
	if err != nil {
		return -1, nil
	}
	defer client.Close()
	lastID, err := client.Cmd("GET", "LastID").Int()
	if err != nil {
		return -1, nil
	}
	return lastID, nil
}

// GetIDOfOpenBet returns the id of the open bet if there is any.
// in redis, openBet is indicated by `OpenBet` identifier.
// returns error in case of a connection error.
func (repo *RedisRepo) GetIDOfOpenBet() (int, error) {
	client, err := repo.openRedisClient()
	if err != nil {
		return -1, nil
	}
	defer client.Close()

	result := client.Cmd("GET", "OpenBet")
	if result.IsType(redis.Nil) {
		return -1, nil
	}
	return result.Int()
}

// GetWinnerScore returns the winner score that belongs to the bet with betID.
// Returns -1 if bet doesn't have a winnerScore.
// returns error in case of a connection error.
func (repo *RedisRepo) GetWinnerScore(betID int) (int, error) {
	client, err := repo.openRedisClient()
	if err != nil {
		return -1, nil
	}
	defer client.Close()

	scoreExists, err := client.Cmd("HEXISTS", betID, "winner").Int()
	if err != nil || scoreExists != 1 {
		return -1, err
	}
	winnerScore, err := client.Cmd("HGET", betID, "winner").Int()
	if err != nil {
		return -1, err
	}
	return winnerScore, nil
}

// SetBetAsEnded marks the bet as ended and sets the endDate with given date.
// returns error in case of a connection error.
func (repo *RedisRepo) SetBetAsEnded(betID int, date string) error {
	client, err := repo.openRedisClient()
	if err != nil {
		return nil
	}
	defer client.Close()

	err = client.Cmd("HMSET", betID, "status", "closed", "endDate", date).Err
	if err != nil {
		return err
	}
	err = client.Cmd("DEL", "OpenBet").Err
	if err != nil {
		return err
	}
	return nil
}

// AddNewBet adds a new bet info with given id and startDate.
// returns error in case of a connection error.
func (repo *RedisRepo) AddNewBet(betID int, startDate string) error {
	client, err := repo.openRedisClient()
	if err != nil {
		return nil
	}
	defer client.Close()

	client.PipeAppend("HMSET", strconv.Itoa(betID), "startDate", startDate, "status", "open", "details", "[]")
	client.PipeAppend("SET", "LastID", betID)
	client.PipeAppend("SET", "OpenBet", betID)
	if err = client.PipeResp().Err; err != nil {
		return err
	}
	if err = client.PipeResp().Err; err != nil {
		return err
	}
	if err = client.PipeResp().Err; err != nil {
		return err
	}
	client.PipeClear()
	return nil
}
func (repo *RedisRepo) SetBetDetail(betID int, details []BetDetail) error {
	client, err := repo.openRedisClient()
	if err != nil {
		return nil
	}
	defer client.Close()

	marshalledDetails, err := json.Marshal(details)
	if err != nil {
		return err
	}
	encodedDetails, err := encodeString(marshalledDetails, repo.EncryptionKey)
	if err != nil {
		return err
	}
	err = client.Cmd("HSET", betID, "details", encodedDetails).Err
	if err != nil {
		return err
	}
	return nil
}
func encodeString(plaintext []byte, key string) (string, error) {
	if key == "" {
		return string(plaintext), nil
	}
	c, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	cfb := cipher.NewCFBEncrypter(c, commonIV)
	ciphertext := make([]byte, len(plaintext))
	cfb.XORKeyStream(ciphertext, plaintext)
	return string(ciphertext), nil
}
func (repo *RedisRepo) openRedisClient() (*redis.Client, error) {
	client, err := redis.Dial("tcp", "localhost:37564")
	if err != nil {
		return nil, err
	}
	return client, nil
}
