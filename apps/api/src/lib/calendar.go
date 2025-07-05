package lib

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

var calsvc *calendar.Service

func getCalendarClient(conf *oauth2.Config) (*http.Client, error) {
	tokFile, err := os.Open("token.json")
	if err != nil {
		return nil, err
	}
	defer tokFile.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(tokFile).Decode(tok); err != nil {
		return nil, err
	}

	cli := conf.Client(context.Background(), tok)
	return cli, nil
}
func gapiGetCalendarService() (svc *calendar.Service, err error) {
	if calsvc != nil {
		return calsvc, nil
	}
	secretsPath := os.Getenv("SECRETS_DIR")
	b, err := os.ReadFile(path.Join(secretsPath, "admin-sdk-credentials.json"))
	if err != nil {
		return nil, err
	}
	conf, err := google.ConfigFromJSON(b)
	if err != nil {
		return nil, err
	}
	cli, _ := getCalendarClient(conf)
	srv, err := calendar.NewService(context.Background(), option.WithHTTPClient(cli))
	return srv, err
}
func GAPICreateCalendarService(ctx context.Context, tok *oauth2.Token, conf *oauth2.Config) (svc *calendar.Service, err error) {
	if conf != nil {
		return calendar.NewService(ctx, option.WithTokenSource(conf.TokenSource(ctx, tok)))
	}
	secretsPath := os.Getenv("SECRETS_DIR")
	cc := path.Join(secretsPath, "client_secret.json")
	log.Printf("cc: %s\n", cc)
	b, err := os.ReadFile(cc)
	if err != nil {
		return nil, err
	}
	conf, err = google.ConfigFromJSON(b)
	if err != nil {
		return nil, err
	}
	cli := conf.Client(ctx, tok)
	return calendar.NewService(ctx, option.WithHTTPClient(cli))
}

func GAPIGetPrimaryCalendar() (cal *calendar.Calendar, err error) {
	return GAPIGetCalendar("primary", nil)
}
func GAPIGetCalendar(c string, s *calendar.Service) (cal *calendar.Calendar, err error) {
	if s == nil {
		s, err = gapiGetCalendarService()
		if err != nil {
			return nil, err
		}
	}
	cli := s.Calendars.Get(c)
	return cli.Do()
}
func GAPIGetEvents(calId string, s *calendar.Service) (list *calendar.Events, err error) {
	if s == nil {
		s, err = gapiGetCalendarService()
		if err != nil {
			return nil, err
		}
	}
	cli := s.Events.List(calId)
	return cli.Do()
}
func GAPIAddEvent(calId string, e *calendar.Event, s *calendar.Service) (err error) {
	if s == nil {
		s, err = gapiGetCalendarService()
		if err != nil {
			return err
		}
	}
	cli := s.Events.Insert(calId, e)
	_, err = cli.Do()
	return err
}
func GAPIUpdateEvent(calId string, e *calendar.Event, s *calendar.Service) (err error) {
	if s == nil {
		s, err = gapiGetCalendarService()
		if err != nil {
			return err
		}
	}
	cli := s.Events.Update(calId, e.Id, e)
	_, err = cli.Do()
	return err
}
func GAPIAddCalendar(name string, s *calendar.Service) (cal *calendar.Calendar, err error) {
	if s == nil {
		s, err = gapiGetCalendarService()
		if err != nil {
			return nil, err
		}
	}
	cli := s.Calendars.Insert(&calendar.Calendar{
		Summary: name,
	})
	return cli.Do()
}
