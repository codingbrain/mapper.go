package mapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Loader loads and parses map from file/string
type Loader struct {
	Map     map[string]interface{}
	Decoder Decoder
}

// Decoder defines the interface for parsing the content
type Decoder interface {
	Decode(content []byte) (interface{}, error)
}

// LoadString decodes the content in string
func (l *Loader) LoadString(content string) error {
	return l.LoadBytes([]byte(content))
}

// LoadBytes decodes the content in bytes
func (l *Loader) LoadBytes(content []byte) error {
	decoder := l.Decoder
	if decoder == nil {
		decoder = &AutoDecoder{}
	}
	d, err := decoder.Decode(content)
	if err == nil {
		if m, ok := d.(map[string]interface{}); ok {
			l.Map = m
		} else {
			err = fmt.Errorf("content is not a map")
		}
	}
	return err
}

// LoadStream loads from a stream
func (l *Loader) LoadStream(s io.Reader) error {
	content, err := ioutil.ReadAll(s)
	if err != nil {
		return err
	}
	return l.LoadBytes(content)
}

// LoadFile loads from a file or stdin if fn is empty or '-'
func (l *Loader) LoadFile(fn string) error {
	if fn == "" || fn == "-" {
		return l.LoadStream(os.Stdin)
	}
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()
	return l.LoadStream(f)
}

// Loaded determines if content has been loaded
func (l *Loader) Loaded() bool {
	return l.Map != nil
}

// As maps the decoded content into specific type
func (l *Loader) As(out interface{}) error {
	if l.Loaded() {
		return Map(out, l.Map)
	}
	return nil
}

// JSONDecoder decodes content in JSON
type JSONDecoder struct {
}

// Decode implements Decoder
func (d *JSONDecoder) Decode(content []byte) (out interface{}, err error) {
	out = make(map[string]interface{})
	err = json.Unmarshal(content, out)
	return
}

// YAMLDecoder decodes content in YAML
type YAMLDecoder struct {
}

// Decode implements Decoder
func (d *YAMLDecoder) Decode(content []byte) (out interface{}, err error) {
	out = make(map[string]interface{})
	err = yaml.Unmarshal(content, out)
	if err == nil {
		out = StringifyKeys(out)
	}
	return
}

// AutoDecoder selects the correct decoder by detecting the content
type AutoDecoder struct {
}

// Decode implements Decoder
func (d *AutoDecoder) Decode(content []byte) (out interface{}, err error) {
	var decoder Decoder
	if bytes.HasPrefix(bytes.TrimSpace(content), []byte{'{'}) {
		decoder = &JSONDecoder{}
	} else {
		decoder = &YAMLDecoder{}
	}
	return decoder.Decode(content)
}
