package main

import (
	"bufio"
	"fmt"
	"github.com/ahmetalpbalkan/go-cursor"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/term"
	"os"
)

var screenX int = 1
var screenY int = 1
var textX int = 0
var lineno int = 0
var width int = 0
var height int = 0
var quit bool = false
var filename string
var top *line
var topOfScreen *line
var bottom *line
var currentLine *line

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

func walkBack() {
	if textX > 0 {
		textX--
	}
	if screenX > 1 {
		screenX--
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

func left() {
	if currentLine != nil {
		if textX > 0 {
			textX--
			if currentLine.text[textX] == '\t' {
				screenX -= 8
			} else {
				screenX--
			}
			restore()
		}
	}
}

func right() {
	if currentLine != nil {
		if textX < len(currentLine.text)-1 {
			textX++
			if currentLine.text[textX] == '\t' {
				screenX += 8
			} else {
				screenX++
			}
			restore()
		}
	}
}

func up() {
	if screenY > 1 {
		screenY--
		currentLine = currentLine.prev
		lineno--
	} else if topOfScreen.prev != nil {
		clear()
		topOfScreen = topOfScreen.prev
		currentLine = currentLine.prev
		lineno--
	}
	restore()
}

func down() {
	if screenY < height-1 {
		screenY++
		currentLine = currentLine.next
		lineno++
	} else {
		clear()
		topOfScreen = topOfScreen.next
		currentLine = currentLine.next
		lineno++
	}
	restore()
}

func startOfLine() {
	screenX = 1
	textX = 0
	restore()
}

func displayLineno() {
	clearBanner()
	move(50, height)
	fmt.Printf("%d - %d", 1+textX, 1+lineno)
	restore()
}

func flash(msg string) {
	move(1, height)
	fmt.Print(msg)
	restore()
}

func clearBanner() {
	flash("                                                               ")
	restore()
}

func insert() {
	clearBanner()
	flash("-- INSERT --")
	defer clearBanner()
	for {
		c := getchar()
		switch c {
		case ESC_CODE:
			walkBack()
			return
		case ENTER_CODE:
			return // TODO: insert a new line
		default:
			// add character to string at proper position
			pos := textX
			txt := currentLine.text
			if pos == len(currentLine.text) {
				currentLine.text = fmt.Sprintf("%s%c", txt, c)
				textX++
				screenX++
			} else {
				currentLine.text = fmt.Sprintf(
					"%s%c%s",
					txt[:pos],
					c,
					txt[pos:],
				)
			}
			displayLine(currentLine.text, screenY)
			right()
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

func displayLine(line string, y int) {
	move(1, y)
	fmt.Print(line)
	restore()
}

func draw() {
	i := 0
	for line := topOfScreen; line != nil; line = line.next {
		if i >= height {
			break
		}
		displayLine(line.text, i)
		i++
	}
}

func scan() {
	for {
		if quit {
			return
		}
		draw()
		displayLineno()
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
		case 'A':
			textX = len(currentLine.text)
			screenX = textX + 1
			insert()
		case ':':
			command()
		case '0':
			startOfLine()
		default:
			flash(fmt.Sprintf("unknown command: '%c'", c))
		}
	}
}

func eventLoop() error {
	var err error

	width, height, err = terminal.GetSize(0)
	if err != nil {
		return err
	}

	oldIn, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldIn)

	oldOut, err := term.MakeRaw(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdout.Fd()), oldOut)

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
		line.prev = lines
		lines = lines.next
	}
	currentLine = top.next
	topOfScreen = top
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
