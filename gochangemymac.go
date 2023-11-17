package main

import (
	"flag"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/jedib0t/go-pretty/v6/table"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
	_ "github.com/google/gopacket/pcap"
	_ "github.com/jedib0t/go-pretty/v6/table"
)

func getOriginalMAC(interfaceName string) (string, error) {

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		cmd = exec.Command("wmic", "nic", "where",
			fmt.Sprintf("NetConnectionID='%s'", interfaceName), "get", "MACAddress", "/format:list")

	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("cat", fmt.Sprintf("/sys/class/net/%s/address", interfaceName))

	} else {
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func restoreOriginalMAC(interfaceName, originalMAC string) error {
	return changeMAC(interfaceName, originalMAC)
}

func winRestoreOriginalMAC() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Fatal(err)
	}

	if len(interfaces) > 0 {
		originalMAC := interfaces[0].HardwareAddr
		fmt.Printf("Original MAC address: %s\n", originalMAC)
	} else {
		log.Fatal("unable to retrieve original MAC address")
	}

	return
}

func setRandomMAC(interfaceName string) error {
	randMAC, err := generateRandomMAC()
	if err != nil {
		return err
	}

	return changeMAC(interfaceName, randMAC)
}

func generateRandomMAC() (string, error) {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)

	firstByte := 0x02
	otherBytes := make([]byte, 5)
	r.Read(otherBytes)

	randomMAC := fmt.Sprintf(
		"%02x:%02x:%02x:%02x:%02x:%02x", firstByte,
		otherBytes[0], otherBytes[1], otherBytes[2], otherBytes[3], otherBytes[4])

	return randomMAC, nil
}

func winChangeMAC(interfaceName string, newMAC string) error {

	// Disable the network interface
	disableCmd := exec.Command("netsh", "interface", "set", "interface", interfaceName, "admin=disable")
	err := disableCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to disable network interface: %w", err)
	}

	handle, err := pcap.OpenLive(interfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return err
	}
	defer handle.Close()

	// Make a packet to change the MAC address
	packetData := []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // Destination MAC (broadcast)
		newMAC[0], newMAC[1], newMAC[2], newMAC[3], newMAC[4], newMAC[5], // Source MAC
		0x08, 0x06, // EtherType: ARP
		0x00, 0x01, // Hardware Type: Ethernet (1)
		0x08, 0x00, // Protocol Type: IPv4 (0x0800)
		0x06,       // Hardware Address Length: 6
		0x04,       // Protocol Address Length: 4
		0x00, 0x02, // Operation: Reply (2)
		// Sender Hardware Address (Source MAC)
		newMAC[0], newMAC[1], newMAC[2], newMAC[3], newMAC[4], newMAC[5],
		// Sender Protocol Address (IPv4 address, e.g., 192.168.1.1)
		0x00, 0x00, 0x00, 0x00,
		// Target Hardware Address (Destination MAC)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Target Protocol Address (IPv4 address, e.g., 192.168.1.2)
		0x00, 0x00, 0x00, 0x00,
	}

	packet := gopacket.NewPacket(packetData, layers.LayerTypeEthernet, gopacket.Default)

	// Send the packet to change the MAC address
	err = handle.WritePacketData(packet.Data())
	if err != nil {
		return err
	}

	// Enable the network interface
	enableCmd := exec.Command("netsh", "interface", "set", "interface", interfaceName, "admin=enable")
	err = enableCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to enable network interface: %w", err)
	}

	fmt.Printf("Windows MAC address of %s changed to %s\n", interfaceName, newMAC)
	return nil

}

