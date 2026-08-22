package std

import (
	"fmt"
	"lunex/internal/runtime"
	"strings"
	"time"
)

func DatetimeModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"now": runtime.FuncVal(&runtime.Function{Name: "now", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return timeToObj(time.Now()), nil
		}}),

		"utcNow": runtime.FuncVal(&runtime.Function{Name: "utcNow", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return timeToObj(time.Now().UTC()), nil
		}}),

		"fromTimestamp": runtime.FuncVal(&runtime.Function{Name: "fromTimestamp", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			ts := int64(args[0].ToNumber())
			unit := "ms"
			if len(args) > 1 {
				unit = args[1].ToString()
			}
			var t time.Time
			switch unit {
			case "s", "sec", "seconds":
				t = time.Unix(ts, 0)
			case "us", "micro", "microseconds":
				t = time.UnixMicro(ts)
			case "ns", "nano", "nanoseconds":
				t = time.Unix(0, ts)
			default:
				t = time.UnixMilli(ts)
			}
			return timeToObj(t), nil
		}}),

		"toTimestamp": runtime.FuncVal(&runtime.Function{Name: "toTimestamp", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			unit := "ms"
			if len(args) > 1 {
				unit = args[1].ToString()
			}
			switch unit {
			case "s", "sec", "seconds":
				return runtime.NumberVal(float64(t.Unix())), nil
			case "us", "micro", "microseconds":
				return runtime.NumberVal(float64(t.UnixMicro())), nil
			case "ns", "nano", "nanoseconds":
				return runtime.NumberVal(float64(t.UnixNano())), nil
			default:
				return runtime.NumberVal(float64(t.UnixMilli())), nil
			}
		}}),

		"parse": runtime.FuncVal(&runtime.Function{Name: "parse", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			str := args[0].ToString()
			format := ""
			if len(args) > 1 {
				format = args[1].ToString()
			}
			t, err := parseDatetime(str, format)
			if err != nil {
				return runtime.Null, nil
			}
			return timeToObj(t), nil
		}}),

		"format": runtime.FuncVal(&runtime.Function{Name: "format", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(""), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.StringVal(""), nil
			}
			format := "YYYY-MM-DD HH:mm:ss"
			if len(args) > 1 {
				format = args[1].ToString()
			}
			return runtime.StringVal(formatDatetime(t, format)), nil
		}}),

		"diff": runtime.FuncVal(&runtime.Function{Name: "diff", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a, err := objToTime(args[0])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			b, err := objToTime(args[1])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			unit := "ms"
			if len(args) > 2 {
				unit = args[2].ToString()
			}
			d := a.Sub(b)
			switch unit {
			case "s", "sec", "seconds":
				return runtime.NumberVal(d.Seconds()), nil
			case "m", "min", "minutes":
				return runtime.NumberVal(d.Minutes()), nil
			case "h", "hour", "hours":
				return runtime.NumberVal(d.Hours()), nil
			case "d", "day", "days":
				return runtime.NumberVal(d.Hours() / 24), nil
			case "w", "week", "weeks":
				return runtime.NumberVal(d.Hours() / 168), nil
			case "month", "months":
				return runtime.NumberVal(d.Hours() / 730.5), nil
			case "year", "years":
				return runtime.NumberVal(d.Hours() / 8766), nil
			case "us", "micro", "microseconds":
				return runtime.NumberVal(float64(d.Microseconds())), nil
			case "ns", "nano", "nanoseconds":
				return runtime.NumberVal(float64(d.Nanoseconds())), nil
			default:
				return runtime.NumberVal(float64(d.Milliseconds())), nil
			}
		}}),

		"add": runtime.FuncVal(&runtime.Function{Name: "add", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.Null, nil
			}
			amount := args[1].ToNumber()
			unit := "ms"
			if len(args) > 2 {
				unit = args[2].ToString()
			}
			return timeToObj(addDuration(t, amount, unit)), nil
		}}),

		"subtract": runtime.FuncVal(&runtime.Function{Name: "subtract", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.Null, nil
			}
			amount := args[1].ToNumber()
			unit := "ms"
			if len(args) > 2 {
				unit = args[2].ToString()
			}
			return timeToObj(addDuration(t, -amount, unit)), nil
		}}),

		"isBefore": runtime.FuncVal(&runtime.Function{Name: "isBefore", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			a, err := objToTime(args[0])
			if err != nil {
				return runtime.False, nil
			}
			b, err := objToTime(args[1])
			if err != nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(a.Before(b)), nil
		}}),

		"isAfter": runtime.FuncVal(&runtime.Function{Name: "isAfter", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			a, err := objToTime(args[0])
			if err != nil {
				return runtime.False, nil
			}
			b, err := objToTime(args[1])
			if err != nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(a.After(b)), nil
		}}),

		"isEqual": runtime.FuncVal(&runtime.Function{Name: "isEqual", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.False, nil
			}
			a, err := objToTime(args[0])
			if err != nil {
				return runtime.False, nil
			}
			b, err := objToTime(args[1])
			if err != nil {
				return runtime.False, nil
			}
			return runtime.BoolVal(a.Equal(b)), nil
		}}),

		"compare": runtime.FuncVal(&runtime.Function{Name: "compare", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.NumberVal(0), nil
			}
			a, err := objToTime(args[0])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			b, err := objToTime(args[1])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			if a.Before(b) {
				return runtime.NumberVal(-1), nil
			}
			if a.After(b) {
				return runtime.NumberVal(1), nil
			}
			return runtime.NumberVal(0), nil
		}}),

		"startOf": runtime.FuncVal(&runtime.Function{Name: "startOf", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.Null, nil
			}
			unit := args[1].ToString()
			return timeToObj(startOf(t, unit)), nil
		}}),

		"endOf": runtime.FuncVal(&runtime.Function{Name: "endOf", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.Null, nil
			}
			unit := args[1].ToString()
			return timeToObj(endOf(t, unit)), nil
		}}),

		"weekday": runtime.FuncVal(&runtime.Function{Name: "weekday", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(float64(time.Now().Weekday())), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(t.Weekday())), nil
		}}),

		"weekdayName": runtime.FuncVal(&runtime.Function{Name: "weekdayName", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(time.Now().Weekday().String()), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(t.Weekday().String()), nil
		}}),

		"monthName": runtime.FuncVal(&runtime.Function{Name: "monthName", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.StringVal(time.Now().Month().String()), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.StringVal(""), nil
			}
			return runtime.StringVal(t.Month().String()), nil
		}}),

		"dayOfYear": runtime.FuncVal(&runtime.Function{Name: "dayOfYear", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(float64(time.Now().YearDay())), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(float64(t.YearDay())), nil
		}}),

		"weekOfYear": runtime.FuncVal(&runtime.Function{Name: "weekOfYear", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				_, w := time.Now().ISOWeek()
				return runtime.NumberVal(float64(w)), nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.NumberVal(0), nil
			}
			_, w := t.ISOWeek()
			return runtime.NumberVal(float64(w)), nil
		}}),

		"daysInMonth": runtime.FuncVal(&runtime.Function{Name: "daysInMonth", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var t time.Time
			if len(args) == 0 {
				t = time.Now()
			} else {
				var err error
				t, err = objToTime(args[0])
				if err != nil {
					return runtime.NumberVal(0), nil
				}
			}
			first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
			last := first.AddDate(0, 1, -1)
			return runtime.NumberVal(float64(last.Day())), nil
		}}),

		"isLeapYear": runtime.FuncVal(&runtime.Function{Name: "isLeapYear", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var year int
			if len(args) == 0 {
				year = time.Now().Year()
			} else if args[0].Tag == runtime.TypeNumber {
				year = int(args[0].ToNumber())
			} else {
				t, err := objToTime(args[0])
				if err != nil {
					return runtime.False, nil
				}
				year = t.Year()
			}
			isLeap := year%4 == 0 && (year%100 != 0 || year%400 == 0)
			return runtime.BoolVal(isLeap), nil
		}}),

		"isWeekend": runtime.FuncVal(&runtime.Function{Name: "isWeekend", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var t time.Time
			if len(args) == 0 {
				t = time.Now()
			} else {
				var err error
				t, err = objToTime(args[0])
				if err != nil {
					return runtime.False, nil
				}
			}
			wd := t.Weekday()
			return runtime.BoolVal(wd == time.Saturday || wd == time.Sunday), nil
		}}),

		"isValid": runtime.FuncVal(&runtime.Function{Name: "isValid", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.False, nil
			}
			_, err := objToTime(args[0])
			return runtime.BoolVal(err == nil), nil
		}}),

		"timezone": runtime.FuncVal(&runtime.Function{Name: "timezone", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			t, err := objToTime(args[0])
			if err != nil {
				return runtime.Null, nil
			}
			tzName := args[1].ToString()
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				return runtime.Null, fmt.Errorf("datetime.timezone: unknown timezone %q", tzName)
			}
			return timeToObj(t.In(loc)), nil
		}}),

		"sleep": runtime.FuncVal(&runtime.Function{Name: "sleep", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Undefined, nil
			}
			ms := args[0].ToNumber()
			unit := "ms"
			if len(args) > 1 {
				unit = args[1].ToString()
			}
			var d time.Duration
			switch unit {
			case "s", "sec", "seconds":
				d = time.Duration(ms * float64(time.Second))
			case "m", "min", "minutes":
				d = time.Duration(ms * float64(time.Minute))
			case "us", "micro":
				d = time.Duration(ms * float64(time.Microsecond))
			case "ns", "nano":
				d = time.Duration(ms * float64(time.Nanosecond))
			default:
				d = time.Duration(ms * float64(time.Millisecond))
			}
			time.Sleep(d)
			return runtime.Undefined, nil
		}}),

		"fromParts": runtime.FuncVal(&runtime.Function{Name: "fromParts", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			year, month, day := 1970, 1, 1
			hour, min, sec, nsec := 0, 0, 0, 0
			if len(args) > 0 && args[0].Tag == runtime.TypeObject {
				obj := args[0].ObjVal
				if v, ok := obj["year"]; ok {
					year = int(v.ToNumber())
				}
				if v, ok := obj["month"]; ok {
					month = int(v.ToNumber())
				}
				if v, ok := obj["day"]; ok {
					day = int(v.ToNumber())
				}
				if v, ok := obj["hour"]; ok {
					hour = int(v.ToNumber())
				}
				if v, ok := obj["minute"]; ok {
					min = int(v.ToNumber())
				}
				if v, ok := obj["second"]; ok {
					sec = int(v.ToNumber())
				}
				if v, ok := obj["ms"]; ok {
					nsec = int(v.ToNumber()) * 1000000
				}
			} else {
				if len(args) > 0 {
					year = int(args[0].ToNumber())
				}
				if len(args) > 1 {
					month = int(args[1].ToNumber())
				}
				if len(args) > 2 {
					day = int(args[2].ToNumber())
				}
				if len(args) > 3 {
					hour = int(args[3].ToNumber())
				}
				if len(args) > 4 {
					min = int(args[4].ToNumber())
				}
				if len(args) > 5 {
					sec = int(args[5].ToNumber())
				}
			}
			t := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.Local)
			return timeToObj(t), nil
		}}),
	})
}

