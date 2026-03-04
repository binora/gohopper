package osm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var iso8601Re = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)(?:\.(\d+))?S)?)?$`)

// parseDuration parses a duration string (hh:mm, hh:mm:ss, mm, or ISO 8601)
// and returns the duration in seconds.
func parseDuration(str string) (int64, error) {
	if str == "" {
		return 0, nil
	}
	if strings.HasPrefix(str, "P") {
		return parseISO8601Duration(str)
	}
	if !strings.Contains(str, ":") {
		return parseMinutesOnly(str)
	}
	return parseColonDuration(str)
}

// parseMinutesOnly parses a plain number as minutes and returns seconds.
func parseMinutesOnly(str string) (int64, error) {
	minutes, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("cannot parse duration tag value: %s", str)
	}
	return int64(minutes) * 60, nil
}

// parseColonDuration parses "hh:mm" or "hh:mm:ss" and returns seconds.
func parseColonDuration(str string) (int64, error) {
	parts := strings.SplitN(str, ":", 3)

	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("cannot parse duration tag value: %s", str)
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("cannot parse duration tag value: %s", str)
	}

	seconds := 0
	if len(parts) == 3 {
		if len(parts[2]) < 2 {
			return 0, fmt.Errorf("cannot parse duration tag value: %s", str)
		}
		seconds, err = strconv.Atoi(parts[2][:2])
		if err != nil {
			return 0, fmt.Errorf("cannot parse duration tag value: %s", str)
		}
	}

	return int64(hours)*3600 + int64(minutes)*60 + int64(seconds), nil
}

// parseISO8601Duration parses an ISO 8601 duration like PT1H30M, P2M, PT5H12M36S.
// Uses a reference date in July 1970 for month calculation (31-day months),
// matching Java GraphHopper behavior.
func parseISO8601Duration(str string) (int64, error) {
	m := iso8601Re.FindStringSubmatch(str)
	if m == nil {
		return 0, fmt.Errorf("cannot parse duration tag value: %s", str)
	}

	years := parseGroup(m[1])
	months := parseGroup(m[2])
	days := parseGroup(m[3])
	hours := parseGroup(m[4])
	minutes := parseGroup(m[5])
	seconds := parseGroup(m[6])

	// Match Java: use a day in July 1970 which makes two identical 31-day months.
	// Months are 31 days each (matching Java DatatypeFactory behavior with this static date).
	totalSeconds := int64(years)*365*24*3600 +
		int64(months)*31*24*3600 +
		int64(days)*24*3600 +
		int64(hours)*3600 +
		int64(minutes)*60 +
		int64(seconds)

	return totalSeconds, nil
}

func parseGroup(s string) int {
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}
