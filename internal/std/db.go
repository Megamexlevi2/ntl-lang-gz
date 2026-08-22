package std

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lunex/internal/runtime"
	shared "lunex/internal/std/shared"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var globalDBEngine = &dbEngine{databases: make(map[string]*lunexDB)}

type dbEngine struct {
	databases map[string]*lunexDB
	mu        sync.Mutex
}

func (e *dbEngine) open(name string) (*lunexDB, error) {
	if name == "" {
		name = "default"
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if db, ok := e.databases[name]; ok {
		return db, nil
	}
	store, err := openStore(name)
	if err != nil {
		return nil, err
	}
	db := &lunexDB{
		name:   name,
		store:  store,
		tables: make(map[string]*lunexTable),
	}
	e.databases[name] = db
	return db, nil
}

func (e *dbEngine) drop(name string) error {
	if name == "" {
		name = "default"
	}
	e.mu.Lock()
	delete(e.databases, name)
	e.mu.Unlock()
	return dropStoreFile(name)
}

func (e *dbEngine) list() []string {
	names := listStoreFiles()
	if names == nil {
		return []string{}
	}
	sort.Strings(names)
	return names
}

type lunexDB struct {
	name   string
	store  *sqliteStore
	tables map[string]*lunexTable
	mu     sync.Mutex
}

func (db *lunexDB) table(name string) *lunexTable {
	db.mu.Lock()
	defer db.mu.Unlock()
	if t, ok := db.tables[name]; ok {
		return t
	}
	t := &lunexTable{
		name:    name,
		db:      db,
		schema:  make(map[string]*fieldDef),
		indexes: make(map[string]*tableIndex),
		loaded:  false,
	}
	db.tables[name] = t
	return t
}

func (db *lunexDB) tableNames() []string {
	names, err := db.store.listTables()
	if err != nil {
		return nil
	}
	sort.Strings(names)
	return names
}

type lunexTable struct {
	name    string
	db      *lunexDB
	rows    []dbRow
	loaded  bool
	schema  map[string]*fieldDef
	indexes map[string]*tableIndex
	watches []*tableWatch
	mu      sync.RWMutex
}

type dbRow struct {
	id  string
	doc map[string]*runtime.Value
}

type fieldDef struct {
	Type       string
	Required   bool
	Unique     bool
	DefaultVal *runtime.Value
	DefaultFn  string
	Min        float64
	Max        float64
	MinLen     int
	MaxLen     int
	MaxLenSet  bool
	Enum       []string
	Index      bool
	Primary    bool
	OnUpdate   string
	Ref        string
}

type tableIndex struct {
	fields []string
	unique bool
}

type tableWatch struct {
	id     string
	filter *runtime.Value
	fn     *runtime.Value
}

func genUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:]))
}

func (t *lunexTable) ensureLoaded() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.db.store.ensureTable(t.name); err != nil {
		return fmt.Errorf("db: could not create table '%s': %w", t.name, err)
	}
	if t.loaded {
		return nil
	}
	raw, err := t.db.store.loadAll(t.name)
	if err != nil {
		return fmt.Errorf("db: could not load table '%s': %w", t.name, err)
	}
	rows := make([]dbRow, len(raw))
	for i, r := range raw {
		rows[i] = dbRow{id: r.ID, doc: docFromNative(r.Doc)}
	}
	t.rows = rows
	t.loaded = true
	return nil
}

func rowJSON(row dbRow) ([]byte, error) {
	return json.Marshal(docToNative(row.doc))
}

