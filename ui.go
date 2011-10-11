package main

import (
	"bytes"
	"fmt"
	"strconv"
	stfl "github.com/akrennmair/go-stfl"
)

type UserInterface struct {
	form                  *stfl.Form
	actionchan            chan UserInterfaceAction
	tweetchan             chan []*Tweet
	updatechan            chan Tweet
	lookupchan            chan TweetRequest
	in_reply_to_status_id int64
}

type ActionId int

const (
	RESET_LAST_LINE ActionId = iota
	RAW_INPUT
)

type UserInterfaceAction struct {
	Action ActionId
	Args   []string
}

func NewUserInterface(tc chan []*Tweet, uc chan Tweet, lc chan TweetRequest) *UserInterface {
	stfl.Init()
	ui := &UserInterface{
		form:                  stfl.Create("<ui.stfl>"),
		actionchan:            make(chan UserInterfaceAction, 10),
		tweetchan:             tc,
		updatechan:            uc,
		in_reply_to_status_id: 0,
		lookupchan:            lc,
	}
	return ui
}

func (ui *UserInterface) GetActionChannel() chan UserInterfaceAction {
	return ui.actionchan
}

func (ui *UserInterface) Run() {
	for {
		select {
		case newtweets := <-ui.tweetchan:
			str := formatTweets(newtweets)
			ui.form.Modify("tweets", "insert_inner", str)
			ui.form.Run(-1)
		case action := <-ui.actionchan:
			ui.HandleAction(action)
		}
	}
}

func (ui *UserInterface) HandleAction(action UserInterfaceAction) {
	switch action.Action {
	case RESET_LAST_LINE:
		ui.ResetLastLine()
	case RAW_INPUT:
		input := action.Args[0]
		ui.HandleRawInput(input)
	}
}

func (ui *UserInterface) ResetLastLine() {
	ui.form.Modify("lastline", "replace", "{hbox[lastline] .expand:0 {label text[msg]:\"\" .expand:h}}")
}

func (ui *UserInterface) HandleRawInput(input string) {
	switch input {
	case "ENTER":
		ui.SetInputField("Tweet: ", "", "end-input")
	case "r":
		ui.in_reply_to_status_id, _ = strconv.Atoi64(ui.form.Get("status_id"))
		tweet := ui.LookupTweet(ui.in_reply_to_status_id)
		if tweet != nil {
			ui.SetInputField("Reply: ", "@"+*tweet.User.Screen_name+" ","end-input")
		} else {
			//TODO: show error
		}
	case "end-input":
		tweet_text := new(string)
		*tweet_text = ui.form.Get("inputfield")
		if len(*tweet_text) > 0 {
			t := Tweet{Text: tweet_text}
			if ui.in_reply_to_status_id != 0 {
				t.In_reply_to_status_id = new(int64)
				*t.In_reply_to_status_id = ui.in_reply_to_status_id
				ui.in_reply_to_status_id = int64(0)
			}
			ui.updatechan <- t
		}
		ui.ResetLastLine()
	case "cancel-input":
		ui.ResetLastLine()
	}
	ui.form.Run(-1)
}

func (ui *UserInterface) LookupTweet(status_id int64) *Tweet {
	reply := make(chan *Tweet)
	req := TweetRequest{Status_id: status_id, Reply: reply}
	ui.lookupchan <- req
	return <-reply
}

func (ui *UserInterface) InputLoop() {
	event := ""
	for event != "q" {
		event = ui.form.Run(0)
		if event != "" {
			ui.actionchan <- UserInterfaceAction{RAW_INPUT, []string{event}}
		}
	}
	stfl.Reset()
}

func (ui *UserInterface) SetInputField(prompt, deftext, endevent string) {
	last_line_text := "{hbox[lastline] .expand:0 {label .expand:0 text[prompt]:" + stfl.Quote(prompt) + "}{input[tweetinput] on_ESC:cancel-input on_ENTER:" + endevent + " modal:1 .expand:h text[inputfield]:" + stfl.Quote(deftext) + "}}"

	ui.form.Modify("lastline", "replace", last_line_text)
	ui.form.SetFocus("tweetinput")
}

func formatTweets(tweets []*Tweet) string {
	buf := bytes.NewBufferString("{list")

	for _, t := range tweets {
		tweetline := fmt.Sprintf("[%16s] %s", "@"+*t.User.Screen_name, *t.Text)
		buf.WriteString(fmt.Sprintf("{listitem[%v] text:%v}", *t.Id, stfl.Quote(tweetline)))
	}

	buf.WriteString("}")
	return string(buf.Bytes())
}
