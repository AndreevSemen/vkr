package detector

import (
	"gossip_emulation/detect/types"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPDetector(t *testing.T) {
	cfg := HTTPDetectorConfig{
		Hosts: []url.URL{
			{
				Scheme: "http",
				Host:   "ya.ru",
			},
			{
				Scheme: "http",
				Host:   "mail.ru",
			},
		},
		CheckInterval: time.Second,
		HealthCheck: func(r *http.Response) bool {
			return http.StatusOK == r.StatusCode
		},
		HTTPTimeout: time.Second,
		TCPTimeout:  time.Second,
		ICMPTimeout: time.Second,
	}

	d, err := NewHTTPDetector(cfg)
	require.NoError(t, err)

	expectedYaRu := types.Event{
		Hostname: "ya.ru",
		State:    types.AppState_AppAvailable,
	}
	expectedMailRu := types.Event{
		Hostname: "mail.ru",
		State:    types.AppState_AppAvailable,
	}

	e := <-d.DetectorEvents()
	if e.Hostname == "ya.ru" {
		require.Equal(t, expectedYaRu, e)
		e = <-d.DetectorEvents()
		require.Equal(t, expectedMailRu, e)
	} else {
		require.Equal(t, expectedMailRu, e)
		e = <-d.DetectorEvents()
		require.Equal(t, expectedYaRu, e)
	}

	err = d.Close()
	require.NoError(t, err)
}
