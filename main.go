package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"

	"github.com/ChimeraCoder/anaconda"
	"github.com/aryann/difflib"
)

const (
	CharSet = "UTF-8"
)

type Config struct {
	TwitterScreenName        string
	TwitterConsumerKey       string
	TwitterConsumerSecret    string
	TwitterAccessToken       string
	TwitterAccessTokenSecret string

	//if these are set, it'll attempt to send email via AWS SES
	//If sending via AWS SES, you MUST set AWS_ACCESS_KEY_ID and AWS_SECRET_KEY environment variables.
	FromEmail string
	ToEmail   string
}

type User struct {
	Id          int64
	Name        string
	ScreenName  string
	Description string
	Location    string
	ProfileUrl  string
}

func (u User) ToString() string {
	u.Description = strings.Replace(u.Description, "\n", " ", -1)
	u.Description = strings.Replace(u.Description, "\r", " ", -1)
	return fmt.Sprintf("%+v", u)
}

//gross user sorting stuff
type ById []User

func (u ById) Len() int           { return len(u) }
func (u ById) Swap(i, j int)      { u[i], u[j] = u[j], u[i] }
func (u ById) Less(i, j int) bool { return u[i].Id < u[j].Id }

var conf = readConfig()
var logger = log.New(os.Stdout, "", log.LstdFlags)

func main() {
	users := getFriendsAsUsers()
	userList := usersToString(users)
	prepareFiles()
	saveUsers(userList)
	added, deleted := diff()

	logger.Printf("Added: \n\n%+v\n\nDeleted: \n\n%+v\n\n", added, deleted)
	if conf.ToEmail != "" {
		email(added, deleted)
	}
}

//readConfig reads a "config.json" file in the directory from which the app is executed.
//see config_sample.json for what it should look like
func readConfig() Config {
	file, err := os.Open("config.json")
	if err != nil {
		logger.Panic("Could not open config.json %+v", err)
	}

	decoder := json.NewDecoder(file)
	conf := Config{}
	err = decoder.Decode(&conf)
	if err != nil {
		logger.Panic("Error decoding json config", err)
	}

	logger.Println("Configuration complete")
	return conf
}

//getFriendsAsUsers returns twitter friends as a slice of users sorted by User ID
func getFriendsAsUsers() []User {
	anaconda.SetConsumerKey(conf.TwitterConsumerKey)
	anaconda.SetConsumerSecret(conf.TwitterConsumerSecret)
	api := anaconda.NewTwitterApi(conf.TwitterAccessToken, conf.TwitterAccessTokenSecret)
	result := make(chan anaconda.UserCursor, 1000000) //YOLO

	logger.Println("getting friends")

	v := url.Values{}
	v.Set("screen_name", conf.TwitterScreenName)
	getFriends(api, result, v)

	logger.Println("got em")

	users := []User{}

	for uc := range result {
		for _, u := range uc.Users {
			users = append(users, User{u.Id, u.Name, u.ScreenName, u.Description, u.Location, "https://twitter.com/" + u.ScreenName})
		}
	}

	sort.Sort(ById(users))
	return users
}

//getFriends uses the Twitter API to get people you follow, which it dubs "friends". Don't blame me for twitter's sense of self-importance.
func getFriends(api *anaconda.TwitterApi, result chan anaconda.UserCursor, v url.Values) {
	v.Set("count", "200")
	v.Set("skip_status", "true")
	v.Set("include_user_entities", "false")

	cursor, err := api.GetFriendsList(v)
	if err != nil {
		logger.Panic("Error getting friends from twitter", err)
	}
	result <- cursor

	if cursor.Next_cursor == 0 {
		close(result)
		return
	}

	logger.Println("Fetching next set of friends, starting with cursor ", cursor.Next_cursor_str)
	v.Set("cursor", cursor.Next_cursor_str)
	getFriends(api, result, v)
}

//usersToString turns the slice of Users into a slice of strings; each string is a single-line representation of the user
func usersToString(users []User) string {
	var buf []string
	for _, u := range users {
		buf = append(buf, u.ToString())
	}

	return strings.Join(buf, "\n")
}

//forgive me
const CURRENT = "current.txt"
const PREV = "prev.txt"
const THIRD = "third.txt"

//prepareFiles shuffles files around. This is so chump.
func prepareFiles() {
	_, pperr := os.Stat(THIRD)
	_, perr := os.Stat(PREV)
	_, cerr := os.Stat(CURRENT)

	if pperr == nil && perr == nil {
		os.Remove(THIRD)
	}
	if perr == nil {
		os.Rename(PREV, THIRD)
	}

	if cerr == nil {
		os.Rename(CURRENT, PREV)
	}
}

//saveUsers saves the user list, which is a newline-delimited string of users, in a text file
func saveUsers(userList string) {
	err := ioutil.WriteFile(CURRENT, []byte(userList), 0644)
	if err != nil {
		logger.Panic("Error saving users", err)
	}
}

//diff returns a slice of added users and a slice of deleted users; this is lame in that if a user's name, location, or description changes, it doesn't currently identify it as "changed" but just as both added and deleted
func diff() ([]string, []string) {
	prev, _ := ioutil.ReadFile(PREV)
	current, _ := ioutil.ReadFile(CURRENT)

	rawdiff := difflib.Diff(strings.Split(string(current), "\n"), strings.Split(string(prev), "\n"))

	var added []string
	var deleted []string
	for _, record := range rawdiff {
		if record.Delta == difflib.LeftOnly {
			added = append(added, record.Payload)
		}
		if record.Delta == difflib.RightOnly {
			deleted = append(deleted, record.Payload)
		}
	}
	return added, deleted
}

//email uses AWS SES to send an email with the results. It's an ugly email.
func email(added, deleted []string) {

	addedList := ""
	deletedList := ""

	for _, u := range added {
		addedList += u + "\n\n"
	}
	for _, u := range deleted {
		deletedList += u + "\n\n"
	}

	logger.Println("Deleted:")
	for _, u := range deleted {
		logger.Println(u)
	}

	message := fmt.Sprintf("%d Added, %d Deleted\n\nAdded: \n\n %v \n\nDeleted: \n\n %v\n\n", len(added), len(deleted), addedList, deletedList)
	logger.Println("message is " + message)

	Email(conf.FromEmail, conf.ToEmail, "Twitter friend changes", message)
}

func Email(from, to, subject, text string) {

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	svc := ses.New(sess)

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(to),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(text),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(from),
	}

	result, err := svc.SendEmail(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}

		return
	}

	fmt.Println("Email Sent to address: " + to)
	fmt.Println(result)
}