func (t *lunexTable) applySchema(doc map[string]*runtime.Value, isInsert bool) (map[string]*runtime.Value, error) {
	out := make(map[string]*runtime.Value, len(doc))
	for k, v := range doc {
		out[k] = v
	}
	for field, def := range t.schema {
		if field == "_id" {
			continue
		}
		val, exists := out[field]
		if !exists || val == nil || val.IsNullish() {
			if isInsert {
				switch def.DefaultFn {
				case "$uuid":
					out[field] = runtime.StringVal(genUUID())
				case "$now":
					out[field] = runtime.NumberVal(float64(time.Now().UnixNano() / int64(time.Millisecond)))
				case "$seq":
					seq, err := t.db.store.nextSeq(t.name + "_" + field)
					if err != nil {
						return nil, fmt.Errorf("db: could not generate sequence for '%s': %w", field, err)
					}
					out[field] = runtime.NumberVal(float64(seq))
				default:
					if def.DefaultVal != nil {
						out[field] = def.DefaultVal
					} else if def.Required {
						return nil, fmt.Errorf("field '%s' is required", field)
					}
				}
			}
			continue
		}
		if def.OnUpdate == "$now" && !isInsert {
			out[field] = runtime.NumberVal(float64(time.Now().UnixNano() / int64(time.Millisecond)))
			continue
		}
		if err := validateField(field, val, def); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func validateField(name string, val *runtime.Value, def *fieldDef) error {
	if def.Type != "" && def.Type != "any" {
		typeName := shared.GetTypeName(val)
		if def.Type == "date" {
			if typeName != "number" && typeName != "string" {
				return fmt.Errorf("field '%s' must be a date (number or string), got %s", name, typeName)
			}
		} else if typeName != def.Type {
			return fmt.Errorf("field '%s' must be %s, got %s", name, def.Type, typeName)
		}
	}
	if def.Type == "number" || val.Tag == runtime.TypeNumber {
		n := val.ToNumber()
		if def.Min != 0 && n < def.Min {
			return fmt.Errorf("field '%s' must be >= %v", name, def.Min)
		}
		if def.Max != 0 && n > def.Max {
			return fmt.Errorf("field '%s' must be <= %v", name, def.Max)
		}
	}
	if def.Type == "string" || val.Tag == runtime.TypeString {
		s := val.ToString()
		if def.MinLen > 0 && len(s) < def.MinLen {
			return fmt.Errorf("field '%s' must be at least %d characters", name, def.MinLen)
		}
		if def.MaxLenSet && len(s) > def.MaxLen {
			return fmt.Errorf("field '%s' must be at most %d characters", name, def.MaxLen)
		}
	}
	if len(def.Enum) > 0 {
		s := val.ToString()
		found := false
		for _, e := range def.Enum {
			if e == s {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field '%s' must be one of: %s", name, strings.Join(def.Enum, ", "))
		}
	}
	return nil
}

func rowToValue(row dbRow) *runtime.Value {
	out := make(map[string]*runtime.Value, len(row.doc)+1)
	out["_id"] = runtime.StringVal(row.id)
	for k, v := range row.doc {
		out[k] = v
	}
	return runtime.ObjectVal(out)
}

func matchFilter(doc map[string]*runtime.Value, filter *runtime.Value) bool {
	if filter == nil || filter.IsNullish() {
		return true
	}
	if filter.Tag != runtime.TypeObject {
		return true
	}
	for key, cond := range filter.ObjVal {
		switch key {
		case "$and":
			if cond.Tag != runtime.TypeArray {
				return false
			}
			for _, f := range cond.ArrVal {
				if !matchFilter(doc, f) {
					return false
				}
			}
		case "$or":
			if cond.Tag != runtime.TypeArray {
				return false
			}
			any := false
			for _, f := range cond.ArrVal {
				if matchFilter(doc, f) {
					any = true
					break
				}
			}
			if !any {
				return false
			}
		case "$not":
			if matchFilter(doc, cond) {
				return false
			}
		case "$nor":
			if cond.Tag != runtime.TypeArray {
				return false
			}
			for _, f := range cond.ArrVal {
				if matchFilter(doc, f) {
					return false
				}
			}
		default:
			fieldVal := doc[key]
			if fieldVal == nil {
				fieldVal = runtime.Undefined
			}

			if cond != nil && cond.Tag == runtime.TypeObject {
				for op, opVal := range cond.ObjVal {
					if !matchOp(fieldVal, op, opVal) {
						return false
					}
				}
			} else {
				if !fieldVal.StrictEquals(cond) {
					return false
				}
			}
		}
	}
	return true
}

func matchRowFilter(row dbRow, filter *runtime.Value) bool {
	if filter == nil || filter.IsNullish() {
		return true
	}
	if filter.Tag != runtime.TypeObject {
		return true
	}
	doc := make(map[string]*runtime.Value, len(row.doc)+1)
	doc["_id"] = runtime.StringVal(row.id)
	for k, v := range row.doc {
		doc[k] = v
	}
	return matchFilter(doc, filter)
}

func matchOp(field *runtime.Value, op string, val *runtime.Value) bool {
	if field == nil {
		field = runtime.Undefined
	}
	switch op {
	case "$eq":
		return field.StrictEquals(val)
	case "$ne":
		return !field.StrictEquals(val)
	case "$gt":
		return field.ToNumber() > val.ToNumber()
	case "$gte":
		return field.ToNumber() >= val.ToNumber()
	case "$lt":
		return field.ToNumber() < val.ToNumber()
	case "$lte":
		return field.ToNumber() <= val.ToNumber()
	case "$in":
		if val == nil || val.Tag != runtime.TypeArray {
			return false
		}
		for _, item := range val.ArrVal {
			if field.StrictEquals(item) {
				return true
			}
		}
		return false
	case "$nin":
		if val == nil || val.Tag != runtime.TypeArray {
			return true
		}
		for _, item := range val.ArrVal {
			if field.StrictEquals(item) {
				return false
			}
		}
		return true
	case "$like":
		return matchLike(field.ToString(), val.ToString(), false)
	case "$ilike":
		return matchLike(field.ToString(), val.ToString(), true)
	case "$regex":
		re, err := regexp.Compile(val.ToString())
		if err != nil {
			return false
		}
		return re.MatchString(field.ToString())
	case "$exists":
		exists := field.Tag != runtime.TypeUndefined && field.Tag != runtime.TypeNull
		if val != nil && val.Tag == runtime.TypeBool {
			return exists == val.BoolVal
		}
		return exists
	case "$between":
		if val == nil || val.Tag != runtime.TypeArray || len(val.ArrVal) < 2 {
			return false
		}
		n := field.ToNumber()
		return n >= val.ArrVal[0].ToNumber() && n <= val.ArrVal[1].ToNumber()
	case "$contains":
		if field.Tag == runtime.TypeArray {
			for _, item := range field.ArrVal {
				if item.StrictEquals(val) {
					return true
				}
			}
			return false
		}
		return strings.Contains(field.ToString(), val.ToString())
	case "$size":
		if field.Tag == runtime.TypeArray {
			return float64(len(field.ArrVal)) == val.ToNumber()
		}
		if field.Tag == runtime.TypeString {
			return float64(len(field.StrVal)) == val.ToNumber()
		}
		return false
	case "$type":
		return shared.GetTypeName(field) == val.ToString()
	case "$startsWith":
		return strings.HasPrefix(field.ToString(), val.ToString())
	case "$endsWith":
		return strings.HasSuffix(field.ToString(), val.ToString())
	}
	return true
}

func matchLike(s, pattern string, caseInsensitive bool) bool {
	if caseInsensitive {
		s = strings.ToLower(s)
		pattern = strings.ToLower(pattern)
	}
	re := "^"
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '%':
			re += ".*"
		case '_':
			re += "."
		case '.', '+', '*', '?', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			re += "\\" + string(pattern[i])
		default:
			re += string(pattern[i])
		}
	}
	re += "$"
	matched, _ := regexp.MatchString(re, s)
	return matched
}

func projectRow(row dbRow, fields []string) *runtime.Value {
	if len(fields) == 0 {
		return rowToValue(row)
	}
	out := make(map[string]*runtime.Value, len(fields))
	src := make(map[string]*runtime.Value, len(row.doc)+1)
	src["_id"] = runtime.StringVal(row.id)
	for k, v := range row.doc {
		src[k] = v
	}
	for _, f := range fields {
		if v, ok := src[f]; ok {
			out[f] = v
		}
	}
	return runtime.ObjectVal(out)
}

func (t *lunexTable) execQuery(filter *runtime.Value, proj []string, sorts []sortEntry, limitN, offsetN int) []*runtime.Value {
	if err := t.ensureLoaded(); err != nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	var matched []dbRow
	for _, row := range t.rows {
		if matchRowFilter(row, filter) {
			matched = append(matched, row)
		}
	}
	if len(sorts) > 0 {
		sort.SliceStable(matched, func(i, j int) bool {
			for _, s := range sorts {
				a := matched[i].doc[s.field]
				b := matched[j].doc[s.field]
				if s.field == "_id" {
					a = runtime.StringVal(matched[i].id)
					b = runtime.StringVal(matched[j].id)
				}
				if a == nil {
					a = runtime.Undefined
				}
				if b == nil {
					b = runtime.Undefined
				}
				var cmp int
				if a.Tag == runtime.TypeNumber && b.Tag == runtime.TypeNumber {
					if a.ToNumber() < b.ToNumber() {
						cmp = -1
					} else if a.ToNumber() > b.ToNumber() {
						cmp = 1
					}
				} else {
					cmp = strings.Compare(a.ToString(), b.ToString())
				}
				if cmp != 0 {
					if s.desc {
						return cmp > 0
					}
					return cmp < 0
				}
			}
			return false
		})
	}
	if offsetN > 0 {
		if offsetN >= len(matched) {
			return nil
		}
		matched = matched[offsetN:]
	}
	if limitN > 0 && limitN < len(matched) {
		matched = matched[:limitN]
	}
	out := make([]*runtime.Value, len(matched))
	for i, row := range matched {
		out[i] = projectRow(row, proj)
	}
	return out
}

func (t *lunexTable) insertDoc(doc map[string]*runtime.Value) (*runtime.Value, error) {
	if err := t.ensureLoaded(); err != nil {
		return nil, err
	}
	out, err := t.applySchema(doc, true)
	if err != nil {
		return nil, err
	}
	id, hasID := out["_id"]
	if !hasID || id == nil || id.IsNullish() {
		id = runtime.StringVal(genUUID())
	}
	idStr := id.ToString()
	delete(out, "_id")

	if err := t.checkUnique(idStr, out, false); err != nil {
		return nil, err
	}

	row := dbRow{id: idStr, doc: out}
	payload, err := rowJSON(row)
	if err != nil {
		return nil, fmt.Errorf("db: could not serialize document: %w", err)
	}
	if err := t.db.store.insertRow(t.name, idStr, payload); err != nil {
		return nil, fmt.Errorf("db: insert failed: %w", err)
	}

	t.mu.Lock()
	t.rows = append(t.rows, row)
	t.mu.Unlock()

	t.notifyWatches("insert", row)
	result := rowToValue(row)
	return result, nil
}

func (t *lunexTable) checkUnique(excludeID string, doc map[string]*runtime.Value, isUpdate bool) error {
	var uniqueFields []string
	for field, def := range t.schema {
		if def.Unique {
			uniqueFields = append(uniqueFields, field)
		}
	}
	if len(uniqueFields) == 0 {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, field := range uniqueFields {
		val, ok := doc[field]
		if !ok || val == nil || val.IsNullish() {
			continue
		}
		for _, row := range t.rows {
			if row.id == excludeID {
				continue
			}
			if existing, ok := row.doc[field]; ok && existing.StrictEquals(val) {
				return fmt.Errorf("field '%s' must be unique, value already exists", field)
			}
		}
	}
	return nil
}

func (t *lunexTable) updateRows(filter *runtime.Value, changes map[string]*runtime.Value, onlyFirst bool) int {
	if err := t.ensureLoaded(); err != nil {
		return 0
	}
	t.mu.Lock()
	var toPersist []dbRow
	n := 0
	for i, row := range t.rows {
		if matchRowFilter(row, filter) {
			for k, v := range changes {
				t.rows[i].doc[k] = v
			}
			if def, ok := t.schema["updatedAt"]; ok && def.OnUpdate == "$now" {
				t.rows[i].doc["updatedAt"] = runtime.NumberVal(float64(time.Now().UnixNano() / int64(time.Millisecond)))
			}
			toPersist = append(toPersist, t.rows[i])
			n++
			if onlyFirst {
				break
			}
		}
	}
	t.mu.Unlock()

	for _, row := range toPersist {
		payload, err := rowJSON(row)
		if err != nil {
			continue
		}
		_ = t.db.store.updateRow(t.name, row.id, payload)
		t.notifyWatches("update", row)
	}
	return n
}

func (t *lunexTable) deleteRows(filter *runtime.Value, onlyFirst bool) int {
	if err := t.ensureLoaded(); err != nil {
		return 0
	}
	t.mu.Lock()
	var kept []dbRow
	var removed []dbRow
	n := 0
	for _, row := range t.rows {
		if (onlyFirst == false || n == 0) && matchRowFilter(row, filter) {
			removed = append(removed, row)
			n++
			continue
		}
		kept = append(kept, row)
	}
	t.rows = kept
	t.mu.Unlock()

	for _, row := range removed {
		_ = t.db.store.deleteRow(t.name, row.id)
		t.notifyWatches("delete", row)
	}
	return n
}

func (t *lunexTable) notifyWatches(event string, row dbRow) {
	t.mu.RLock()
	watches := make([]*tableWatch, len(t.watches))
	copy(watches, t.watches)
	t.mu.RUnlock()
	if len(watches) == 0 {
		return
	}
	for _, w := range watches {
		if matchRowFilter(row, w.filter) {
			if runtime.CallFunction != nil {
				runtime.CallFunction(w.fn, []*runtime.Value{
					runtime.ObjectVal(map[string]*runtime.Value{
						"type": runtime.StringVal(event),
						"doc":  rowToValue(row),
					}),
				})
			}
		}
	}
}

func (t *lunexTable) aggregate(pipeline *runtime.Value) []*runtime.Value {
	if err := t.ensureLoaded(); err != nil {
		return nil
	}
	t.mu.RLock()
	current := make([]dbRow, len(t.rows))
	copy(current, t.rows)
	t.mu.RUnlock()
	if pipeline == nil || pipeline.Tag != runtime.TypeArray {
		return nil
	}
	var rows []*runtime.Value
	for _, r := range current {
		rows = append(rows, rowToValue(r))
	}
	for _, stage := range pipeline.ArrVal {
		if stage == nil || stage.Tag != runtime.TypeObject {
			continue
		}
		for op, conf := range stage.ObjVal {
			switch op {
			case "$match":
				var next []*runtime.Value
				for _, r := range rows {
					if matchFilter(r.ObjVal, conf) {
						next = append(next, r)
					}
				}
				rows = next
			case "$sort":
				if conf == nil || conf.Tag != runtime.TypeObject {
					break
				}
				type sf struct {
					field string
					dir   float64
				}
				var fields []sf
				for f, d := range conf.ObjVal {
					fields = append(fields, sf{f, d.ToNumber()})
				}
				sort.SliceStable(rows, func(i, j int) bool {
					for _, s := range fields {
						a := rows[i].ObjVal[s.field]
						b := rows[j].ObjVal[s.field]
						if a == nil {
							a = runtime.Undefined
						}
						if b == nil {
							b = runtime.Undefined
						}
						var cmp int
						if a.Tag == runtime.TypeNumber && b.Tag == runtime.TypeNumber {
							if a.ToNumber() < b.ToNumber() {
								cmp = -1
							} else if a.ToNumber() > b.ToNumber() {
								cmp = 1
							}
						} else {
							cmp = strings.Compare(a.ToString(), b.ToString())
						}
						if cmp != 0 {
							if s.dir < 0 {
								return cmp > 0
							}
							return cmp < 0
						}
					}
					return false
				})
			case "$limit":
				n := int(conf.ToNumber())
				if n > 0 && n < len(rows) {
					rows = rows[:n]
				}
			case "$skip":
				n := int(conf.ToNumber())
				if n > 0 {
					if n >= len(rows) {
						rows = nil
					} else {
						rows = rows[n:]
					}
				}
			case "$project":
				if conf == nil || conf.Tag != runtime.TypeObject {
					break
				}
				var include []string
				for f, v := range conf.ObjVal {
					if v != nil && v.ToNumber() != 0 && v.Tag != runtime.TypeBool || (v.Tag == runtime.TypeBool && v.BoolVal) {
						include = append(include, f)
					}
				}
				if len(include) > 0 {
					for idx, r := range rows {
						out := make(map[string]*runtime.Value, len(include))
						for _, f := range include {
							if v, ok := r.ObjVal[f]; ok {
								out[f] = v
							}
						}
						rows[idx] = runtime.ObjectVal(out)
					}
				}
			case "$group":
				if conf == nil || conf.Tag != runtime.TypeObject {
					break
				}
				byField := ""
				if by, ok := conf.ObjVal["by"]; ok {
					byField = by.ToString()
				}
				groups := make(map[string][]*runtime.Value)
				groupOrder := []string{}
				for _, r := range rows {
					key := ""
					if byField != "" {
						if v, ok := r.ObjVal[byField]; ok {
							key = v.ToString()
						}
					}
					if _, exists := groups[key]; !exists {
						groupOrder = append(groupOrder, key)
					}
					groups[key] = append(groups[key], r)
				}
				var next []*runtime.Value
				for _, key := range groupOrder {
					grp := groups[key]
					out := make(map[string]*runtime.Value)
					if byField != "" {
						out[byField] = runtime.StringVal(key)
					}
					for aggField, aggDef := range conf.ObjVal {
						if aggField == "by" {
							continue
						}
						if aggDef.Tag == runtime.TypeBool && aggDef.BoolVal {
							out[aggField] = runtime.NumberVal(float64(len(grp)))
							continue
						}
						if aggDef.Tag != runtime.TypeObject {
							continue
						}
						for aggOp, aggTarget := range aggDef.ObjVal {
							targetField := aggTarget.ToString()
							switch aggOp {
							case "$count":
								out[aggField] = runtime.NumberVal(float64(len(grp)))
							case "$sum":
								sum := 0.0
								for _, r := range grp {
									if v, ok := r.ObjVal[targetField]; ok {
										sum += v.ToNumber()
									}
								}
								out[aggField] = runtime.NumberVal(sum)
							case "$avg":
								sum := 0.0
								for _, r := range grp {
									if v, ok := r.ObjVal[targetField]; ok {
										sum += v.ToNumber()
									}
								}
								if len(grp) > 0 {
									out[aggField] = runtime.NumberVal(sum / float64(len(grp)))
								} else {
									out[aggField] = runtime.NumberVal(0)
								}
							case "$min":
								min := math.MaxFloat64
								for _, r := range grp {
									if v, ok := r.ObjVal[targetField]; ok {
										if v.ToNumber() < min {
											min = v.ToNumber()
										}
									}
								}
								if min == math.MaxFloat64 {
									min = 0
								}
								out[aggField] = runtime.NumberVal(min)
							case "$max":
								max := -math.MaxFloat64
								for _, r := range grp {
									if v, ok := r.ObjVal[targetField]; ok {
										if v.ToNumber() > max {
											max = v.ToNumber()
										}
									}
								}
								if max == -math.MaxFloat64 {
									max = 0
								}
								out[aggField] = runtime.NumberVal(max)
							case "$first":
								if len(grp) > 0 {
									if v, ok := grp[0].ObjVal[targetField]; ok {
										out[aggField] = v
									}
								}
							case "$last":
								if len(grp) > 0 {
									if v, ok := grp[len(grp)-1].ObjVal[targetField]; ok {
										out[aggField] = v
									}
								}
							case "$push":
								arr := make([]*runtime.Value, 0, len(grp))
								for _, r := range grp {
									if v, ok := r.ObjVal[targetField]; ok {
										arr = append(arr, v)
									}
								}
								out[aggField] = runtime.ArrayVal(arr)
							case "$addToSet":
								seen := make(map[string]bool)
								var arr []*runtime.Value
								for _, r := range grp {
									if v, ok := r.ObjVal[targetField]; ok {
										k := v.ToString()
										if !seen[k] {
											seen[k] = true
											arr = append(arr, v)
										}
									}
								}
								out[aggField] = runtime.ArrayVal(arr)
							}
						}
					}
					next = append(next, runtime.ObjectVal(out))
				}
				rows = next
			case "$unwind":
				field := conf.ToString()
				var next []*runtime.Value
				for _, r := range rows {
					arr, ok := r.ObjVal[field]
					if !ok || arr.Tag != runtime.TypeArray {
						next = append(next, r)
						continue
					}
					for _, item := range arr.ArrVal {
						clone := make(map[string]*runtime.Value, len(r.ObjVal))
						for k, v := range r.ObjVal {
							clone[k] = v
						}
						clone[field] = item
						next = append(next, runtime.ObjectVal(clone))
					}
				}
				rows = next
			case "$count":
				name := conf.ToString()
				if name == "" {
					name = "count"
				}
				rows = []*runtime.Value{runtime.ObjectVal(map[string]*runtime.Value{
					name: runtime.NumberVal(float64(len(rows))),
				})}
			}
		}
	}
	return rows
}

func (t *lunexTable) search(text string, fields []string) []*runtime.Value {
	if err := t.ensureLoaded(); err != nil {
		return nil
	}
	text = strings.ToLower(text)
	t.mu.RLock()
	defer t.mu.RUnlock()
	var out []*runtime.Value
	for _, row := range t.rows {
		searchFields := fields
		if len(searchFields) == 0 {
			for k := range row.doc {
				searchFields = append(searchFields, k)
			}
		}
		for _, f := range searchFields {
			if v, ok := row.doc[f]; ok {
				if strings.Contains(strings.ToLower(v.ToString()), text) {
					out = append(out, rowToValue(row))
					break
				}
			}
		}
	}
	return out
}

type sortEntry struct {
	field string
	desc  bool
}

func tableObject(t *lunexTable) *runtime.Value {
	makeQB := func(filter *runtime.Value) *runtime.Value {
		return newQueryBuilder(t, filter)
	}

	obj := runtime.ObjectVal(map[string]*runtime.Value{
		"name": runtime.StringVal(t.name),

		"schema": runtime.FuncVal(&runtime.Function{Name: "schema", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 && args[0] != nil && args[0].Tag == runtime.TypeObject {
				t.mu.Lock()
				for field, defVal := range args[0].ObjVal {
					fd := &fieldDef{}
					if defVal.Tag == runtime.TypeString {
						fd.Type = defVal.StrVal
					} else if defVal.Tag == runtime.TypeObject {
						if v, ok := defVal.ObjVal["type"]; ok {
							fd.Type = v.ToString()
						}
						if v, ok := defVal.ObjVal["required"]; ok {
							fd.Required = v.BoolVal
						}
						if v, ok := defVal.ObjVal["unique"]; ok {
							fd.Unique = v.BoolVal
						}
						if v, ok := defVal.ObjVal["index"]; ok {
							fd.Index = v.BoolVal
						}
						if v, ok := defVal.ObjVal["primary"]; ok {
							fd.Primary = v.BoolVal
						}
						if v, ok := defVal.ObjVal["min"]; ok {
							fd.Min = v.ToNumber()
						}
						if v, ok := defVal.ObjVal["max"]; ok {
							fd.Max = v.ToNumber()
						}
						if v, ok := defVal.ObjVal["minLength"]; ok {
							fd.MinLen = int(v.ToNumber())
						}
						if v, ok := defVal.ObjVal["maxLength"]; ok {
							fd.MaxLen = int(v.ToNumber())
							fd.MaxLenSet = true
						}
						if v, ok := defVal.ObjVal["ref"]; ok {
							fd.Ref = v.ToString()
						}
						if v, ok := defVal.ObjVal["onUpdate"]; ok {
							fd.OnUpdate = "$" + strings.TrimPrefix(v.ToString(), "$")
						}
						if v, ok := defVal.ObjVal["default"]; ok {
							if v.Tag == runtime.TypeString {
								s := v.StrVal
								if s == "uuid" || s == "$uuid" {
									fd.DefaultFn = "$uuid"
								} else if s == "now" || s == "$now" {
									fd.DefaultFn = "$now"
								} else if s == "seq" || s == "$seq" {
									fd.DefaultFn = "$seq"
								} else {
									fd.DefaultVal = v
								}
							} else {
								fd.DefaultVal = v
							}
						}
						if v, ok := defVal.ObjVal["enum"]; ok && v.Tag == runtime.TypeArray {
							for _, item := range v.ArrVal {
								fd.Enum = append(fd.Enum, item.ToString())
							}
						}
					}
					t.schema[field] = fd
				}
				t.mu.Unlock()
				for field, fd := range t.schema {
					if fd.Index || fd.Unique {
						_ = t.db.store.createIndex(t.name, field, []string{field}, fd.Unique)
					}
				}
			}
			return tableObject(t), nil
		}}),

		"index": runtime.FuncVal(&runtime.Function{Name: "index", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var fields []string
			if len(args) > 0 {
				if args[0].Tag == runtime.TypeArray {
					for _, f := range args[0].ArrVal {
						fields = append(fields, f.ToString())
					}
				} else {
					fields = []string{args[0].ToString()}
				}
			}
			if len(fields) == 0 {
				return tableObject(t), nil
			}
			unique := false
			if len(args) > 1 && args[1].Tag == runtime.TypeObject {
				if v, ok := args[1].ObjVal["unique"]; ok {
					unique = v.BoolVal
				}
			}
			name := strings.Join(fields, "_")
			t.mu.Lock()
			t.indexes[name] = &tableIndex{fields: fields, unique: unique}
			t.mu.Unlock()
			if err := t.db.store.createIndex(t.name, name, fields, unique); err != nil {
				return tableObject(t), fmt.Errorf("db: could not create index '%s': %w", name, err)
			}
			return tableObject(t), nil
		}}),

		"insert": runtime.FuncVal(&runtime.Function{Name: "insert", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("insert: expected object")
			}
			doc := make(map[string]*runtime.Value, len(args[0].ObjVal))
			for k, v := range args[0].ObjVal {
				doc[k] = v
			}
			return t.insertDoc(doc)
		}}),

		"insertMany": runtime.FuncVal(&runtime.Function{Name: "insertMany", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeArray {
				return runtime.ArrayVal(nil), nil
			}
			var out []*runtime.Value
			for _, item := range args[0].ArrVal {
				if item == nil || item.Tag != runtime.TypeObject {
					continue
				}
				doc := make(map[string]*runtime.Value, len(item.ObjVal))
				for k, v := range item.ObjVal {
					doc[k] = v
				}
				r, err := t.insertDoc(doc)
				if err != nil {
					return nil, err
				}
				out = append(out, r)
			}
			return runtime.ArrayVal(out), nil
		}}),

		"upsert": runtime.FuncVal(&runtime.Function{Name: "upsert", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, nil
			}
			rows := t.execQuery(args[0], nil, nil, 1, 0)
			if len(rows) > 0 {
				changes := make(map[string]*runtime.Value)
				if args[1].Tag == runtime.TypeObject {
					for k, v := range args[1].ObjVal {
						changes[k] = v
					}
				}
				t.updateRows(args[0], changes, true)
				rows = t.execQuery(args[0], nil, nil, 1, 0)
				if len(rows) > 0 {
					return rows[0], nil
				}
				return runtime.Null, nil
			}
			doc := make(map[string]*runtime.Value)
			if args[1].Tag == runtime.TypeObject {
				for k, v := range args[1].ObjVal {
					doc[k] = v
				}
			}
			return t.insertDoc(doc)
		}}),

		"find": runtime.FuncVal(&runtime.Function{Name: "find", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			var proj []string
			var sorts []sortEntry
			limitN, offsetN := 0, 0
			if len(args) > 1 && args[1] != nil && args[1].Tag == runtime.TypeObject {
				opts := args[1].ObjVal
				if v, ok := opts["select"]; ok && v.Tag == runtime.TypeArray {
					for _, f := range v.ArrVal {
						proj = append(proj, f.ToString())
					}
				}
				if v, ok := opts["sort"]; ok && v.Tag == runtime.TypeObject {
					for f, d := range v.ObjVal {
						sorts = append(sorts, sortEntry{f, d.ToNumber() < 0})
					}
				}
				if v, ok := opts["orderBy"]; ok {
					sorts = append(sorts, sortEntry{v.ToString(), false})
				}
				if v, ok := opts["limit"]; ok {
					limitN = int(v.ToNumber())
				}
				if v, ok := opts["offset"]; ok {
					offsetN = int(v.ToNumber())
				}
				if v, ok := opts["skip"]; ok {
					offsetN = int(v.ToNumber())
				}
			}
			return runtime.ArrayVal(t.execQuery(filter, proj, sorts, limitN, offsetN)), nil
		}}),

		"findOne": runtime.FuncVal(&runtime.Function{Name: "findOne", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			rows := t.execQuery(filter, nil, nil, 1, 0)
			if len(rows) == 0 {
				return runtime.Null, nil
			}
			return rows[0], nil
		}}),

		"findById": runtime.FuncVal(&runtime.Function{Name: "findById", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.Null, err
			}
			id := args[0].ToString()
			t.mu.RLock()
			defer t.mu.RUnlock()
			for _, row := range t.rows {
				if row.id == id {
					return rowToValue(row), nil
				}
			}
			return runtime.Null, nil
		}}),

		"update": runtime.FuncVal(&runtime.Function{Name: "update", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[1].Tag != runtime.TypeObject {
				return runtime.NumberVal(0), nil
			}
			changes := make(map[string]*runtime.Value)
			if set, ok := args[1].ObjVal["$set"]; ok && set.Tag == runtime.TypeObject {
				for k, v := range set.ObjVal {
					changes[k] = v
				}
			} else {
				for k, v := range args[1].ObjVal {
					changes[k] = v
				}
			}
			return runtime.NumberVal(float64(t.updateRows(args[0], changes, false))), nil
		}}),

		"updateOne": runtime.FuncVal(&runtime.Function{Name: "updateOne", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[1].Tag != runtime.TypeObject {
				return runtime.NumberVal(0), nil
			}
			changes := make(map[string]*runtime.Value)
			if set, ok := args[1].ObjVal["$set"]; ok && set.Tag == runtime.TypeObject {
				for k, v := range set.ObjVal {
					changes[k] = v
				}
			} else {
				for k, v := range args[1].ObjVal {
					changes[k] = v
				}
			}
			return runtime.NumberVal(float64(t.updateRows(args[0], changes, true))), nil
		}}),

		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			return runtime.NumberVal(float64(t.deleteRows(filter, false))), nil
		}}),

		"deleteOne": runtime.FuncVal(&runtime.Function{Name: "deleteOne", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			return runtime.NumberVal(float64(t.deleteRows(filter, true))), nil
		}}),

		"deleteById": runtime.FuncVal(&runtime.Function{Name: "deleteById", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			id := args[0].ToString()
			return runtime.NumberVal(float64(t.deleteRows(runtime.ObjectVal(map[string]*runtime.Value{"_id": runtime.StringVal(id)}), true))), nil
		}}),

		"count": runtime.FuncVal(&runtime.Function{Name: "count", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.NumberVal(0), err
			}
			t.mu.RLock()
			defer t.mu.RUnlock()
			n := 0
			for _, row := range t.rows {
				if matchRowFilter(row, filter) {
					n++
				}
			}
			return runtime.NumberVal(float64(n)), nil
		}}),

		"exists": runtime.FuncVal(&runtime.Function{Name: "exists", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			rows := t.execQuery(filter, nil, nil, 1, 0)
			return runtime.BoolVal(len(rows) > 0), nil
		}}),

		"distinct": runtime.FuncVal(&runtime.Function{Name: "distinct", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			field := args[0].ToString()
			var filter *runtime.Value
			if len(args) > 1 {
				filter = args[1]
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.ArrayVal(nil), err
			}
			t.mu.RLock()
			defer t.mu.RUnlock()
			seen := make(map[string]bool)
			var out []*runtime.Value
			for _, row := range t.rows {
				if !matchRowFilter(row, filter) {
					continue
				}
				var v *runtime.Value
				if field == "_id" {
					v = runtime.StringVal(row.id)
				} else {
					v = row.doc[field]
				}
				if v == nil {
					continue
				}
				k := v.ToString()
				if !seen[k] {
					seen[k] = true
					out = append(out, v)
				}
			}
			return runtime.ArrayVal(out), nil
		}}),

		"where": runtime.FuncVal(&runtime.Function{Name: "where", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			if len(args) > 0 {
				filter = args[0]
			}
			return makeQB(filter), nil
		}}),

		"select": runtime.FuncVal(&runtime.Function{Name: "select", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return makeQB(nil), nil
		}}),

		"orderBy": runtime.FuncVal(&runtime.Function{Name: "orderBy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return makeQB(nil), nil
		}}),

		"limit": runtime.FuncVal(&runtime.Function{Name: "limit", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return makeQB(nil), nil
		}}),

		"aggregate": runtime.FuncVal(&runtime.Function{Name: "aggregate", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			return runtime.ArrayVal(t.aggregate(args[0])), nil
		}}),

		"sum": runtime.FuncVal(&runtime.Function{Name: "sum", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			field := args[0].ToString()
			var filter *runtime.Value
			if len(args) > 1 {
				filter = args[1]
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.NumberVal(0), err
			}
			t.mu.RLock()
			defer t.mu.RUnlock()
			sum := 0.0
			for _, row := range t.rows {
				if !matchRowFilter(row, filter) {
					continue
				}
				if v, ok := row.doc[field]; ok {
					sum += v.ToNumber()
				}
			}
			return runtime.NumberVal(sum), nil
		}}),

		"avg": runtime.FuncVal(&runtime.Function{Name: "avg", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.NumberVal(0), nil
			}
			field := args[0].ToString()
			var filter *runtime.Value
			if len(args) > 1 {
				filter = args[1]
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.NumberVal(0), err
			}
			t.mu.RLock()
			defer t.mu.RUnlock()
			sum, n := 0.0, 0
			for _, row := range t.rows {
				if !matchRowFilter(row, filter) {
					continue
				}
				if v, ok := row.doc[field]; ok {
					sum += v.ToNumber()
					n++
				}
			}
			if n == 0 {
				return runtime.NumberVal(0), nil
			}
			return runtime.NumberVal(sum / float64(n)), nil
		}}),

		"min": runtime.FuncVal(&runtime.Function{Name: "min", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			field := args[0].ToString()
			var filter *runtime.Value
			if len(args) > 1 {
				filter = args[1]
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.Null, err
			}
			t.mu.RLock()
			defer t.mu.RUnlock()
			min := math.MaxFloat64
			for _, row := range t.rows {
				if !matchRowFilter(row, filter) {
					continue
				}
				if v, ok := row.doc[field]; ok {
					if v.ToNumber() < min {
						min = v.ToNumber()
					}
				}
			}
			if min == math.MaxFloat64 {
				return runtime.Null, nil
			}
			return runtime.NumberVal(min), nil
		}}),

		"max": runtime.FuncVal(&runtime.Function{Name: "max", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, nil
			}
			field := args[0].ToString()
			var filter *runtime.Value
			if len(args) > 1 {
				filter = args[1]
			}
			if err := t.ensureLoaded(); err != nil {
				return runtime.Null, err
			}
			t.mu.RLock()
			defer t.mu.RUnlock()
			max := -math.MaxFloat64
			for _, row := range t.rows {
				if !matchRowFilter(row, filter) {
					continue
				}
				if v, ok := row.doc[field]; ok {
					if v.ToNumber() > max {
						max = v.ToNumber()
					}
				}
			}
			if max == -math.MaxFloat64 {
				return runtime.Null, nil
			}
			return runtime.NumberVal(max), nil
		}}),

		"join": runtime.FuncVal(&runtime.Function{Name: "join", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.ArrayVal(nil), nil
			}
			_, ok := args[0].ObjVal["__lunex_table__"]
			if !ok {
				return runtime.ArrayVal(nil), fmt.Errorf("join: expected table object")
			}
			return runtime.ArrayVal(nil), nil
		}}),

		"search": runtime.FuncVal(&runtime.Function{Name: "search", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.ArrayVal(nil), nil
			}
			text := args[0].ToString()
			var fields []string
			if len(args) > 1 && args[1].Tag == runtime.TypeArray {
				for _, f := range args[1].ArrVal {
					fields = append(fields, f.ToString())
				}
			}
			return runtime.ArrayVal(t.search(text, fields)), nil
		}}),

		"watch": runtime.FuncVal(&runtime.Function{Name: "watch", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			var filter *runtime.Value
			var fn *runtime.Value
			if len(args) >= 2 {
				filter = args[0]
				fn = args[1]
			} else if len(args) == 1 {
				fn = args[0]
			}
			if fn == nil || fn.Tag != runtime.TypeFunction {
				return runtime.Undefined, nil
			}
			watchID := genUUID()
			t.mu.Lock()
			t.watches = append(t.watches, &tableWatch{id: watchID, filter: filter, fn: fn})
			t.mu.Unlock()
			return runtime.FuncVal(&runtime.Function{Name: "unwatch", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
				t.mu.Lock()
				for i, w := range t.watches {
					if w.id == watchID {
						t.watches = append(t.watches[:i], t.watches[i+1:]...)
						break
					}
				}
				t.mu.Unlock()
				return runtime.Undefined, nil
			}}), nil
		}}),

		"clear": runtime.FuncVal(&runtime.Function{Name: "clear", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if err := t.db.store.ensureTable(t.name); err != nil {
				return runtime.Undefined, fmt.Errorf("db: could not create table '%s': %w", t.name, err)
			}
			if err := t.db.store.clearRows(t.name); err != nil {
				return runtime.Undefined, fmt.Errorf("db: could not clear table '%s': %w", t.name, err)
			}
			t.mu.Lock()
			t.rows = nil
			t.loaded = true
			t.mu.Unlock()
			return runtime.Undefined, nil
		}}),

		"drop": runtime.FuncVal(&runtime.Function{Name: "drop", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if err := t.db.store.dropTable(t.name); err != nil {
				return runtime.Undefined, fmt.Errorf("db: could not drop table '%s': %w", t.name, err)
			}
			t.mu.Lock()
			t.rows = nil
			t.loaded = false
			t.schema = make(map[string]*fieldDef)
			t.indexes = make(map[string]*tableIndex)
			t.watches = nil
			t.mu.Unlock()
			return runtime.Undefined, nil
		}}),

		"dump": runtime.FuncVal(&runtime.Function{Name: "dump", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			rows := t.execQuery(nil, nil, nil, 0, 0)
			return runtime.ArrayVal(rows), nil
		}}),

		"indexes": runtime.FuncVal(&runtime.Function{Name: "indexes", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			t.mu.RLock()
			defer t.mu.RUnlock()
			var out []*runtime.Value
			for name, idx := range t.indexes {
				fields := make([]*runtime.Value, len(idx.fields))
				for i, f := range idx.fields {
					fields[i] = runtime.StringVal(f)
				}
				out = append(out, runtime.ObjectVal(map[string]*runtime.Value{
					"name":   runtime.StringVal(name),
					"fields": runtime.ArrayVal(fields),
					"unique": runtime.BoolVal(idx.unique),
				}))
			}
			return runtime.ArrayVal(out), nil
		}}),

		"__lunex_table__": runtime.BoolVal(true),
	})
	schemaFn := obj.ObjVal["schema"]
	obj.ObjVal["define"] = schemaFn
	return obj
}

