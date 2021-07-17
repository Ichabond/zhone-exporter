// zhone-exporter - a Prometheus exporter for the Zhone ZNID-GPON-2726A1-UK gateway

package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// InterfaceData is a struct providing a container for all relevant interface metrics available on the Zhone CPE platform
type InterfaceData struct {
	ID       string
	Name     string
	Status   float64
	IfSpeed  float64
	rxBytes  float64
	txBytes  float64
	rxFrames float64
	txFrames float64
	rxDrops  float64
	txDrops  float64
	rxErrs   float64
	txErrs   float64
}

// GPONData contains all metrics available for the GPON interface
type GPONData struct {
	ID          string
	Name        string
	Status      float64
	RXPower     float64
	TXPower     float64
	Transitions float64
}

// WifiClient collects the metrics provided for a wifi client on a given WLAN interface
type WifiClient struct {
	Interface       string
	MAC             string
	AssociatedTime  float64
	txFrames        float64
	TXUnicastFrames float64
	txErrs          float64
	TXRetries       float64
	TXRate          float64
	TxRetryRate     float64
	RXUnicastFrames float64
	RXBcastFrames   float64
	RXRate          float64
	RSSI            float64
	Noise           float64
	SNR             float64
	Quality         float64
}

var (
	rxBytes = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "receive_bytes"), "Received bytes per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	txBytes = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "transmit_bytes"), "Transmitted bytes per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	rxFrames = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "receive_frames"), "Received frames per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	txFrames = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "transmit_frames"), "Transmitted frames per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	rxErrs = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "receive_errors"), "Received errors per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	txErrs = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "transmit_errors"), "Transmitted errors per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	rxDrops = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "receive_drops"), "Received drops per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	txDrops = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "transmit_drops"), "Transmitted drops per interface.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	interfaceSpeed = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "if_speed"), "Interface Speed.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	interfaceStatus = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "", "if_status"), "Interface Status.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	gponRX = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "gpon", "receive_power"), "GPON Receive Power.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	gponTX = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "gpon", "transmit_power"), "GPON Transmit Power.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	gponTransitions = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "gpon", "up_transitions"), "GPON Link Up Transitions.", []string{
			"instance",
			"interface",
			"interface_name",
		}, nil)
	wifiAssoc = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "time_associated"), "Time Associated", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiTX = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "transmit_frames"), "Transmit Frames", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiTXUnicast = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "transmit_unicast_frames"), "Transmit Unicast Frames", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiErrs = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "transmit_errors"), "Transmit Failures", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiRetries = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "transmit_retries"), "Transmit Retries", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiRetryRate = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "transmit_retry_rate"), "Transmit Retry Rate", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiRXUnicast = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "receive_unicast_frames"), "Receive Unicast Frames", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiBcast = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "receive_broadcast_frames"), "Receive Multicast/Broadcast Frames", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiTXRate = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "transmit_rate"), "Transmit Rate", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiRXRate = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "receive_rate"), "Receive Rate", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiRSSI = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "rssi"), "RSSI", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiNoise = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "noise"), "Noise", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiSNR = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "snr"), "SNR", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
	wifiQuality = prometheus.NewDesc(
		prometheus.BuildFQName(
			"cpe", "wifi", "quality"), "Quality", []string{
			"instance",
			"wlan_interface",
			"client_mac",
		}, nil)
)

// ZhoneExporter contains the authentication parameters for the Zhone Web Interface
type ZhoneExporter struct {
	URL, username, password string
}

// NewZhoneExporter builds a new ZhoneExporter with the credentials provided
func NewZhoneExporter(url string, username string, password string) *ZhoneExporter {
	return &ZhoneExporter{
		URL:      url,
		username: username,
		password: password,
	}
}

