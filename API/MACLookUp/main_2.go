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
    "time"
)

// Device represents information about a network device
type Device struct {
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
        log.Printf("Bad request: missing mac parameter from %s", r.RemoteAddr)
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
    if err := json.NewEncoder(w).Encode(device); err != nil {
        log.Printf("Error encoding response: %v", err)
    }
}

type statusRecorder struct {
    http.ResponseWriter
    status int
}

func (r *statusRecorder) WriteHeader(code int) {
    r.status = code
    r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        rec := &statusRecorder{ResponseWriter: w, status: 200}
        log.Printf("Started %s %s?%s from %s", r.Method, r.URL.Path, r.URL.RawQuery, r.RemoteAddr)
        next.ServeHTTP(rec, r)
        log.Printf("Completed %d %s in %v", rec.status, http.StatusText(rec.status), time.Since(start))
    })
}

func main() {
    // Load IEEE OUI database
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

    // Server mode with logging
    handler := http.HandlerFunc(handleMacLookup)
    http.Handle("/", loggingMiddleware(handler))
    http.Handle("/metrics", loggingMiddleware(handler))
    http.Handle("/api/mac-lookup", loggingMiddleware(handler))
    http.Handle("/routerease/mac-address/api/mac-lookup", loggingMiddleware(handler))

    port := ":8080"
    log.Printf("Starting MAC lookup server on port %s", port)
    log.Fatal(http.ListenAndServe(port, nil))
}