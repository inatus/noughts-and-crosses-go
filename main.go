package main

import (
	"fmt"
	"github.com/mattn/go-gtk/gtk"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	PORT   int = 6392
	HEIGHT int = 3
	WIDTH  int = 3
)

var (
	window               *gtk.Window
	label                *gtk.Label
	button               [][]*gtk.Button
	state                [][]int
	address              map[string]int
	addressList          *gtk.ComboBoxText
	blank, nought, cross *os.File
	startButton          *gtk.Button
	opponent             string
	isMyTurn             bool
	localAddr            net.Addr
)

func main() {
	opponent = ""
	isMyTurn = false
	blank = readResource("blank.png")
	nought = readResource("nought.png")
	cross = readResource("cross.png")

	gtk.Init(nil)
	window = gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("GTK Noughts & Crosses")
	window.Connect("destroy", gtk.MainQuit)

	// Message label
	label = gtk.NewLabel("Select an opponent")
	label.ModifyFontEasy("DejaVu Serif 20")

	address = make(map[string]int)

	// Buttons for game
	alignment := gtk.NewAlignment(1, 1, 1, 1)
	table := gtk.NewTable(3, 3, false)
	button = make([][]*gtk.Button, HEIGHT, HEIGHT)
	state = make([][]int, HEIGHT, HEIGHT)
	for i := 0; i < HEIGHT; i++ {
		button[i] = make([]*gtk.Button, WIDTH, WIDTH)
		state[i] = make([]int, WIDTH, WIDTH)
		for j := 0; j < WIDTH; j++ {
			button[i][j] = gtk.NewButton()
			image := gtk.NewImageFromFile(blank.Name())
			button[i][j].SetImage(image)
			table.Attach(button[i][j], uint(i), uint(i+1), uint(j), uint(j+1), gtk.FILL, gtk.FILL, 0, 0)
			state[i][j] = 0
			copiedI := i
			copiedJ := j
			button[i][j].Connect("clicked", func() {
				if opponent != "" && isMyTurn == true && state[copiedI][copiedJ] == 0 {
					state[copiedI][copiedJ] = 1
					image := gtk.NewImageFromFile(nought.Name())
					button[copiedI][copiedJ].SetImage(image)
					message := "done " + strconv.Itoa(copiedI) + " " + strconv.Itoa(copiedJ) + " "
					sendMessage(opponent, message)
					if finished := judge(); finished == false {
						label.SetLabel("Opponent's turn")
					}
					isMyTurn = false
				}
			})

		}
	}
	addressList = gtk.NewComboBoxText()

	// Start button
	startButton = gtk.NewButtonWithLabel("Start")
	startButton.Connect("clicked", func() {
		if addressList.GetActiveText() != "" {
			sendMessage(addressList.GetActiveText(), "start ")
			startButton.SetSensitive(false)
			addressList.SetSensitive(false)
			for i := 0; i < HEIGHT; i++ {
				for j := 0; j < WIDTH; j++ {
					image := gtk.NewImageFromFile(blank.Name())
					button[i][j].SetImage(image)
					state[i][j] = 0
				}
			}
		}

	})

	// Run goroutines for listening messages and broadcasting my IP
	go listen(addressList)
	go broadcast()

	fixed := gtk.NewFixed()

	alignment.Add(table)
	fixed.Put(label, 0, 10)
	fixed.Put(alignment, 0, 40)
	fixed.Put(addressList, 0, 370)
	fixed.Put(startButton, 120, 370)

	window.Add(fixed)

	window.SetSizeRequest(330, 400)
	window.ShowAll()

	gtk.Main()
}

func sendMessage(remoteAddr, message string) {
	serverAddr, err := net.ResolveUDPAddr("udp", remoteAddr+":"+strconv.Itoa(PORT))
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Panic("Unicast cannot be sent")
	}
	log.Println("Message sent to " + conn.RemoteAddr().String() + ": " + message)
	fmt.Fprintf(conn, message)
}

