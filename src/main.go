package main

import (
    "log"
    "os"
    "time"
    "strconv"
    "encoding/json"
    "crypto/rand"
    "github.com/Equanox/gotron"
    "github.com/mitchellh/go-homedir"
    "github.com/thanhpk/randstr"
    "github.com/schollz/jsonstore"
    "github.com/cavaliercoder/grab"
    "github.com/janosgyerik/portping"
    "github.com/vsergeev/btckeygenie/btckey"
)

type Node struct {
   ID string `json:"id"`
   IPADDRESS string `json:"ipaddress"`
   RPCUSER string `json:"rpcuser"`
   RPCPASS string `json:"rpcpass"`
   RPCPORT int `json:"rpcport"`
   P2PPORT int `json:"p2pport"`
   DATADIR string `json:"datadir"`
   BTHADDRESS string `json:"bthaddress"`
}

type Config struct {
    ID string `json:"id"`
    EXISTING_NODES []Node `json:"existing_nodes"`
    CREATED_NODES []Node `json:"created_nodes"`
}

type ConfigEvent struct {
    *gotron.Event
    Config Config `json:"config"`
}

type DownloadProgressEvent struct {
    *gotron.Event
    BytesComplete int64 `json:"bytescomplete"`
    Size int64 `json:"size"`
    Progress float64 `json:"progress"`
}

type DownloadStatusEvent struct {
    *gotron.Event
    WasSuccess bool `json:"wassuccess"`
    WasError bool `json:"waserror"`
}

type AddressEvent struct {
    *gotron.Event
    Address string `json:"address"`
    WIF string `json:"wif"`
}

type Address struct {
    ADDRESS string `json:"address"`
}

type Wif struct {
    WIF string `json:"wif"`
}

const CONFIG_NAME = "config"
const CONFIG_FOLDER_NAME = "bithereum-node-tool"
const AVAILABLE_P2P_PORT_START = 18553
const AVAILABLE_RPC_PORT_START = 18554
const AVAILABLE_PORT_TIMEOUT = 2000

func checkPortOpen(port int) bool {
  var address = "127.0.0.1:" + strconv.Itoa(port)
  var err = portping.Ping("tcp", address, 2000)
  return err != nil
}

func checkPortForAvilableFrom(port int) int {
   var availablePort = 0
   var _port = port
   for {
      if availablePort == 0 {
          if checkPortOpen(_port) {
            availablePort = _port;
          } else {
             _port++
          }
      } else {
        break;
      }
   }
   return availablePort
}

func saveConfig(c Config) bool {

  // Set our configuration to the keystore
  ks := new(jsonstore.JSONStore)
  ks.Set("config", c);

  // Get home directory
  homeDir, err := homedir.Dir()
  if err != nil {
      panic(err)
  }

  // Make sure our configuration directory exists, if it doesn't create it.
  if _, err := os.Stat(homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME); os.IsNotExist(err) {
      os.MkdirAll(homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME, os.ModePerm)
  }

  // Save the keystore to our JSON configuration file
  var configPath = homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME + string(os.PathSeparator) + CONFIG_NAME + ".json"
  err = jsonstore.Save(ks, configPath);

  if err != nil {
      return false
  }

  return true
}

func saveKeys(address Address, wif Wif, path string) bool {

  // Set our configuration to the keystore
  ks := new(jsonstore.JSONStore)
  ks.Set("address", address);
  ks.Set("wif", wif);

  // Make sure our configuration directory exists, if it doesn't create it.
  if _, err := os.Stat(path); os.IsNotExist(err) {
      os.MkdirAll(path, os.ModePerm)
  }

  // Save the keystore to our JSON configuration file
  var configPath = path + string(os.PathSeparator) + "bth_keys.json"
  var err = jsonstore.Save(ks, configPath);

  if err != nil {
      return false
  }

  return true
}

func readConfig() Config {

  // Get home directory
  homeDir, err := homedir.Dir()
  if err != nil {
      panic(err)
  }

  // Absolute path of configuration file
  var configPath = homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME + string(os.PathSeparator) + CONFIG_NAME + ".json"

  // Attempt to read the configuration
  ks, err := jsonstore.Open(configPath)
  if err == nil {
      var config Config
      err = ks.Get("config", &config)
      if err == nil {
          return config
      }
  }

  return Config{};
}

func download(url string, path string, window *gotron.BrowserWindow) {

    // Check to see if path exists before proceeding
    if _, err := os.Stat(path); os.IsNotExist(err) {
        window.Send(&DownloadStatusEvent{
          Event: &gotron.Event{Event: "download-status"},
          WasSuccess: false,
          WasError: true})
        return;
    }

    // start file download
    client := grab.NewClient()
    req, _ := grab.NewRequest(path, url)

    // block until HTTP/1.1 GET response is received
    resp := client.Do(req)

    // start UI loop
  	t := time.NewTicker(200 * time.Millisecond)
  	defer t.Stop()

    Loop:
        for {
           select {
          		case <-t.C:
                window.Send(&DownloadProgressEvent{
                    Event: &gotron.Event{Event: "download-progress"},
                    BytesComplete: resp.BytesComplete(),
                    Size: resp.Size,
                    Progress: 100*resp.Progress()})

          		case <-resp.Done:
                break Loop
          	}
        }


    // check for errors
  	if err := resp.Err(); err != nil {
       window.Send(&DownloadStatusEvent{
           Event: &gotron.Event{Event: "download-status"},
           WasSuccess: false,
           WasError: true})
  	} else {
      window.Send(&DownloadStatusEvent{
          Event: &gotron.Event{Event: "download-status"},
          WasSuccess: true,
          WasError: false})
    }
}