// Describe provides the superset of descriptors to the provided channel
func (e *ZhoneExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- rxBytes
	ch <- txBytes
	ch <- rxFrames
	ch <- txFrames
	ch <- rxErrs
	ch <- txErrs
	ch <- rxDrops
	ch <- txDrops
	ch <- interfaceSpeed
	ch <- interfaceStatus
	ch <- gponRX
	ch <- gponTX
	ch <- gponTransitions
	ch <- wifiAssoc
	ch <- wifiTX
	ch <- wifiTXUnicast
	ch <- wifiErrs
	ch <- wifiRetries
	ch <- wifiRetryRate
	ch <- wifiRXUnicast
	ch <- wifiBcast
	ch <- wifiTXRate
	ch <- wifiRXRate
	ch <- wifiRSSI
	ch <- wifiNoise
	ch <- wifiSNR
	ch <- wifiQuality

}

// Collect will gather, parse and present the available Prometheus metrics
func (e *ZhoneExporter) Collect(ch chan<- prometheus.Metric) {
	statsdata, status, gpondata := e.FetchData()
	gpon := ParseGPONData(gpondata)
	interfaces := ParseInterfaceData(statsdata, status)
	wlanRE := regexp.MustCompile(`wl(\d+)$`)
	var wlanIDs []string
	for _, Interface := range interfaces {
		wlanMatch := wlanRE.FindStringSubmatch(Interface.ID)
		if wlanMatch != nil {
			wlanIDs = append(wlanIDs, wlanMatch[1])
		}
		if Interface.ID == "eth0" {
			Interface.Status = gpon.Status
			gpon.ID = Interface.ID
			gpon.Name = Interface.Name
			ch <- prometheus.MustNewConstMetric(
				gponRX, prometheus.GaugeValue, gpon.RXPower, e.URL, Interface.ID, Interface.Name,
			)
			ch <- prometheus.MustNewConstMetric(
				gponTX, prometheus.GaugeValue, gpon.TXPower, e.URL, Interface.ID, Interface.Name,
			)
			ch <- prometheus.MustNewConstMetric(
				gponTransitions, prometheus.GaugeValue, gpon.Transitions, e.URL, Interface.ID, Interface.Name,
			)
		}
		ch <- prometheus.MustNewConstMetric(
			rxBytes, prometheus.GaugeValue, Interface.rxBytes, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			txBytes, prometheus.GaugeValue, Interface.txBytes, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			rxFrames, prometheus.GaugeValue, Interface.rxFrames, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			txFrames, prometheus.GaugeValue, Interface.txFrames, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			rxDrops, prometheus.GaugeValue, Interface.rxDrops, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			txDrops, prometheus.GaugeValue, Interface.txDrops, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			rxErrs, prometheus.GaugeValue, Interface.rxErrs, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			txErrs, prometheus.GaugeValue, Interface.txErrs, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			interfaceSpeed, prometheus.GaugeValue, Interface.IfSpeed, e.URL, Interface.ID, Interface.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			interfaceStatus, prometheus.GaugeValue, Interface.Status, e.URL, Interface.ID, Interface.Name,
		)
	}
	wifi := e.FetchWirelessData(wlanIDs)
	wlanClients := ParseWirelessData(wifi)
	for i := range wlanClients {
		wlan := wlanClients[i]
		ch <- prometheus.MustNewConstMetric(
			wifiAssoc, prometheus.GaugeValue, wlan.AssociatedTime, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiTX, prometheus.GaugeValue, wlan.txFrames, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiTXUnicast, prometheus.GaugeValue, wlan.TXUnicastFrames, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiErrs, prometheus.GaugeValue, wlan.txErrs, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiRetries, prometheus.GaugeValue, wlan.TXRetries, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiRetryRate, prometheus.GaugeValue, wlan.TxRetryRate, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiRXUnicast, prometheus.GaugeValue, wlan.RXUnicastFrames, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiBcast, prometheus.GaugeValue, wlan.RXBcastFrames, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiTXRate, prometheus.GaugeValue, wlan.TXRate, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiRXRate, prometheus.GaugeValue, wlan.RXRate, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiRSSI, prometheus.GaugeValue, wlan.RSSI, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiNoise, prometheus.GaugeValue, wlan.Noise, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiSNR, prometheus.GaugeValue, wlan.SNR, e.URL, wlan.Interface, wlan.MAC,
		)
		ch <- prometheus.MustNewConstMetric(
			wifiQuality, prometheus.GaugeValue, wlan.Quality, e.URL, wlan.Interface, wlan.MAC,
		)
	}

}

