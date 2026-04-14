// Package caldav implements a minimal CalDAV client for syncing calendar events.
// Uses only net/http and encoding/xml — no external CalDAV library.
package caldav

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Event is a parsed VEVENT from a CalDAV source.
type Event struct {
	UID         string
	ETag        string
	Title       string
	Description string
	Location    string
	StartAt     time.Time
	EndAt       time.Time
	IsAllDay    bool
	RRule       string
}

// Client performs CalDAV REPORT requests against a single calendar URL.
type Client struct {
	URL        string
	Username   string
	Password   string
	httpClient *http.Client
}

// NewClient creates a Client with a 30-second timeout.
func NewClient(url, username, password string) *Client {
	return &Client{
		URL:        url,
		Username:   username,
		Password:   password,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

const reportBody = `<?xml version="1.0" encoding="UTF-8"?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:getetag/>
    <C:calendar-data/>
  </D:prop>
  <C:filter>
    <C:comp-filter name="VCALENDAR">
      <C:comp-filter name="VEVENT">
        <C:time-range start="%s" end="%s"/>
      </C:comp-filter>
    </C:comp-filter>
  </C:filter>
</C:calendar-query>`

// FetchEvents fetches VEVENT objects in [from, to] from the CalDAV endpoint.
func (c *Client) FetchEvents(from, to time.Time) ([]Event, error) {
	body := fmt.Sprintf(reportBody,
		from.UTC().Format("20060102T150405Z"),
		to.UTC().Format("20060102T150405Z"),
	)

	req, err := http.NewRequest("REPORT", c.URL, bytes.NewBufferString(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Depth", "1")
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("caldav REPORT: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("caldav REPORT %d: %s", resp.StatusCode, raw)
	}

	return parseMultistatus(resp.Body)
}

// ── XML parsing ───────────────────────────────────────────────────────────────

type xmlMultistatus struct {
	Responses []xmlResponse `xml:"response"`
}

type xmlResponse struct {
	Propstats []xmlPropstat `xml:"propstat"`
}

type xmlPropstat struct {
	Status string   `xml:"status"`
	Prop   xmlProp  `xml:"prop"`
}

type xmlProp struct {
	ETag         string `xml:"getetag"`
	CalendarData string `xml:"calendar-data"`
}

func parseMultistatus(r io.Reader) ([]Event, error) {
	var ms xmlMultistatus
	dec := xml.NewDecoder(r)
	dec.DefaultSpace = "DAV:"
	if err := dec.Decode(&ms); err != nil {
		return nil, fmt.Errorf("parse multistatus: %w", err)
	}

	var events []Event
	for _, resp := range ms.Responses {
		for _, ps := range resp.Propstats {
			if !strings.Contains(ps.Status, "200") {
				continue
			}
			if ps.Prop.CalendarData == "" {
				continue
			}
			etag := strings.Trim(ps.Prop.ETag, `"`)
			evs, err := parseICalendar(ps.Prop.CalendarData, etag)
			if err != nil {
				continue // skip malformed events
			}
			events = append(events, evs...)
		}
	}
	return events, nil
}

// ── iCalendar parser ──────────────────────────────────────────────────────────

// parseICalendar parses one VCALENDAR blob (may contain multiple VEVENTs).
func parseICalendar(data, etag string) ([]Event, error) {
	lines := unfoldLines(data)

	var events []Event
	inEvent := false
	var cur map[string]string

	for _, line := range lines {
		switch {
		case line == "BEGIN:VEVENT":
			inEvent = true
			cur = make(map[string]string)
		case line == "END:VEVENT":
			if inEvent {
				ev := buildEvent(cur, etag)
				if ev.UID != "" && !ev.StartAt.IsZero() {
					events = append(events, ev)
				}
			}
			inEvent = false
			cur = nil
		case inEvent:
			k, v := splitProperty(line)
			// For properties with parameters (e.g. DTSTART;TZID=...), keep base key
			baseKey := strings.SplitN(k, ";", 2)[0]
			cur[baseKey] = v
			// Also store the full key for TZID lookups
			if k != baseKey {
				cur[k] = v
			}
		}
	}
	return events, nil
}

// unfoldLines joins folded iCal lines (continuation lines start with space/tab).
func unfoldLines(data string) []string {
	raw := strings.ReplaceAll(data, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	var result []string
	for _, line := range strings.Split(raw, "\n") {
		if len(line) == 0 {
			continue
		}
		if (line[0] == ' ' || line[0] == '\t') && len(result) > 0 {
			result[len(result)-1] += line[1:]
		} else {
			result = append(result, line)
		}
	}
	return result
}

// splitProperty splits "KEY:VALUE" or "KEY;PARAM=X:VALUE" into (KEY;PARAM=X, VALUE).
func splitProperty(line string) (key, value string) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, ""
	}
	return line[:idx], line[idx+1:]
}

func buildEvent(props map[string]string, etag string) Event {
	ev := Event{
		UID:         props["UID"],
		ETag:        etag,
		Title:       unescapeIcal(props["SUMMARY"]),
		Description: unescapeIcal(props["DESCRIPTION"]),
		Location:    unescapeIcal(props["LOCATION"]),
		RRule:       props["RRULE"],
	}

	ev.StartAt, ev.IsAllDay = parseICalDate(props["DTSTART"])
	ev.EndAt, _ = parseICalDate(props["DTEND"])

	// If DTEND missing and all-day, set to start+1 day
	if ev.EndAt.IsZero() && ev.IsAllDay {
		ev.EndAt = ev.StartAt.AddDate(0, 0, 1)
	}

	return ev
}

// parseICalDate parses DATE (YYYYMMDD) or DATE-TIME (YYYYMMDDTHHmmSSZ or local).
// Returns (time.Time, isAllDay).
func parseICalDate(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	// Strip any TZID parameter that might have leaked through (shouldn't with splitProperty)
	if idx := strings.IndexByte(s, ':'); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimSpace(s)

	// All-day: YYYYMMDD
	if len(s) == 8 {
		t, err := time.Parse("20060102", s)
		return t, err == nil
	}

	// UTC datetime
	if strings.HasSuffix(s, "Z") {
		t, _ := time.Parse("20060102T150405Z", s)
		return t, false
	}

	// Floating datetime (no timezone) — treat as UTC
	if len(s) == 15 {
		t, err := time.Parse("20060102T150405", s)
		_ = err
		return t, false
	}

	return time.Time{}, false
}

// unescapeIcal handles basic iCal text escaping: \n \, \; \\
func unescapeIcal(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\N`, "\n")
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\;`, ";")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
