package core

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"time"
)

const (
	recordLayout = "15-04"
)

var (
	ErrRecordNotFound      = errors.New("record not found")
	ErrRecordAlreadyExists = errors.New("record already exists")
)

type Record struct {
	Start      time.Time  `json:"start"`
	End        *time.Time `json:"end"`
	Project    *Project   `json:"project"`
	IsBillable bool       `json:"is_billable"`
}

// LoadRecord loads the record with the given start time. Returns
// ErrRecordNotFound if the record cannot be found.
func (t *Timetrace) LoadRecord(start time.Time) (*Record, error) {
	path := t.fs.RecordFilepath(start)
	return t.loadRecord(path)
}

// ListRecords loads and returns all records from the given date. If no records
// are found, an empty slice and no error will be returned.
func (t *Timetrace) ListRecords(date time.Time) ([]*Record, error) {
	dir := t.fs.RecordDirFromDate(date)
	paths, err := t.fs.RecordFilepaths(dir, func(_, _ string) bool {
		return true
	})
	if err != nil {
		return nil, err
	}

	records := make([]*Record, 0)

	for _, path := range paths {
		record, err := t.loadRecord(path)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// SaveRecord persists the given record. Returns ErrRecordAlreadyExists if the
// record already exists and saving isn't forced.
func (t *Timetrace) SaveRecord(record Record, force bool) error {
	path := t.fs.RecordFilepath(record.Start)

	if _, err := os.Stat(path); os.IsExist(err) && !force {
		return ErrRecordAlreadyExists
	}

	if err := t.fs.EnsureRecordDir(record.Start); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(&record, "", "\t")
	if err != nil {
		return err
	}

	_, err = file.Write(bytes)

	return err
}

// DeleteRecord removes the given record. Returns ErrRecordNotFound if the
// project doesn't exist.
func (t *Timetrace) DeleteRecord(record Record) error {
	path := t.fs.RecordFilepath(record.Start)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ErrRecordNotFound
	}

	return os.Remove(path)
}

func (t *Timetrace) loadAllRecords(date time.Time) ([]*Record, error) {
	dir := t.fs.RecordDirFromDate(date)

	recordFilepaths, err := t.fs.RecordFilepaths(dir, func(_, _ string) bool {
		return true
	})
	if err != nil {
		return nil, err
	}

	var records []*Record

	for _, recordFilepath := range recordFilepaths {
		record, err := t.loadRecord(recordFilepath)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// loadLatestRecord loads the youngest record. This may also be a record from
// another day. If there is no latest record, nil and no error will be returned.
func (t *Timetrace) loadLatestRecord() (*Record, error) {
	latestDirs, err := t.fs.RecordDirs()
	if err != nil {
		return nil, err
	}

	if len(latestDirs) == 0 {
		return nil, nil
	}

	dir := latestDirs[len(latestDirs)-1]

	latestRecords, err := t.fs.RecordFilepaths(dir, func(a, b string) bool {
		timeA, _ := time.Parse(recordLayout, a)
		timeB, _ := time.Parse(recordLayout, b)
		return timeA.Before(timeB)
	})
	if err != nil {
		return nil, err
	}

	if len(latestRecords) == 0 {
		return nil, nil
	}

	path := latestRecords[len(latestRecords)-1]

	return t.loadRecord(path)
}

// loadOldestRecord returns the oldest record of the given date. If there is no
// oldest record, nil and no error will be returned.
func (t *Timetrace) loadOldestRecord(date time.Time) (*Record, error) {
	dir := t.fs.RecordDirFromDate(date)

	oldestRecords, err := t.fs.RecordFilepaths(dir, func(a, b string) bool {
		timeA, _ := time.Parse(recordLayout, a)
		timeB, _ := time.Parse(recordLayout, b)
		return timeA.After(timeB)
	})
	if err != nil {
		return nil, err
	}

	if len(oldestRecords) == 0 {
		return nil, nil
	}

	path := oldestRecords[0]

	return t.loadRecord(path)
}

func (t *Timetrace) loadRecord(path string) (*Record, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	var record Record

	if err := json.Unmarshal(file, &record); err != nil {
		return nil, err
	}

	return &record, nil
}