func timeToObj(t time.Time) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"year":      runtime.NumberVal(float64(t.Year())),
		"month":     runtime.NumberVal(float64(t.Month())),
		"day":       runtime.NumberVal(float64(t.Day())),
		"hour":      runtime.NumberVal(float64(t.Hour())),
		"minute":    runtime.NumberVal(float64(t.Minute())),
		"second":    runtime.NumberVal(float64(t.Second())),
		"ms":        runtime.NumberVal(float64(t.Nanosecond() / 1e6)),
		"weekday":   runtime.NumberVal(float64(t.Weekday())),
		"timestamp": runtime.NumberVal(float64(t.UnixMilli())),
		"unix":      runtime.NumberVal(float64(t.Unix())),
		"timezone":  runtime.StringVal(t.Location().String()),
		"iso":       runtime.StringVal(t.Format(time.RFC3339)),
		"__time__":  runtime.StringVal(t.Format(time.RFC3339Nano)),
	})
}

func objToTime(v *runtime.Value) (time.Time, error) {
	if v == nil || v.IsNullish() {
		return time.Time{}, fmt.Errorf("null datetime")
	}
	if v.Tag == runtime.TypeNumber {
		return time.UnixMilli(int64(v.ToNumber())), nil
	}
	if v.Tag == runtime.TypeString {
		t, err := parseDatetime(v.StrVal, "")
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
	}
	if v.Tag == runtime.TypeObject {
		if raw, ok := v.ObjVal["__time__"]; ok {
			t, err := time.Parse(time.RFC3339Nano, raw.ToString())
			if err == nil {
				return t, nil
			}
		}
		if ts, ok := v.ObjVal["timestamp"]; ok {
			return time.UnixMilli(int64(ts.ToNumber())), nil
		}
		if iso, ok := v.ObjVal["iso"]; ok {
			t, err := time.Parse(time.RFC3339, iso.ToString())
			if err == nil {
				return t, nil
			}
		}
		year, month, day := 1970, 1, 1
		hour, min, sec := 0, 0, 0
		if y, ok := v.ObjVal["year"]; ok {
			year = int(y.ToNumber())
		}
		if m, ok := v.ObjVal["month"]; ok {
			month = int(m.ToNumber())
		}
		if d, ok := v.ObjVal["day"]; ok {
			day = int(d.ToNumber())
		}
		if h, ok := v.ObjVal["hour"]; ok {
			hour = int(h.ToNumber())
		}
		if m, ok := v.ObjVal["minute"]; ok {
			min = int(m.ToNumber())
		}
		if s, ok := v.ObjVal["second"]; ok {
			sec = int(s.ToNumber())
		}
		return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local), nil
	}
	return time.Time{}, fmt.Errorf("invalid datetime value")
}