// ParseWirelessData ingests an array with 2 maps, containing multiple goquery Documents. This is needed, as the WLAN client information is spread across 2 webpages
func ParseWirelessData(data [2]map[string]*goquery.Document) []WifiClient {
	//data[0] == zhnwlstatus
	//data[1] == zhnwlinfo
	var clients []WifiClient
	clientMap := make(map[string]WifiClient)
	// client information is encoded in a javascript variable which we extract
	clientsRE := regexp.MustCompile(`var\ wlClients\ =\ '(.+)';`)
	for wlanID, APs := range data[0] {
		table := APs.Find("#clientTable").Eq(0).Find("tbody").Eq(1).Text()
		clientListMatch := clientsRE.FindStringSubmatch(table)
		if clientListMatch == nil {
			continue
		}
		clientList := clientListMatch[1]
		clientListSlice := strings.Split(clientList, "#")
		var err error
		toFloat := func(s string) float64 {
			var f float64
			if err != nil {
				log.Fatal(err)
			}
			f, err = strconv.ParseFloat(s, 64)
			return f
		}
		for i := range clientListSlice {
			var clientMac net.HardwareAddr
			clientData := strings.Split(clientListSlice[i], "|")
			clientMac, err = net.ParseMAC(clientData[1])
			if err != nil {
				log.Fatal(err)
			}
			rssi := toFloat(clientData[2])
			noise := toFloat(clientData[3])
			snr := toFloat(clientData[4])
			quality := toFloat(clientData[5])
			if err != nil {
				log.Fatal(err)
			}
			clientMap[clientMac.String()] = WifiClient{Interface: "wl" + wlanID, MAC: clientMac.String(), RSSI: rssi, Noise: noise, SNR: snr, Quality: quality}
		}
	}
	for _, APs := range data[1] {
		clientListMatch := clientsRE.FindStringSubmatch(APs.Text())
		if clientListMatch == nil {
			continue
		}
		clientList := clientListMatch[1]
		clientListSlice := strings.Split(clientList, "#")
		var err error
		toFloat := func(s string) float64 {
			var f float64
			if err != nil {
				log.Fatal(err)
			}
			f, err = strconv.ParseFloat(s, 64)
			return f
		}
		for i := range clientListSlice {
			var clientMac net.HardwareAddr
			clientData := strings.Split(clientListSlice[i], "|")
			clientMac, err := net.ParseMAC(clientData[0])
			if err != nil {
				log.Fatal(err)
			}
			timeAssociated := toFloat(clientData[1])
			txFrames := toFloat(clientData[2])
			TXUnicastFrames := toFloat(clientData[3])
			txErrs := toFloat(clientData[4])
			TXRetries := toFloat(clientData[5])
			TxRetryRate := toFloat(clientData[6])
			RXUnicastFrames := toFloat(clientData[7])
			RXBcastFrames := toFloat(clientData[8])
			TXRate := toFloat(clientData[9])
			RXRate := toFloat(clientData[10])
			if err != nil {
				log.Fatal(err)
			}
			client := clientMap[clientMac.String()]
			client.AssociatedTime = timeAssociated
			client.txFrames = txFrames
			client.TXUnicastFrames = TXUnicastFrames
			client.txErrs = txErrs
			client.TXRetries = TXRetries
			client.TxRetryRate = TxRetryRate
			client.RXUnicastFrames = RXUnicastFrames
			client.RXBcastFrames = RXBcastFrames
			client.TXRate = TXRate
			client.RXRate = RXRate
			clientMap[clientMac.String()] = client
		}
	}
	for _, client := range clientMap {
		clients = append(clients, client)
	}
	return clients
}

