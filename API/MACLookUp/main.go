package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// Device represents information about a network device
type Device struct {
	MAC          string `json:"mac"`
	Manufacturer string `json:"manufacturer"`
	Type         string `json:"type,omitempty"`
	IsRandom     bool   `json:"is_random"`
}

type DeviceFull struct {
	MAC          string `json:"mac"`
	Manufacturer string `json:"manufacturer"`
	Type         string `json:"type,omitempty"`
	IsRandom     bool   `json:"is_random"`
}

var ouiDB = make(map[string]string)

func loadOUIDatabase(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Match OUI format from IEEE database (example: "00AABB   Manufacturer Name")
		if match := regexp.MustCompile(`^([0-9A-F]{6})\s+(.+)$`).FindStringSubmatch(line); len(match) == 3 {
			ouiDB[match[1]] = strings.TrimSpace(match[2])
		}
	}

	return scanner.Err()
}

func normalizeMac(mac string) string {
	// Remove any separators and convert to uppercase
	cleanMac := strings.ToUpper(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(mac, ":", ""),
				"-", ""),
			".", ""))
	return cleanMac
}

func getManufacturer(mac string) string {
	// Normalize MAC and get first 6 characters (OUI)
	normalizedMac := normalizeMac(mac)
	if len(normalizedMac) < 6 {
		return "Invalid MAC"
	}

	oui := normalizedMac[:6]
	manufacturer, exists := ouiDB[oui]
	if !exists {
		return "Unknown"
	}
	return manufacturer
}

func isRandomizedMAC(mac string) bool {
	// Check if the MAC is likely a randomized/privacy MAC
	// Locally administered addresses have second bit of first byte set
	normalizedMac := normalizeMac(mac)
	if len(normalizedMac) < 2 {
		return false
	}

	// Convert first byte to int
	var firstByte int
	fmt.Sscanf(normalizedMac[:2], "%X", &firstByte)

	// Check if second bit is set (locally administered)
	return (firstByte & 0x02) == 0x02
}

func handleMacLookup(w http.ResponseWriter, r *http.Request) {
	mac := r.URL.Query().Get("mac")
	if mac == "" {
		http.Error(w, "MAC address is required", http.StatusBadRequest)
		return
	}

	device := Device{
		MAC:          mac,
		Manufacturer: getManufacturer(mac),
		IsRandom:     isRandomizedMAC(mac),
	}

	// Infer device type if possible (example logic)
	if strings.Contains(strings.ToLower(device.Manufacturer), "apple") {
		device.Type = "iOS/Mac Device"
	} else if strings.Contains(strings.ToLower(device.Manufacturer), "samsung") {
		device.Type = "Samsung Device"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}
func handleMacLookupFull(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		Devices []map[string]interface{} `json:"devices"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	for i, device := range requestBody.Devices {
		mac, ok := device["mac"].(string)
		if !ok || mac == "" {
			device["manufacturer"] = "Invalid MAC"
			continue
		}

		manufacturer := getManufacturer(mac)
		isRandom := isRandomizedMAC(mac)

		device["manufacturer"] = manufacturer
		device["is_random"] = isRandom

		// Infer device type if possible
		if strings.Contains(strings.ToLower(manufacturer), "apple") {
			device["type"] = "iOS/Mac Device"
		} else if strings.Contains(strings.ToLower(manufacturer), "samsung") {
			device["type"] = "Samsung Device"
		} else {
			device["type"] = "Unknown"
		}

		requestBody.Devices[i] = device
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requestBody)
}
func main() {
	// Load IEEE OUI database

	//err := loadOUIDatabase("/usr/share/arp-scan/ieee-oui.txt")
	err := loadOUIDatabase("ieee-oui.txt")
	if err != nil {
		log.Printf("Warning: Could not load OUI database: %v", err)
		log.Println("Attempting to download database...")
	}

	// Command line mode
	if len(os.Args) > 1 && os.Args[1] != "server" {
		mac := os.Args[1]
		device := Device{
			MAC:          mac,
			Manufacturer: getManufacturer(mac),
			IsRandom:     isRandomizedMAC(mac),
		}

		jsonOutput, _ := json.MarshalIndent(device, "", "  ")
		fmt.Println(string(jsonOutput))
		return
	}

	// Server mode
	http.HandleFunc("/", handleMacLookupFull)
	//http.HandleFunc("/", handleMacLookup)

	port := ":8080"
	log.Printf("Starting MAC lookup server on port %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