func parseDatetime(s, format string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"01/02/2006",
		"01/02/2006 15:04:05",
		"02-01-2006",
		"02-01-2006 15:04:05",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006/01/02",
		time.RFC1123Z,
		time.RFC1123,
		time.RFC850,
		time.RFC822Z,
		time.RFC822,
	}
	if format != "" {
		goFormat := ntlToGoTimeFormat(format)
		if t, err := time.Parse(goFormat, s); err == nil {
			return t, nil
		}
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse datetime: %q", s)
}

func ntlToGoTimeFormat(format string) string {
	replacements := []struct{ from, to string }{
		{"YYYY", "2006"},
		{"YY", "06"},
		{"MMMM", "January"},
		{"MMM", "Jan"},
		{"MM", "01"},
		{"M", "1"},
		{"DD", "02"},
		{"D", "2"},
		{"HH", "15"},
		{"hh", "03"},
		{"h", "3"},
		{"mm", "04"},
		{"ss", "05"},
		{"SSS", "000"},
		{"A", "PM"},
		{"a", "pm"},
		{"ZZ", "-0700"},
		{"Z", "Z07:00"},
		{"ddd", "Mon"},
		{"dddd", "Monday"},
	}
	result := format
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.from, r.to)
	}
	return result
}

func formatDatetime(t time.Time, format string) string {
	goFormat := ntlToGoTimeFormat(format)
	return t.Format(goFormat)
}

