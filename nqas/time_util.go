package nqas

import (
	"errors"
	"regexp"
	"time"
)

var FirstDayMonday bool
var TimeFormats = []string{"1/2/2006", "1/2/2006 15:4:5", "2006-1-2 15:4:5", "2006-1-2 15:4", "2006-1-2", "1-2", "15:4:5", "15:4", "15", "15:4:5 Jan 2, 2006 MST", "2006-01-02 15:04:05.999999999 -0700 MST"}

type UserDefinedTime struct {
	time.Time
}

func NewUserDefinedTime(t time.Time) *UserDefinedTime {
	return &UserDefinedTime{t}
}

func BeginningOfMinute() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfMinute()
}

func BeginningOfHour() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfHour()
}

func BeginningOfDay() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfDay()
}

func BeginningOfWeek() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfWeek()
}

func BeginningOfMonth() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfMonth()
}

func BeginningOfQuarter() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfQuarter()
}

func BeginningOfYear() time.Time {
	return NewUserDefinedTime(time.Now()).BeginningOfYear()
}

func EndOfMinute() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfMinute()
}

func EndOfHour() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfHour()
}

func EndOfDay() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfDay()
}

func EndOfWeek() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfWeek()
}

func EndOfMonth() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfMonth()
}

func EndOfQuarter() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfQuarter()
}

func EndOfYear() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfYear()
}

func Monday() time.Time {
	return NewUserDefinedTime(time.Now()).Monday()
}

func Sunday() time.Time {
	return NewUserDefinedTime(time.Now()).Sunday()
}

func EndOfSunday() time.Time {
	return NewUserDefinedTime(time.Now()).EndOfSunday()
}

func Parse(strs ...string) (time.Time, error) {
	return NewUserDefinedTime(time.Now()).Parse(strs...)
}

func MustParse(strs ...string) time.Time {
	return NewUserDefinedTime(time.Now()).MustParse(strs...)
}

func Between(time1, time2 string) bool {
	return NewUserDefinedTime(time.Now()).Between(time1, time2)
}

func (j *UserDefinedTime) BeginningOfMinute() time.Time {
	return j.Truncate(time.Minute)
}

func (j *UserDefinedTime) BeginningOfHour() time.Time {
	return j.Truncate(time.Hour)
}

func (j *UserDefinedTime) BeginningOfDay() time.Time {
	d := time.Duration(-j.Hour()) * time.Hour
	return j.BeginningOfHour().Add(d)
}

func (j *UserDefinedTime) BeginningOfWeek() time.Time {
	t := j.BeginningOfDay()
	weekday := int(t.Weekday())
	if FirstDayMonday {
		if weekday == 0 {
			weekday = 7
		}
		weekday = weekday - 1
	}

	d := time.Duration(-weekday) * 24 * time.Hour
	return t.Add(d)
}

func (j *UserDefinedTime) BeginningOfMonth() time.Time {
	t := j.BeginningOfDay()
	d := time.Duration(-int(t.Day())+1) * 24 * time.Hour
	return t.Add(d)
}

func (j *UserDefinedTime) BeginningOfQuarter() time.Time {
	month := j.BeginningOfMonth()
	offset := (int(month.Month()) - 1) % 3
	return month.AddDate(0, -offset, 0)
}

func (j *UserDefinedTime) BeginningOfYear() time.Time {
	t := j.BeginningOfDay()
	d := time.Duration(-int(t.YearDay())+1) * 24 * time.Hour
	return t.Truncate(time.Hour).Add(d)
}

func (j *UserDefinedTime) EndOfMinute() time.Time {
	return j.BeginningOfMinute().Add(time.Minute - time.Nanosecond)
}

func (j *UserDefinedTime) EndOfHour() time.Time {
	return j.BeginningOfHour().Add(time.Hour - time.Nanosecond)
}

func (j *UserDefinedTime) EndOfDay() time.Time {
	return j.BeginningOfDay().Add(24*time.Hour - time.Nanosecond)
}

func (j *UserDefinedTime) EndOfWeek() time.Time {
	return j.BeginningOfWeek().AddDate(0, 0, 7).Add(-time.Nanosecond)
}

func (j *UserDefinedTime) EndOfMonth() time.Time {
	return j.BeginningOfMonth().AddDate(0, 1, 0).Add(-time.Nanosecond)
}

func (j *UserDefinedTime) EndOfQuarter() time.Time {
	return j.BeginningOfQuarter().AddDate(0, 3, 0).Add(-time.Nanosecond)
}

func (j *UserDefinedTime) EndOfYear() time.Time {
	return j.BeginningOfYear().AddDate(1, 0, 0).Add(-time.Nanosecond)
}

func (j *UserDefinedTime) Monday() time.Time {
	t := j.BeginningOfDay()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	d := time.Duration(-weekday+1) * 24 * time.Hour
	return t.Truncate(time.Hour).Add(d)
}

func (j *UserDefinedTime) Sunday() time.Time {
	t := j.BeginningOfDay()
	weekday := int(t.Weekday())
	if weekday == 0 {
		return t
	} else {
		d := time.Duration(7-weekday) * 24 * time.Hour
		return t.Truncate(time.Hour).Add(d)
	}
}

func (j *UserDefinedTime) EndOfSunday() time.Time {
	return j.Sunday().Add(24*time.Hour - time.Nanosecond)
}

func parseWithFormat(str string) (t time.Time, err error) {
	for _, format := range TimeFormats {
		t, err = time.Parse(format, str)
		if err == nil {
			return
		}
	}
	err = errors.New("Can't parse string as time: " + str)
	return
}

func (j *UserDefinedTime) Parse(strs ...string) (t time.Time, err error) {
	var setCurrentTime bool
	parseTime := []int{}
	currentTime := []int{j.Second(), j.Minute(), j.Hour(), j.Day(), int(j.Month()), j.Year()}
	currentLocation := j.Location()

	for _, str := range strs {
		onlyTime := regexp.MustCompile(`^\s*\d+(:\d+)*\s*$`).MatchString(str) // match 15:04:05, 15

		t, err = parseWithFormat(str)
		location := t.Location()
		if location.String() == "UTC" {
			location = currentLocation
		}

		if err == nil {
			parseTime = []int{t.Second(), t.Minute(), t.Hour(), t.Day(), int(t.Month()), t.Year()}
			onlyTime = onlyTime && (parseTime[3] == 1) && (parseTime[4] == 1)

			for i, v := range parseTime {
				// Don't reset hour, minute, second if it is a time only string
				if onlyTime && i <= 2 {
					continue
				}

				// Fill up missed information with current time
				if v == 0 {
					if setCurrentTime {
						parseTime[i] = currentTime[i]
					}
				} else {
					setCurrentTime = true
				}

				// Default day and month is 1, fill up it if missing it
				if onlyTime {
					if i == 3 || i == 4 {
						parseTime[i] = currentTime[i]
						continue
					}
				}
			}
		}

		if len(parseTime) > 0 {
			t = time.Date(parseTime[5], time.Month(parseTime[4]), parseTime[3], parseTime[2], parseTime[1], parseTime[0], 0, location)
			currentTime = []int{t.Second(), t.Minute(), t.Hour(), t.Day(), int(t.Month()), t.Year()}
		}
	}
	return
}

func (j *UserDefinedTime) MustParse(strs ...string) (t time.Time) {
	t, err := j.Parse(strs...)
	if err != nil {
		panic(err)
	}
	return t
}

func (j *UserDefinedTime) Between(time1, time2 string) bool {
	restime := j.MustParse(time1)
	restime2 := j.MustParse(time2)
	return j.After(restime) && j.Before(restime2)
}
