package data

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"
)

// This should be separate from regular data classes in case the structures change and we need to restore data from an older version

type BackupUser struct {
	User
}

type BackupFeeditem struct {
	Feeditem
	FeeditemKey
}

type BackupPagemonitor struct {
	PagemonitorPage
	UserPagemonitor
}

type BackupData struct {
	Users        []*BackupUser
	Feeds        []*BackupFeeditem
	Pagemonitor  []*BackupPagemonitor
	ServerConfig map[string]string
}

func (service *DBService) Backup() (string, error) {
	data := BackupData{}

	done := make(chan bool)
	userChan := make(chan *User)
	go func() {
		for user := range userChan {
			backupUser := &BackupUser{User: *user}
			user.Username = ""
			data.Users = append(data.Users, backupUser)
		}
		done <- true
	}()
	if err := service.ReadAllUsers(userChan); err != nil {
		return "", errors.Wrap(err, "Error backing up users")
	}
	<-done

	feedChan := make(chan *Feeditem)
	go func() {
		for feedItem := range feedChan {
			// Flatten/reformat data
			backupFeeditem := &BackupFeeditem{
				Feeditem:    *feedItem,
				FeeditemKey: *feedItem.Key,
			}
			backupFeeditem.Feeditem.Key = nil
			data.Feeds = append(data.Feeds, backupFeeditem)
		}
		done <- true
	}()
	if err := service.ReadAllFeedItems(feedChan); err != nil {
		return "", errors.Wrap(err, "Error backing up feed items")
	}
	<-done

	pageChan := make(chan *PagemonitorPage)
	go func() {
		for page := range pageChan {
			// Flatten/reformat data
			backupPagemonitor := &BackupPagemonitor{
				PagemonitorPage: *page,
				UserPagemonitor: *page.Config,
			}
			backupPagemonitor.PagemonitorPage.Config = nil
			data.Pagemonitor = append(data.Pagemonitor, backupPagemonitor)
		}
		close(done)
	}()

	if err := service.ReadAllPages(pageChan); err != nil {
		return "", errors.Wrap(err, "Error backing up pagemonitor pages")
	}
	<-done

	serverConfig, err := service.GetAllConfigVariables()
	if err != nil {
		return "", errors.Wrap(err, "Error backing up server configuration")
	}
	data.ServerConfig = serverConfig

	value, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "Error marshaling json")
	}

	return string(value), nil
}

func (service *DBService) Restore(value string) error {
	data := BackupData{}
	failed := false
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		return errors.Wrap(err, "Error unmarshaling json")
	}

	for _, user := range data.Users {
		if err := service.NewUserService(user.Username).Save(&user.User); err != nil {
			failed = true
			log.Printf("Error saving user %v %v", user, err)
		}
	}
	convertFeeditems := func() []*Feeditem {
		convertedFeeditems := make([]*Feeditem, 0, len(data.Feeds))

		for _, feedItem := range data.Feeds {
			feedItem.Key = &feedItem.FeeditemKey
			convertedFeeditems = append(convertedFeeditems, &feedItem.Feeditem)
		}
		return convertedFeeditems
	}
	if err := service.SaveFeeditems(convertFeeditems()...); err != nil {
		failed = true
		log.Printf("Error saving feed items %v", err)
	}
	for _, page := range data.Pagemonitor {
		page.Config = &page.UserPagemonitor
		if err := service.SavePage(&page.PagemonitorPage); err != nil {
			failed = true
			log.Printf("Error saving page %v %v", page, err)
		}
	}
	for key, value := range data.ServerConfig {
		if err := service.SetConfigVariable(key, value); err != nil {
			failed = true
			log.Printf("Error saving config variable %v %v %v", key, value, err)
		}
	}
	if failed {
		return fmt.Errorf("Failed to restore at least one item")
	}
	return nil
}
