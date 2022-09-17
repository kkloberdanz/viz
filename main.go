package main

import (
	"bufio"
	"fmt"
	"github.com/ahmetalpbalkan/go-cursor"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/term"
	"os"
)

var screenX int = 0
var screenY int = 0
var width int = 0
var height int = 0
var quit bool = false
var filename string
var top *line
var bottom *line

const (
	ENTER_CODE = 13
	ESC_CODE   = 27
)

type line struct {
	text string
	prev *line
	next *line
}

func lineNew() *line {
	return &line{
		text: "",
		prev: nil,
		next: nil,
	}
}

func restore() {
	move(screenX, screenY)
}

func clear() {
	fmt.Print(cursor.ClearEntireScreen())
}

func getchar() byte {
	var b []byte = make([]byte, 1)
	os.Stdin.Read(b)
	return b[0]
}

func move(x int, y int) {
	fmt.Print(cursor.MoveTo(y, x))
}

func right() {
	screenX++
	restore()
}

func up() {
	screenY--
	restore()
}

func down() {
	if screenY < height {
		screenY++
	}
	restore()
}

func left() {
	if screenX > 0 {
		screenX--
	}
	restore()
}

func flash(msg string) {
	move(0, height)
	fmt.Print(msg)
	restore()
}

func clearBanner() {
	flash("                                                               ")
}

func insert() {
	clearBanner()
	flash("-- INSERT --")
	defer clearBanner()
	for {
		c := getchar()
		switch c {
		case ESC_CODE:
			return
		default:
			fmt.Printf("%c", c)
		}
	}
}

func execute(cmd string) {
	for _, c := range cmd[1:] {
		switch c {
		case 'w':
			writeFile()
		case 'q':
			quit = true
		default:
			flash(fmt.Sprintf("unknown command: '%c'", c))
			return
		}
	}
}

func command() {
	clearBanner()
	cmd := ":"
	for {
		flash(cmd)
		c := getchar()
		switch c {
		case ENTER_CODE:
			execute(cmd)
			return
		case ESC_CODE:
			clearBanner()
			return
		default:
			cmd += string(c)
		}
	}
}

func writeFile() {
	file, err := os.Create(filename)
	if err != nil {
		flash(fmt.Sprintf("failed to write: \"%s\": %v", filename, err))
		return
	}
	defer file.Close()

	for line := top.next; line != nil; line = line.next {
		file.Write([]byte(line.text))
		file.Write([]byte("\n"))
	}
	flash(fmt.Sprintf("wrote file: \"%s\"", filename))
}

func scan() {
	for {
		if quit {
			return
		}
		c := getchar()
		switch c {
		case 'l':
			right()
		case 'h':
			left()
		case 'j':
			down()
		case 'k':
			up()
		case 'i':
			insert()
		case ':':
			command()
		case '0':
			screenX = 0
			restore()
		}
	}
}

func eventLoop() error {
	var err error

	width, height, err = terminal.GetSize(0)
	if err != nil {
		return err
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	clear()
	move(screenX, screenY)
	scan()
	return nil
}

func readFile(filename string) {
	readFile, err := os.Open(filename)
	if err != nil {
		return // file does not exist, so we'll create a new one
	}
	defer readFile.Close()

	top = lineNew()
	lines := top

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	for fileScanner.Scan() {
		line := lineNew()
		line.text = fileScanner.Text()
		lines.next = line
		lines = lines.next
	}
}

func main() {
	if len(os.Args) > 1 {
		filename = os.Args[1]
		readFile(filename)
	}
	err := eventLoop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
}