func newQueryBuilder(t *lunexTable, filter *runtime.Value) *runtime.Value {
	qb := &qbState{table: t, filter: filter, limitN: 0, offsetN: 0}
	return qbObject(qb)
}

type qbState struct {
	table   *lunexTable
	filter  *runtime.Value
	proj    []string
	sorts   []sortEntry
	limitN  int
	offsetN int
}

func qbObject(qb *qbState) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"where": runtime.FuncVal(&runtime.Function{Name: "where", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				qb.filter = args[0]
			}
			return qbObject(qb), nil
		}}),
		"and": runtime.FuncVal(&runtime.Function{Name: "and", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 && qb.filter != nil {
				qb.filter = runtime.ObjectVal(map[string]*runtime.Value{
					"$and": runtime.ArrayVal([]*runtime.Value{qb.filter, args[0]}),
				})
			} else if len(args) > 0 {
				qb.filter = args[0]
			}
			return qbObject(qb), nil
		}}),
		"or": runtime.FuncVal(&runtime.Function{Name: "or", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 && qb.filter != nil {
				qb.filter = runtime.ObjectVal(map[string]*runtime.Value{
					"$or": runtime.ArrayVal([]*runtime.Value{qb.filter, args[0]}),
				})
			} else if len(args) > 0 {
				qb.filter = args[0]
			}
			return qbObject(qb), nil
		}}),
		"select": runtime.FuncVal(&runtime.Function{Name: "select", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			qb.proj = nil
			if len(args) > 0 && args[0].Tag == runtime.TypeArray {
				for _, f := range args[0].ArrVal {
					qb.proj = append(qb.proj, f.ToString())
				}
			}
			return qbObject(qb), nil
		}}),
		"orderBy": runtime.FuncVal(&runtime.Function{Name: "orderBy", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				desc := false
				if len(args) > 1 {
					desc = strings.ToLower(args[1].ToString()) == "desc"
				}
				qb.sorts = append(qb.sorts, sortEntry{args[0].ToString(), desc})
			}
			return qbObject(qb), nil
		}}),
		"limit": runtime.FuncVal(&runtime.Function{Name: "limit", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				qb.limitN = int(args[0].ToNumber())
			}
			return qbObject(qb), nil
		}}),
		"offset": runtime.FuncVal(&runtime.Function{Name: "offset", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				qb.offsetN = int(args[0].ToNumber())
			}
			return qbObject(qb), nil
		}}),
		"skip": runtime.FuncVal(&runtime.Function{Name: "skip", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				qb.offsetN = int(args[0].ToNumber())
			}
			return qbObject(qb), nil
		}}),
		"page": runtime.FuncVal(&runtime.Function{Name: "page", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) >= 2 {
				n := int(args[0].ToNumber())
				size := int(args[1].ToNumber())
				if n < 1 {
					n = 1
				}
				qb.offsetN = (n - 1) * size
				qb.limitN = size
			}
			return qbObject(qb), nil
		}}),
		"exec": runtime.FuncVal(&runtime.Function{Name: "exec", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.ArrayVal(qb.table.execQuery(qb.filter, qb.proj, qb.sorts, qb.limitN, qb.offsetN)), nil
		}}),
		"first": runtime.FuncVal(&runtime.Function{Name: "first", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			rows := qb.table.execQuery(qb.filter, qb.proj, qb.sorts, 1, qb.offsetN)
			if len(rows) == 0 {
				return runtime.Null, nil
			}
			return rows[0], nil
		}}),
		"last": runtime.FuncVal(&runtime.Function{Name: "last", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			rows := qb.table.execQuery(qb.filter, qb.proj, qb.sorts, 0, 0)
			if len(rows) == 0 {
				return runtime.Null, nil
			}
			return rows[len(rows)-1], nil
		}}),
		"count": runtime.FuncVal(&runtime.Function{Name: "count", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			rows := qb.table.execQuery(qb.filter, nil, nil, 0, 0)
			return runtime.NumberVal(float64(len(rows))), nil
		}}),
		"exists": runtime.FuncVal(&runtime.Function{Name: "exists", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			rows := qb.table.execQuery(qb.filter, nil, nil, 1, 0)
			return runtime.BoolVal(len(rows) > 0), nil
		}}),
		"delete": runtime.FuncVal(&runtime.Function{Name: "delete", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.NumberVal(float64(qb.table.deleteRows(qb.filter, false))), nil
		}}),
		"update": runtime.FuncVal(&runtime.Function{Name: "update", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.NumberVal(0), nil
			}
			changes := make(map[string]*runtime.Value)
			for k, v := range args[0].ObjVal {
				changes[k] = v
			}
			return runtime.NumberVal(float64(qb.table.updateRows(qb.filter, changes, false))), nil
		}}),
	})
}

