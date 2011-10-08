package main

import (
	"fmt"
	"os"
	"json"
	"io/ioutil"
	"time"
	"bytes"
	stfl "github.com/akrennmair/go-stfl"
	oauth "github.com/hokapoka/goauth"
)

func main() {

	tapi := NewTwitterAPI("sDggzGbHbyAfl5fJ87XOCA", "MOCQDL7ot7qIxMwYL5x1mMAqiYBYxNTxPWS6tc6hw")

	at, aterr := LoadAccessToken()

	if aterr == nil {
		tapi.SetAccessToken(at)
	} else {
		auth_url, err := tapi.GetRequestAuthorizationURL()
		if err != nil {
			fmt.Println(err.String())
			return
		}

		var pin string
		fmt.Printf("Open %s\n", auth_url)
		fmt.Printf("PIN Number: ")
		fmt.Scanln(&pin)

		tapi.SetPIN(pin)

		if saveerr := SaveAccessToken(tapi.GetAccessToken()); saveerr != nil {
			fmt.Printf("saving access token failed: %s\n", saveerr.String())
			return
		}
	}

	newtweetchan := make(chan []Tweet, 1)
	viewchan := make(chan []Tweet, 1)

	go func() {
		last_id := int64(0)

		for {

			home_tl, err := tapi.HomeTimeline(0, last_id)

			if err != nil {
				//fmt.Println(err.String())
				//TODO: signal error
			} else {
				if len(home_tl.Tweets) > 0 {
					newtweetchan <- home_tl.Tweets
					if home_tl.Tweets[0].Id != nil {
						last_id = *home_tl.Tweets[0].Id
					}
				}
			}

			time.Sleep(20e9)
		}
	}()

	go func() {
		tweets := []Tweet{}

		for {
			select {
			case newtweets := <-newtweetchan:
				//fmt.Fprintf(os.Stderr, "received %d tweets in controller\n", len(newtweets))
				tweets = append(newtweets, tweets...)
				viewchan <- newtweets
			}
		}

	}()

	form := stfl.Create("<ui.stfl>")
	form.Set("program", os.Args[0])


	go func() {

		for {
			select {
			case newtweets := <-viewchan:
				//fmt.Fprintf(os.Stderr, "received %d tweets in view\n", len(newtweets))
				str := formatTweets(newtweets)
				//fmt.Fprintf(os.Stderr, "formatted new tweets: %s\n", str)
				form.Modify("tweets", "insert_inner", str)
				form.Run(-1)
			}
		}
	}()

	event := ""
	for event != "q" {
		event = form.Run(0)
	}

	stfl.Reset()


//	foo, posterr := goauthcon.Post(
//		"http://api.twitter.com/1/statuses/update.json",
//		oauth.Params{
//			&oauth.Pair{
//				Key:"status",
//				Value:"Test posting using Gockel prototype",
//			},
//		}, at )
//
//	fmt.Printf("foo = %v\n", foo)
//
//	if posterr != nil {
//		fmt.Println(err.String())
//		return
//	}
//
//	fmt.Println("Twitter Status is updated")
}

func SaveAccessToken(at *oauth.AccessToken) os.Error {
	data, marshalerr := json.Marshal(at)
	if marshalerr != nil {
		return marshalerr
	}

	f, ferr := os.OpenFile("access_token.json", os.O_WRONLY | os.O_CREATE, 0600)
	if ferr != nil {
		return ferr
	}
	defer f.Close()

	_, werr := f.Write(data)
	if werr != nil {
		return werr
	}

	return nil
}

func LoadAccessToken() (*oauth.AccessToken, os.Error) {
	f, ferr := os.Open("access_token.json")
	if ferr != nil {
		return nil, ferr
	}
	defer f.Close()

	data, readerr := ioutil.ReadAll(f)
	if readerr != nil {
		return nil, readerr
	}

	at := &oauth.AccessToken{}
	
	err := json.Unmarshal(data, at)
	if err != nil {
		return nil, err
	}

	return at, nil
}

func formatTweets(tweets []Tweet) string {
	buf := bytes.NewBufferString("{list")

	for _, t := range tweets {
		tweetline := fmt.Sprintf("[%s] %s", *t.User.Screen_name, *t.Text)
		buf.WriteString("{listitem text:")
		buf.WriteString(stfl.Quote(tweetline))
		buf.WriteString("}")
	}

	buf.WriteString("}")
	return string(buf.Bytes())
}