func run() {

    // Attempt to read values from the
    // saved configuration file
    config := readConfig();


    // Attempt to read the current config file.
    // If we are unsuccessful at reading the config file,
    // then we'll assume the file does not exist.
    if config.ID == "" || config.CREATED_NODES == nil || config.EXISTING_NODES == nil {
        config.ID = randstr.Hex(16)
        config.CREATED_NODES = []Node{}
        config.EXISTING_NODES = []Node{}
        saveConfig(config)
    }

    // Create a new browser window instance
    window, err := gotron.New("ui")
    if err != nil {
        panic(err)
    }

    // Alter default window size and window title.
    window.WindowOptions.Width = 960
    window.WindowOptions.Height = 800
    window.WindowOptions.Title = "Bithereum Node Tool"

    // Start the browser window.
    // This will establish a golang <=> nodejs bridge using websockets,
    // to control ElectronBrowserWindow with our window object.
    done, err := window.Start()
    if err != nil {
        panic(err)
    }


    window.On(&gotron.Event{Event: "config-fetch"}, func(bin []byte) {
          window.Send(&ConfigEvent{
            Event: &gotron.Event{Event: "config-fetch"},
            Config: config})
    })

    window.On(&gotron.Event{Event: "generate-bth-address"}, func(bin []byte) {
         var priv, _ = btckey.GenerateKey(rand.Reader)
         window.Send(&AddressEvent{
            Event: &gotron.Event{Event: "generate-bth-address"},
            Address: priv.ToAddress(),
            WIF: priv.ToWIF()})
    })

    window.On(&gotron.Event{Event: "createnode-download-presync"}, func(bin []byte) {
         var data map[string]interface{}
         json.Unmarshal(bin, &data)
         download("http://ipv4.download.thinkbroadband.com/50MB.zip", data["path"].(string), window)
    })

    window.On(&gotron.Event{Event: "addnode-done"}, func(bin []byte) {
         log.Println(string(bin));
         var data map[string]interface{}
         json.Unmarshal(bin, &data)
         var node = Node{}
         node.ID = randstr.Hex(16)
         node.IPADDRESS = data["ipaddress"].(string)
         node.RPCUSER = data["rpcusername"].(string)
         node.RPCPASS = data["rpcpassword"].(string)
         node.DATADIR = ""
         node.BTHADDRESS = data["address"].(string)

         var RPCPORT, _ = strconv.Atoi(data["rpcport"].(string))
         var P2PPORT, _ = strconv.Atoi(data["p2pport"].(string))

         node.RPCPORT = RPCPORT
         node.P2PPORT = P2PPORT

         config.EXISTING_NODES = append(config.EXISTING_NODES, node)
         saveConfig(config)
    })

    window.On(&gotron.Event{Event: "createnode-done-networksync"}, func(bin []byte) {
         log.Println(string(bin));
         var data map[string]interface{}
         json.Unmarshal(bin, &data)
         var node = Node{}
         node.ID = randstr.Hex(16)
         node.IPADDRESS = "127.0.0.1"
         node.RPCUSER = "bithereum"
         node.RPCPASS = "bithereum"
         node.RPCPORT = checkPortForAvilableFrom(AVAILABLE_RPC_PORT_START)
         node.P2PPORT = checkPortForAvilableFrom(AVAILABLE_P2P_PORT_START)
         node.DATADIR = data["path"].(string)
         node.BTHADDRESS = data["address"].(string)
         config.CREATED_NODES = append(config.CREATED_NODES, node)
         saveConfig(config)
    })

    window.On(&gotron.Event{Event: "createnode-done-presync"}, func(bin []byte) {
         log.Println(string(bin));
         var data map[string]interface{}
         json.Unmarshal(bin, &data)
         var node = Node{}
         node.ID = randstr.Hex(16)
         node.IPADDRESS = "127.0.0.1"
         node.RPCUSER = "bithereum"
         node.RPCPASS = "bithereum"
         node.RPCPORT = checkPortForAvilableFrom(AVAILABLE_RPC_PORT_START)
         node.P2PPORT = checkPortForAvilableFrom(AVAILABLE_P2P_PORT_START)
         node.DATADIR = data["path"].(string)
         node.BTHADDRESS = data["address"].(string)
         config.CREATED_NODES = append(config.CREATED_NODES, node)
         saveConfig(config)
    })

    window.On(&gotron.Event{Event: "save-keys"}, func(bin []byte) {
         log.Println(string(bin));
         var data map[string]interface{}
         json.Unmarshal(bin, &data)

         var address = Address{}
         address.ADDRESS = data["address"].(string)

         var wif = Wif{}
         wif.WIF = data["wif"].(string)

         saveKeys(address, wif, data["path"].(string))
    })

    // Open dev tools must be used after window.Start
    window.OpenDevTools()

    // Wait for the application to close
    <-done
}

func main() {
	 run()
}
