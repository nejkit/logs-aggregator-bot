package utils

import "time"

func GetOnlyTime(date time.Time) string {
	return date.Format(time.TimeOnly)
}

func GetOnlyDate(date time.Time) string {
	return date.Format(time.DateOnly)
}

func RoundTimeToHour(date time.Time) time.Time {
	if date.Minute() > 30 {
		date = date.Add(-time.Minute * 30)
	}
	return date.Round(time.Hour)
}

func RoundTimeToMinutes(date time.Time) time.Time {
	return date.Round(5 * time.Minute)
}

func GetInterval(startDate time.Time, endDate time.Time, step time.Duration) []time.Time {
	var result []time.Time

	for {
		if startDate.After(endDate) {
			break
		}
		result = append(result, startDate)
		startDate = startDate.Add(step)
	}

	return result
}