// ParseinterfaceStatus will parse the status of interfaces, presented on the interfaces page
func ParseinterfaceStatus(data *goquery.Document) map[string][2]float64 {
	interfaceStatus := make(map[string][2]float64)
	dump := data.Text()
	// Same deal as with the Wifi bits. Encoded in a javascript var
	portlistRE := regexp.MustCompile(`var\ portlistAll\ \=\ '(.+)'`)
	portList := portlistRE.FindStringSubmatch(dump)[1]
	split := strings.Split(portList, "#")
	IDs := strings.Split(strings.Split(split[0], "/")[0], "|")
	IDs = IDs[0 : len(IDs)-1]
	values := strings.Split(split[1], "/")
	ifstate := strings.Split(values[0], "|")
	ifstate = ifstate[1:]
	ifspeed := strings.Split(values[1], "|")
	ifspeed = ifspeed[1:]
	var speeds []float64
	var states []float64
	for i := range ifspeed {
		if ifspeed[i] == "-" {
			speeds = append(speeds, float64(0))
		} else {
			speed, _ := strconv.ParseFloat(ifspeed[i], 64)
			speeds = append(speeds, speed)
		}
		if ifstate[i] == "Up" {
			states = append(states, float64(1))
		} else {
			states = append(states, float64(0))
		}
	}
	for i := range IDs {
		interfaceStatus[IDs[i]] = [2]float64{states[i], speeds[i]}
	}
	return interfaceStatus

}

// ParseInterfaceData parses the interface metrics provided
func ParseInterfaceData(data *goquery.Document, statusdata *goquery.Document) []InterfaceData {
	var interfaces []InterfaceData
	interfaceMap := ParseinterfaceStatus(statusdata)
	tables := data.Find("#table")
	table := tables.Eq(0)
	tbodies := table.Find("tbody").Slice(1, 3)
	IDRE := regexp.MustCompile(`(.+)\ \((.+)\)`)
	for i := range tbodies.Nodes {
		rows := tbodies.Eq(i).Find("tr")
		for j := range rows.Nodes {
			columns := rows.Eq(j).Find("td").Not("[valign='middle']")

			NameID := IDRE.FindStringSubmatch(columns.Eq(0).Text())
			var values []float64
			for k := range columns.Nodes {
				if k == 0 {
					continue
				}
				value, err := strconv.ParseFloat(columns.Eq(k).Text(), 64)
				if err != nil {
					log.Fatal(err)
				}
				values = append(values, value)
			}
			Interface := InterfaceData{
				ID:       NameID[2],
				Name:     NameID[1],
				rxBytes:  values[0],
				rxFrames: values[1],
				rxErrs:   values[2],
				rxDrops:  values[3],
				txBytes:  values[4],
				txFrames: values[5],
				txErrs:   values[6],
				txDrops:  values[7],
				Status:   interfaceMap[NameID[2]][0],
				IfSpeed:  interfaceMap[NameID[2]][1],
			}
			interfaces = append(interfaces, Interface)
		}
	}
	return interfaces
}

// ParseGPONData parses the GPON information into the GPONData struct
func ParseGPONData(data *goquery.Document) GPONData {
	//type GPONData struct {
	//ID          string
	//Name        string
	//Status      float64
	//RXPower     float64
	//TXPower     float64
	//Transitions float64
	//}

	var gpon GPONData
	table := data.Find("#table1").Eq(0)
	tbodies := table.Find("tbody")
	rows := tbodies.Eq(1).Find("tr")
	for i := range rows.Nodes {
		columns := rows.Eq(i).Find("td").Not(".hd")
		if columns.Eq(0).Text() == "Current Link State" {
			status := float64(0)
			if columns.Eq(1).Text() == "Up" {
				status = float64(1)
			}
			gpon.Status = status
		}
		if columns.Eq(0).Text() == "Link Up Transitions" {
			trans, _ := strconv.ParseFloat(columns.Eq(1).Text(), 64)
			gpon.Transitions = trans
		}
		if columns.Eq(0).Text() == "Receive Level" {
			level, err := strconv.ParseFloat(strings.TrimSpace(strings.Trim(columns.Eq(1).Text(), "dBm")), 64)
			if err != nil {
				log.Fatal(err)
			}
			gpon.RXPower = level
		}
		if columns.Eq(0).Text() == "Transmit Power" {
			level, err := strconv.ParseFloat(strings.TrimSpace(strings.Trim(columns.Eq(1).Text(), "dBm")), 64)
			if err != nil {
				log.Fatal(err)
			}
			gpon.TXPower = level
		}

	}
	return gpon
}