func judge() bool {
	msg := []string{"Draw!", "You've won!", "You've lost!"}
	msgNum := -1
	// Check whether either wins
	for n := 1; n <= 2; n++ {
		if (state[0][0] == n && state[1][0] == n && state[2][0] == n) ||
			(state[0][1] == n && state[1][1] == n && state[2][1] == n) ||
			(state[0][2] == n && state[1][2] == n && state[2][2] == n) ||
			(state[0][0] == n && state[0][1] == n && state[0][2] == n) ||
			(state[1][0] == n && state[1][1] == n && state[1][2] == n) ||
			(state[2][0] == n && state[2][1] == n && state[2][2] == n) ||
			(state[0][0] == n && state[1][1] == n && state[2][2] == n) ||
			(state[2][0] == n && state[1][1] == n && state[0][2] == n) {
			msgNum = n
		}
	}
	// Check whether blank cells exist
	blankCell := 0
	for i := 0; i < HEIGHT; i++ {
		for j := 0; j < WIDTH; j++ {
			if state[i][j] == 0 {
				blankCell++
			}
		}
	}
	if blankCell == 0 {
		msgNum = 0
	}
	// Finish game if the above conditions are matched
	if msgNum != -1 {
		label.SetLabel(msg[msgNum])
		opponent = ""
		startButton.SetSensitive(true)
		addressList.SetSensitive(true)
		isMyTurn = false
		return true
	}
	return false
}

func listen(addressList *gtk.ComboBoxText) {
	serverAddr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(PORT))
	l, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Println("Broadcast listening: PORT=" + strconv.Itoa(PORT))

	for {
		// Wait for a connection.
		data := make([]byte, 4096)
		_, remoteAddr, err := l.ReadFromUDP(data)
		if err != nil {
			log.Panic("Error while reading message: " + err.Error())
		}
		message := strings.Split(string(data), " ")
		switch message[0] {
		case "broadcast":
			if remoteAddr.String() == localAddr.String() {
				break
			}
			addressAry := strings.Split(remoteAddr.String(), ":")
			if _, ok := address[addressAry[0]]; ok == false {
				address[addressAry[0]] = len(address)
				addressList.AppendText(addressAry[0])
				log.Println("Broadcast accept: " + remoteAddr.String())
			}
		case "start":
			log.Println("Message received from " + remoteAddr.String() + ": " + string(data))
			addressAry := strings.Split(remoteAddr.String(), ":")
			if opponent == "" {
				opponent = addressAry[0]
				if _, ok := address[opponent]; ok == false {
					address[opponent] = len(address)
					addressList.AppendText(opponent)
				}
				addressList.SetActive(address[opponent])
				sendMessage(opponent, "accept ")
				startButton.SetSensitive(false)
				addressList.SetSensitive(false)
				isMyTurn = true
				for i := 0; i < HEIGHT; i++ {
					for j := 0; j < WIDTH; j++ {
						image := gtk.NewImageFromFile(blank.Name())
						button[i][j].SetImage(image)
						state[i][j] = 0
					}
				}
				label.SetLabel("Challenged: Your turn")
			} else {
				sendMessage(addressAry[0], "deny ")
			}
		case "accept":
			log.Println("Message received from " + remoteAddr.String() + ": " + string(data))
			addressAry := strings.Split(remoteAddr.String(), ":")
			opponent = addressAry[0]
			label.SetLabel("Opponent's turn")
		case "deny":
			log.Println("Message received from " + remoteAddr.String() + ": " + string(data))
			label.SetLabel("Opponent busy")
			startButton.SetSensitive(true)
			addressList.SetSensitive(true)
		case "done":
			log.Println("Message received from " + remoteAddr.String() + ": " + string(data))
			i, err := strconv.Atoi(message[1])
			if err != nil {
				log.Panic("Message cannot be retrieved: " + err.Error())
			}
			j, err := strconv.Atoi(message[2])
			if err != nil {
				log.Panic("Message cannot be retrieved: " + err.Error())
			}
			state[i][j] = 2
			image := gtk.NewImageFromFile(cross.Name())
			button[i][j].SetImage(image)
			isMyTurn = true
			if finished := judge(); finished == false {
				label.SetLabel("Your turn")
			}
		}
	}
}

func broadcast() {
	serverAddr, err := net.ResolveUDPAddr("udp", "255.255.255.255:"+strconv.Itoa(PORT))
	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		log.Panic("Broadcast cannot be sent")
	}
	localAddr = conn.LocalAddr()
	log.Println("Broadcasting...")
	for {
		fmt.Fprintf(conn, "broadcast ")
		time.Sleep(time.Second)
	}
}

func readResource(file string) *os.File {
	filename := os.Args[0]
	result, err := os.Open(path.Join(path.Dir(filename), file))
	if err != nil {
		result, err = os.Open(file)
		if err != nil {
			log.Panicln("Resource file " + file + " are not found.")
		}
	}
	log.Println("Reading resource: " + file)
	return result
}
