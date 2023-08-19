package rec_files

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

type Field struct {
	Name  string
	Value string
}
type Record []Field

func (f Field) String() string {
	return fmt.Sprintf("%s: %s", f.Name, f.Value)
}

func (f Field) IsEmpty() bool {
	return f.Name == "" && f.Value == ""
}

func (f Field) UnEscapedValue() string {
	return regexp.MustCompile(`\n\+\s`).ReplaceAllString(f.Value, "\n")
}

func (f Field) EscapedValue() string {
	return strings.ReplaceAll(f.Value, "\n", "\n+ ")
}

func (r Record) ToMap() map[string]string {
	m := make(map[string]string, len(r))
	for _, field := range r {
		m[field.Name] = field.Value
	}
	return m
}

func Read(file fs.File) []Record {
	records := make([]Record, 0)
	scanner := bufio.NewScanner(file)
	currentRecord := make([]Field, 0)
	currentField := Field{}
	linePart := ""
	fieldNamePattern := regexp.MustCompile(`^([a-zA-Z%][a-zA-Z0-9_]*):[\t ]?`)
	tryCommitCurrentField := func() {
		if !currentField.IsEmpty() {
			currentRecord = append(currentRecord, currentField)
		}
	}
	tryCommitCurrentRecord := func() {
		if len(currentRecord) > 0 {
			records = append(records, currentRecord)
		}
	}
	for scanner.Scan() {
		line := linePart + scanner.Text()
		linePart = ""

		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, "\\") {
			linePart = line[:len(line)-1]
			continue
		}

		if fieldNamePattern.MatchString(line) {
			tryCommitCurrentField()
			matches := fieldNamePattern.FindStringSubmatch(line)
			currentField = Field{
				Name:  matches[1],
				Value: strings.TrimSpace(line[len(matches[0]):]),
			}
		} else if line == "" {
			tryCommitCurrentField()
			currentField = Field{}
			tryCommitCurrentRecord()
			currentRecord = make([]Field, 0)
		} else {
			currentField.Value = strings.TrimSpace(currentField.Value + line)
		}
	}

	tryCommitCurrentField()
	currentField = Field{}
	tryCommitCurrentRecord()

	return records
}

func Write(file *os.File, records []Record) error {
	sanitizeFieldname := func(s string) string {
		saneFieldname := strings.ReplaceAll(s, " ", "_")
		if saneFieldname != s {
			println(fmt.Sprintf("WARNING - Sanitizing fieldname: '%s' -> '%s'", s, saneFieldname))
		}
		return saneFieldname
	}
	for _, record := range records {
		for _, field := range record {
			_, err := file.WriteString(sanitizeFieldname(field.Name) + ": " + field.EscapedValue() + "\n")
			if err != nil {
				return err
			}
		}
		_, err := file.WriteString("\n")
		if err != nil {
			return err
		}
	}
	return nil
}
