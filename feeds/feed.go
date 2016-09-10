package feeds

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron"
	"github.com/uber-go/zap"
	"gopkg.in/telegram-bot-api.v4"
)

var (
	FEEDS_TABLE = "feeds"
)

type (
	pgsqlStore struct {
		pool *sql.DB
	}

	feedURL struct {
		ID             int64
		URL            string
		NewestItemDate time.Time
	}

	feedStore interface {
		// Lese alle gespeicherten Feeds aus der Datenbank
		ReadFeeds() ([]feedURL, error)
		// Update neustes Item Datum
		UpdateNewsItemDate(id int64, newestDate time.Time) error
	}

	pipe interface {
		// Sende neues Item an Receiver.
		// item ist die Position in feed.Items des neuen Items
		Send(feed *gofeed.Feed, item int) error
	}

	feedCtrl struct {
		log      zap.Logger
		store    feedStore
		telegram *tgbotapi.BotAPI
		chatID   int64
	}
)

// Starte Feed Controller. Dieser prüft in regelmäßigen Abständen
// für jede hinterlegt RSS URL ob es einen neuen Beitrag gibt.
// Gibt es einen neuen Beitrag wird dieser an den Benutzer gesendet.
// Momentan wird Telegram als Empfänger unterstüzt.
func StartCtrl(log zap.Logger, db *sql.DB, chatID int64, telegram *tgbotapi.BotAPI) {
	ctrl := feedCtrl{
		log:      log,
		store:    &pgsqlStore{db},
		telegram: telegram,
		chatID:   chatID,
	}

	c := cron.New()
	c.AddFunc("@every 5s", ctrl.run)
	c.Start()
}

func (ctrl feedCtrl) run() {
	ctrl.log.Debug("Start fetching Feeds updates")
	feeds, err := ctrl.store.ReadFeeds()
	if err != nil {
		ctrl.log.Error("Cannot read feeds", zap.Error(err))
		return
	}

	for _, feed := range feeds {
		ctrl.log.Debug("Fetch updates", zap.String("url", feed.URL), zap.Time("newest_date", feed.NewestItemDate))
		err := ctrl.fetchUpdate(feed)
		if err != nil {
			ctrl.log.Error("Cannot fetch feed updates", zap.Error(err))
		}
	}
}

// Lädt den aktuellen Feed für die übergeben URL. Läuft durch alle gefunden Items
// gibt es darin ein Eintrag der neuer ist das Datum des neuesten Eintrags des
// letzten Feed updates sende diesen Eintrag an den Receiver.
// Am Ende jedes Updates wird das neueste Datum der neu gefunden Einträge
// in die Datenbank geschrieben.
func (ctrl feedCtrl) fetchUpdate(feedURL feedURL) error {
	msg := fmt.Sprintf("Attempt to fetch updates from %v", feedURL.URL)
	ctrl.log.Debug(msg)

	p := gofeed.NewParser()
	feed, err := p.ParseURL(feedURL.URL)
	if err != nil {
		return err
	}

	dates := []time.Time{}
	for index, item := range feed.Items {
		// Neues Item gefunden
		newestDate := findNewestDate(item)
		if feedURL.NewestItemDate.Before(newestDate) {
			err := ctrl.sendToTelegram(feed, index)
			if err != nil {
				ctrl.log.Error("Cannot send new item to user", zap.Error(err))
				continue
			}

			dates = append(dates, newestDate)
		}
	}

	newestDateOfItems := findNewestDateOfItems(dates)

	if !newestDateOfItems.IsZero() {
		err = ctrl.store.UpdateNewsItemDate(feedURL.ID, newestDateOfItems)
		if err != nil {
			return err
		}
	}

	return nil
}

// Sende neuen Beitrag an Telegram
func (ctrl feedCtrl) sendToTelegram(feed *gofeed.Feed, item int) error {
	i := feed.Items[item]
	ctrl.log.Debug("Send new feed items to telegram", zap.String("feed", feed.Title), zap.String("title", i.Title))
	text := fmt.Sprintf("%v\n%v - %v", feed.Title, i.Title, i.Link)
	msg := tgbotapi.NewMessage(ctrl.chatID, text)
	ctrl.telegram.Send(msg)
	return nil
}

// Prüft ob das Published Datum oder das Updated Datum neuer ist und gibt das neuer zurück
func findNewestDate(item *gofeed.Item) time.Time {
	if item.UpdatedParsed == nil {
		return *item.PublishedParsed
	}
	if item.PublishedParsed.Before(*item.UpdatedParsed) {
		return *item.UpdatedParsed
	}

	return *item.PublishedParsed
}

// Finde das neuste Datum von allen gefunden Items
func findNewestDateOfItems(dates []time.Time) time.Time {
	if len(dates) == 0 {
		return time.Time{}
	}

	n := dates[0]
	for _, d := range dates[1:] {
		if d.After(n) {
			n = d
		}
	}

	return n
}

// Lese alle Feeds aus Postgres Datenbank
func (db *pgsqlStore) ReadFeeds() ([]feedURL, error) {
	q := fmt.Sprintf("SELECT * FROM %v", FEEDS_TABLE)
	iter, err := db.pool.Query(q)
	if err != nil {
		return []feedURL{}, err
	}

	feedURLs := []feedURL{}
	for iter.Next() {
		feed := feedURL{}
		err := iter.Scan(&feed.ID, &feed.URL, &feed.NewestItemDate)
		if err != nil {
			return []feedURL{}, err
		}

		feedURLs = append(feedURLs, feed)
	}

	return feedURLs, nil
}

// Aktualisiere Datum
func (db *pgsqlStore) UpdateNewsItemDate(id int64, newestDate time.Time) error {
	q := fmt.Sprintf("UPDATE %v SET newest_date=$1 WHERE id=$2", FEEDS_TABLE)
	_, err := db.pool.Exec(q, newestDate, id)
	if err != nil {
		return err
	}

	return nil
}