func addDuration(t time.Time, amount float64, unit string) time.Time {
	switch unit {
	case "ms", "millisecond", "milliseconds":
		return t.Add(time.Duration(amount) * time.Millisecond)
	case "s", "sec", "second", "seconds":
		return t.Add(time.Duration(amount) * time.Second)
	case "m", "min", "minute", "minutes":
		return t.Add(time.Duration(amount) * time.Minute)
	case "h", "hour", "hours":
		return t.Add(time.Duration(amount) * time.Hour)
	case "d", "day", "days":
		return t.AddDate(0, 0, int(amount))
	case "w", "week", "weeks":
		return t.AddDate(0, 0, int(amount)*7)
	case "month", "months":
		return t.AddDate(0, int(amount), 0)
	case "year", "years":
		return t.AddDate(int(amount), 0, 0)
	default:
		return t.Add(time.Duration(amount) * time.Millisecond)
	}
}

func startOf(t time.Time, unit string) time.Time {
	switch unit {
	case "year":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "week":
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		return time.Date(t.Year(), t.Month(), t.Day()-wd+1, 0, 0, 0, 0, t.Location())
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
	case "minute":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
	case "second":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
	}
	return t
}

func endOf(t time.Time, unit string) time.Time {
	switch unit {
	case "year":
		return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
	case "month":
		first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		last := first.AddDate(0, 1, -1)
		return time.Date(last.Year(), last.Month(), last.Day(), 23, 59, 59, 999999999, t.Location())
	case "week":
		wd := int(t.Weekday())
		if wd == 0 {
			wd = 7
		}
		end := t.AddDate(0, 0, 7-wd)
		return time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, t.Location())
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
	case "hour":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 59, 59, 999999999, t.Location())
	case "minute":
		return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 59, 999999999, t.Location())
	}
	return t
}
