package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/common-nighthawk/go-figure"
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

func changeMAC(interfaceName, newMAC string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("wmic", "nic", "where",
			fmt.Sprintf("NetConnectionID='%s'", interfaceName),
			"call", "configure", "setting", fmt.Sprintf("MACAddress='%s'", newMAC))

	} else if runtime.GOOS == "linux" {
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

	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("Error: ", err)
		return
	} else {

		fmt.Print("\nList of network devices: \n")
		for _, iface := range interfaces {
			fmt.Printf("Name: %s\n", iface.Name)
			fmt.Printf("Index: %d\n", iface.Index)
			fmt.Printf("MTU: %d\n", iface.MTU)
			fmt.Printf("Hardware address (MAC): %s\n", iface.HardwareAddr)
			fmt.Println("--------------")
		}
		return
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

	findInterface := flag.Bool("findInterface", false, "List available network interfaces to work with\n")
	interfaceName := flag.String("interface", "", "Name of the network interface\n")
	newMAC := flag.String("mac", "", "New MAC address\n")
	setRandom := flag.Bool("random", false, "Set a randomized MAC address\n")
	restore := flag.Bool("restore", false, "Restore the original MAC address\n")
	showHelp := flag.Bool("help", false, "Show help menu\n")

	flag.Parse()

	if *showHelp {

		banner()
		printHelp()
		os.Exit(0)
	}

	if *findInterface {
		findInterfaces()
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
