package main

import (
	"time"
)

type Model struct {
	updatechan   chan Tweet
	newtweetchan chan []*Tweet
	tapi         *TwitterAPI
	tweets       []*Tweet
	tweet_map    map[int64]*Tweet
	lookupchan   chan TweetRequest
}

type TweetRequest struct {
	Status_id int64
	Reply     chan *Tweet
}

func NewModel(t *TwitterAPI) *Model {
	model := &Model{
		updatechan:   make(chan Tweet, 1),
		newtweetchan: make(chan []*Tweet, 1),
		tapi:         t,
		lookupchan:   make(chan TweetRequest, 1),
		tweet_map:    map[int64]*Tweet{},
	}

	return model
}

func (m *Model) GetUpdateChannel() chan Tweet {
	return m.updatechan
}

func (m *Model) GetNewTweetChannel() chan []*Tweet {
	return m.newtweetchan
}

func (m *Model) GetLookupChannel() chan TweetRequest {
	return m.lookupchan
}

func (m *Model) Run() {
	ticker := make(chan int, 1)
	go Ticker(ticker, 20e9)

	last_id := int64(0)

	for {
		select {
		case tweet := <-m.updatechan:
			if newtweet, err := m.tapi.Update(tweet); err == nil {
				m.tweet_map[*newtweet.Id] = newtweet
				last_id = *newtweet.Id
				m.tweets = append([]*Tweet{newtweet}, m.tweets...)
				m.newtweetchan <- []*Tweet{newtweet}
			}
		case req := <-m.lookupchan:
			tweet := m.tweet_map[req.Status_id]
			req.Reply <- tweet
			close(req.Reply)
		case <-ticker:
			home_tl, err := m.tapi.HomeTimeline(50, last_id)

			if err != nil {
				//TODO: signal error
			} else {
				if len(home_tl.Tweets) > 0 {
					for _, t := range home_tl.Tweets {
						m.tweet_map[*t.Id] = t
					}
					m.tweets = append(home_tl.Tweets, m.tweets...)
					m.newtweetchan <- home_tl.Tweets
					if home_tl.Tweets[0].Id != nil {
						last_id = *home_tl.Tweets[0].Id
					}
				}
			}
		}
	}
}

func Ticker(tickchan chan int, ns int64) {
	for {
		tickchan <- 1
		time.Sleep(ns)
	}
}