// FetchData executes the web scrapes required for Interface and GPON data, and returns the associated goquery Documents
func (e *ZhoneExporter) FetchData() (*goquery.Document, *goquery.Document, *goquery.Document) {
	urls := []url.URL{
		{Scheme: "http", Host: e.URL, Path: "statsifc.html", User: url.UserPassword(e.username, e.password)},
		{Scheme: "http", Host: e.URL, Path: "zhnethernetstatus.html", User: url.UserPassword(e.username, e.password)},
		{Scheme: "http", Host: e.URL, Path: "zhngponstatus.html", User: url.UserPassword(e.username, e.password)}}
	var results [3]*goquery.Document
	for i := range urls {
		res, err := http.Get(urls[i].String())
		if err != nil {
			log.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			log.Fatal("Status code: %d %s", res.StatusCode, res.Status)
		}
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Fatal(err)
		}
		results[i] = doc

	}
	return results[0], results[1], results[2]
}

//FetchWirelessData performs the same functions as FetchData, but specifically for the WLAN clients
func (e *ZhoneExporter) FetchWirelessData(radios []string) [2]map[string]*goquery.Document {
	var (
		urls    [2]map[string]url.URL
		results [2]map[string]*goquery.Document
	)
	urls[0] = make(map[string]url.URL)
	results[0] = make(map[string]*goquery.Document)
	urls[1] = make(map[string]url.URL)
	results[1] = make(map[string]*goquery.Document)
	for _, value := range radios {
		query := url.Values{}
		query.Set("curRadio", value)
		urls[0][value] = url.URL{Scheme: "http",
			Host:     e.URL,
			Path:     "zhnwlstatus.cmd",
			RawQuery: query.Encode(),
			User:     url.UserPassword(e.username, e.password)}
		query.Set("action", "view")
		urls[1][value] = url.URL{Scheme: "http",
			Host:     e.URL,
			Path:     "zhnwlinfo.cmd",
			RawQuery: query.Encode(),
			User:     url.UserPassword(e.username, e.password)}
	}
	for i := range urls {
		for j := range urls[i] {
			url := urls[i][j]
			res, err := http.Get(url.String())
			if err != nil {
				log.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				log.Fatal(fmt.Sprintf("Status code: %d %s: %s", res.StatusCode, res.Status, url.String()))
			}
			doc, err := goquery.NewDocumentFromReader(res.Body)
			if err != nil {
				log.Fatal(err)
			}
			results[i][j] = doc
		}
	}
	return results
}

func main() {
	username := flag.String("u", "user", "Username")
	password := flag.String("p", "user", "Password")
	listenAddress := flag.String("l", ":2112", "Listen Address")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"Usage: %s [FLAGS...] HOSTNAME_TO_QUERY\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(flag.Args()) == 0 || len(flag.Args()) > 1 {
		log.Fatal("Incorrect arguments passed, see usage.")
	}
	host := flag.Args()[0]
	exporter := NewZhoneExporter(host, *username, *password)
	prometheus.MustRegister(exporter)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(*listenAddress, nil)
	if err != http.ErrServerClosed {
		log.Fatal(err)
		os.Exit(1)
	}
}