func dbObject(db *lunexDB) *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"name": runtime.StringVal(db.name),
		"path": runtime.StringVal(db.store.path),
		"table": runtime.FuncVal(&runtime.Function{Name: "table", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("table: name required")
			}
			return tableObject(db.table(args[0].ToString())), nil
		}}),
		"collection": runtime.FuncVal(&runtime.Function{Name: "collection", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("collection: name required")
			}
			return tableObject(db.table(args[0].ToString())), nil
		}}),
		"tables": runtime.FuncVal(&runtime.Function{Name: "tables", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			names := db.tableNames()
			out := make([]*runtime.Value, len(names))
			for i, n := range names {
				out[i] = runtime.StringVal(n)
			}
			return runtime.ArrayVal(out), nil
		}}),
		"drop": runtime.FuncVal(&runtime.Function{Name: "drop", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				name := args[0].ToString()
				if err := db.store.dropTable(name); err != nil {
					return runtime.Undefined, fmt.Errorf("db: could not drop table '%s': %w", name, err)
				}
				db.mu.Lock()
				delete(db.tables, name)
				db.mu.Unlock()
			}
			return runtime.Undefined, nil
		}}),
		"transaction": runtime.FuncVal(&runtime.Function{Name: "transaction", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeFunction {
				return runtime.Undefined, nil
			}
			txObj := dbObject(db)
			if runtime.CallFunction != nil {
				result, err := runtime.CallFunction(args[0], []*runtime.Value{txObj})
				if err != nil {
					return runtime.Null, err
				}
				return result, nil
			}
			return runtime.Undefined, nil
		}}),
		"dump": runtime.FuncVal(&runtime.Function{Name: "dump", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			names := db.tableNames()
			out := make(map[string]*runtime.Value, len(names))
			for _, name := range names {
				t := db.table(name)
				rows := t.execQuery(nil, nil, nil, 0, 0)
				out[name] = runtime.ArrayVal(rows)
			}
			return runtime.ObjectVal(out), nil
		}}),
		"load": runtime.FuncVal(&runtime.Function{Name: "load", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Undefined, nil
			}
			for tname, tdata := range args[0].ObjVal {
				t := db.table(tname)
				if tdata.Tag != runtime.TypeArray {
					continue
				}
				for _, item := range tdata.ArrVal {
					if item == nil || item.Tag != runtime.TypeObject {
						continue
					}
					doc := make(map[string]*runtime.Value, len(item.ObjVal))
					for k, v := range item.ObjVal {
						doc[k] = v
					}
					if _, err := t.insertDoc(doc); err != nil {
						return runtime.Undefined, err
					}
				}
			}
			return runtime.Undefined, nil
		}}),
		"close": runtime.FuncVal(&runtime.Function{Name: "close", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, db.store.close()
		}}),
	})
}

