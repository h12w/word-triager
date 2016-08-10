package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/nsf/termbox-go"
)

func main() {
	log.SetOutput(os.Stderr)
	if len(os.Args) != 2 {
		fmt.Println("word-triager words.txt")
		fmt.Println("type (y/n)")
		return
	}
	inputFile := os.Args[1]
	if err := run(inputFile); err != nil {
		log.Fatal(err)
	}
}
func run(inputFile string) error {
	asker, err := newTermAsker()
	if err != nil {
		return err
	}
	defer asker.Close()
	triager, err := newTriager(asker)
	if err != nil {
		return err
	}
	f, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word == "" {
			continue
		}
		if err := triager.Triage(word); err != nil {
			triager.Save()
			return err
		}
	}
	if scanner.Err() != nil {
		return err
	}
	if err := triager.Save(); err != nil {
		return err
	}
	return nil
}

type termAsker struct{}

func newTermAsker() (*termAsker, error) {
	err := termbox.Init()
	if err != nil {
		return nil, err
	}
	termbox.SetInputMode(termbox.InputEsc)
	return &termAsker{}, nil
}

func (a *termAsker) Close() {
	termbox.Close()
}

func print(x, y int, text string) {
	fg, bg := termbox.ColorWhite, termbox.ColorDefault
	for i, c := range text {
		termbox.SetCell(x+i, y, c, fg, bg)
	}
}

func (a *termAsker) Ask(word string) (WordState, error) {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	w, h := termbox.Size()
	print(w/2-len(word)/2, h/2, word)
	termbox.Flush()
	ev := termbox.PollEvent()
	switch ev.Type {
	case termbox.EventKey:
		switch ev.Ch {
		case 'y', 'Y':
			return Known, nil
		case 'n', 'N', ' ':
			return Unknown, nil
		case 's', 'S':
			return Skip, nil
		}
	case termbox.EventError:
		return Unknown, ev.Err
	}
	return Unknown, errors.New("terminated")
}

type WordState int

const (
	Unknown WordState = iota
	Known
	Skip
)

type Asker interface {
	Ask(word string) (WordState, error)
}

type Triager struct {
	Known   []string
	Unknown []string
	Skip    []string
	Asker
}

func newTriager(asker Asker) (*Triager, error) {
	t := &Triager{Asker: asker}
	return t, t.load()
}

func (t *Triager) load() error {
	var err error
	t.Known, err = loadWords("known.txt")
	if err != nil {
		return err
	}
	t.Unknown, err = loadWords("unknown.txt")
	if err != nil {
		return err
	}
	t.Skip, err = loadWords("skip.txt")
	if err != nil {
		return err
	}
	return nil
}

func (t *Triager) Save() error {
	sort.Strings(t.Known)
	sort.Strings(t.Skip)
	if err := saveWords(t.Known, "known.txt"); err != nil {
		return err
	}
	if err := saveWords(t.Unknown, "unknown.txt"); err != nil {
		return err
	}
	if err := saveWords(t.Skip, "skip.txt"); err != nil {
		return err
	}
	return nil
}

func (t *Triager) Triage(word string) error {
	if in(word, t.Known) || in(word, t.Unknown) || in(word, t.Skip) {
		return nil
	}

	state, err := t.Ask(word)
	if err != nil {
		return err
	}

	switch state {
	case Known:
		t.Known = append(t.Known, word)
	case Unknown:
		t.Unknown = append(t.Unknown, word)
	case Skip:
		t.Skip = append(t.Skip, word)
	}

	return nil
}
func in(word string, words []string) bool {
	for _, w := range words {
		if strings.EqualFold(word, w) {
			return true
		}
	}
	return false
}

func loadWords(file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	words := strings.Split(string(buf), "\n")
	for len(words) > 0 && words[len(words)-1] == "" {
		words = words[:len(words)-1]
	}
	return words, nil
}

func saveWords(words []string, file string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, word := range words {
		if _, err := f.Write([]byte(word)); err != nil {
			return err
		}
		if _, err := f.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return nil
}