func changeMAC(interfaceName, newMAC string) error {
	var cmd *exec.Cmd

	if runtime.GOOS == "linux" {
		cmd = exec.Command("ip", "link", "set", interfaceName, "address", newMAC)

	} else {
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func findInterfaces() {
	banner()

	t := table.NewWriter()

	t.SetTitle("Network interfaces")

	t.AppendHeader(table.Row{
		"#", "Name", "Index", "MTU", "MAC Address"})

	if runtime.GOOS == "linux" {
		interfaces, err := net.Interfaces()

		if err != nil {
			fmt.Println("Error: ", err)
			return

		}

		for i, iface := range interfaces {
			t.AppendRow(table.Row{
				i + 1, iface.Name, iface.Index, iface.MTU, iface.HardwareAddr})

			fmt.Print("\033[H\033[2J")
			banner() // Clear the console
			fmt.Println(t.Render())
			time.Sleep(50 * time.Millisecond)
		}
		return

	} else if runtime.GOOS == "windows" {

		// Get the list of available interfaces
		devices, err := pcap.FindAllDevs()
		if err != nil {
			log.Fatal(err)
		}

		for i, device := range devices {
			t.AppendRow(table.Row{
				i + 1, device.Name, device.Description})

			fmt.Print("\033[H\033[2J")
			banner() // Clear the console
			fmt.Println(t.Render())
			time.Sleep(500 * time.Millisecond)
		}

	}
}

func banner() {

	asciiLogo1 := figure.NewColorFigure("Go change", "slant", "cyan", true)
	asciiLogo2 := figure.NewColorFigure("my mac!", "slant", "yellow", true)

	asciiLogo1.Print()
	asciiLogo2.Print()
}

func printHelp() {
	fmt.Print("\nUsage:  ")
	fmt.Printf("  gochangemymac -interface <interface_name> -mac <new_mac_address>\n\n")
	fmt.Print("Options:\n\n")
	flag.PrintDefaults()
}

func main() {

	if len(os.Args) < 2 {
		banner()
		fmt.Print("\nUse: gochangemymac -help for usage manual.\n\n")
		return
	}

	//Flag parameters
	windows := flag.Bool("windows",
		false, "Execute gochangemymac with windows compatibility\n")

	winRestore := flag.Bool("winRestore",
		false, "Restore Windows interface original MAC address\n"+
			"Both arguments -windows and -winRestore "+
			"have to be set just so restoration takes effect\n")

	findInterface := flag.Bool("findInterface",
		false, "List available network interfaces to work with\n")

	interfaceName := flag.String("interface", "", "Name of the network interface\n")

	newMAC := flag.String("mac", "", "New MAC address\n")

	setRandom := flag.Bool("random", false, "Set a randomized MAC address\n")

	restore := flag.Bool("restore", false, "Restore the original MAC address\n")

	showHelp := flag.Bool("help", false, "Show help menu\n")

	flag.Parse()

	if *windows && *winRestore {
		winRestoreOriginalMAC()
	}

	if *windows == true && *interfaceName == "" {

		t := table.NewWriter()

		t.SetTitle("Network interfaces")

		t.AppendHeader(table.Row{
			"#", "Name", "Index"})

		// Get the list of available interfaces
		devices, err := pcap.FindAllDevs()
		if err != nil {
			log.Fatal(err)
		}

		for i, device := range devices {
			t.AppendRow(table.Row{
				i + 1, device.Name, device.Description})

			fmt.Print("\033[H\033[2J")
			banner() // Clear the console
			fmt.Println(t.Render())
			time.Sleep(50 * time.Millisecond)
		}

		// Prompt the user to choose an interface
		fmt.Print("Enter the number corresponding to the interface to change MAC address: ")
		var choice int
		if _, err := fmt.Scan(&choice); err != nil {
			log.Fatal("Error reading user input:", err)
			os.Exit(1)
		}

		// Validate user input
		if choice < 1 || choice > len(devices) {
			log.Fatal("Invalid choice. Please enter a valid number.")
		}

		// Get the selected network interface
		selectedDevice := devices[choice-1]

		// Prompt the user to enter the new MAC address
		fmt.Print("Enter the new MAC address (in the format XX:XX:XX:XX:XX:XX): ")
		var newMAC string
		if _, err := fmt.Scan(&newMAC); err != nil {
			log.Fatal("Error reading user input:", err)
			os.Exit(1)
		}

		// Change the MAC address of the selected interface
		err = winChangeMAC(selectedDevice.Name, newMAC)
		if err != nil {
			log.Fatal("Error changing MAC address:", err)
			os.Exit(1)
		}

		// Wait for a moment to allow the MAC address change to take effect
		time.Sleep(2 * time.Second)
	}

	if *showHelp {

		banner()
		printHelp()
		os.Exit(0)
	}

	if *findInterface {
		findInterfaces()
		os.Exit(0)
	}

	if *interfaceName == "" {
		log.Fatal("\nInterface name is required")
	}

	if *setRandom && *restore {
		log.Fatal("\nBoth -random and -restore options cannot be used together")
	}

	if *restore {
		originalMAC, err := getOriginalMAC(*interfaceName)
		if err != nil {
			log.Fatalf("\nError retrieving original MAC address: %v", err)
		}

		err = restoreOriginalMAC(*interfaceName, originalMAC)
		if err != nil {
			log.Fatalf("\nError restoring original MAC address: %v", err)
		}

		fmt.Printf("\nOriginal MAC address for %s restored: %s\n", *interfaceName, originalMAC)

	} else if *setRandom {
		err := setRandomMAC(*interfaceName)
		if err != nil {
			log.Fatalf("\nError setting randomized MAC address: %v", err)
		}

		fmt.Printf("\nRandomized MAC address set for %s\n, changed to %s", *interfaceName, *newMAC)

	} else if *newMAC != "" {
		if _, err := net.ParseMAC(*newMAC); err != nil {
			log.Fatalf("\nInvalid MAC address: %v", err)
		}

		err := changeMAC(*interfaceName, *newMAC)
		if err != nil {
			log.Fatalf("\nError changing MAC address: %v", err)
		}

		fmt.Printf("\nMAC address for %s changed to %s\n", *interfaceName, *newMAC)

	} else {
		log.Fatal("\nEither -mac, -random, or -restore option is required")
	}
}