func DbModule() *runtime.Value {
	return runtime.ObjectVal(map[string]*runtime.Value{
		"create": runtime.FuncVal(&runtime.Function{Name: "create", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			name := "default"
			if len(args) > 0 && args[0] != nil && !args[0].IsNullish() {
				name = args[0].ToString()
			}
			db, err := globalDBEngine.open(name)
			if err != nil {
				return runtime.Null, fmt.Errorf("db.create: %w", err)
			}
			return dbObject(db), nil
		}}),
		"open": runtime.FuncVal(&runtime.Function{Name: "open", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			name := "default"
			if len(args) > 0 && !args[0].IsNullish() {
				name = args[0].ToString()
			}
			db, err := globalDBEngine.open(name)
			if err != nil {
				return runtime.Null, fmt.Errorf("db.open: %w", err)
			}
			return dbObject(db), nil
		}}),
		"connect": runtime.FuncVal(&runtime.Function{Name: "connect", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			name := "default"
			if len(args) > 0 && !args[0].IsNullish() {
				name = args[0].ToString()
			}
			db, err := globalDBEngine.open(name)
			if err != nil {
				return runtime.Null, fmt.Errorf("db.connect: %w", err)
			}
			return dbObject(db), nil
		}}),
		"drop": runtime.FuncVal(&runtime.Function{Name: "drop", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) > 0 {
				if err := globalDBEngine.drop(args[0].ToString()); err != nil {
					return runtime.Undefined, fmt.Errorf("db.drop: %w", err)
				}
			}
			return runtime.Undefined, nil
		}}),
		"list": runtime.FuncVal(&runtime.Function{Name: "list", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			names := globalDBEngine.list()
			out := make([]*runtime.Value, len(names))
			for i, n := range names {
				out[i] = runtime.StringVal(n)
			}
			return runtime.ArrayVal(out), nil
		}}),
		"table": runtime.FuncVal(&runtime.Function{Name: "table", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("table: name required")
			}
			db, err := globalDBEngine.open("default")
			if err != nil {
				return runtime.Null, fmt.Errorf("db.table: %w", err)
			}
			return tableObject(db.table(args[0].ToString())), nil
		}}),
		"collection": runtime.FuncVal(&runtime.Function{Name: "collection", Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("collection: name required")
			}
			db, err := globalDBEngine.open("default")
			if err != nil {
				return runtime.Null, fmt.Errorf("db.collection: %w", err)
			}
			return tableObject(db.table(args[0].ToString())), nil
		}}),
	})
}
