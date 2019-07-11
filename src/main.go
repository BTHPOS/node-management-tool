package main

import (
    "os"
    "fmt"
    "os/exec"
    "time"
    "strconv"
    "strings"
    "encoding/json"
    "github.com/Equanox/gotron"
    "github.com/mitchellh/go-homedir"
    "github.com/schollz/jsonstore"
    "github.com/cavaliercoder/grab"
    "io/ioutil"
    "github.com/pbnjay/memory"
    "github.com/matishsiao/goInfo"
)

// Event that is used to store the json of
// the configuration file
type ConfigEvent struct {
    *gotron.Event
    CONFIG string `json:"config"`
}


// Event used to store the RPC and P2P available ports
// that can be connected to by a new node
type PortsEvent struct {
    *gotron.Event
    RPCPORT int `json:"rpcport"`
    PEERPORT int `json:"p2pport"`
}


// Event used to store system details
// that will be used to determine certain constraints
type SystemStateEvent struct {
    *gotron.Event
    OS string `json:"operatingsystem"`
    DIR string `json:"directory"`
    MEMORY uint64 `json:"memory"`
}


// Event used to relay download progress of
// a particular file being downloaded by the backend
type DownloadProgressEvent struct {
    *gotron.Event
    BytesComplete int64 `json:"bytescomplete"`
    Size int64 `json:"size"`
    Progress float64 `json:"progress"`
}

// Event used to relay download status of
// a particular file being downlaoded by the backend
type DownloadStatusEvent struct {
    *gotron.Event
    WasSuccess bool `json:"wassuccess"`
    WasError bool `json:"waserror"`
}

// Event used to relay node status details
// such as uptime and RPC port to connect to
type CheckNodeEvent struct {
    *gotron.Event
    ALIVE bool `json:"alive"`
    RPCPORT float64 `json:"rpcport"`
}


// Event used to relay whether a particular
// directory or group of files within the directory still exists.
// In this case, the directory is likely the datadir.
type DirectoryExistsEvent struct {
    *gotron.Event
    DIRECTORY string `json:"directory"`
    EXISTS bool `json:"exists"`
}

// Used to represent a crypto address
// that can be saved to a file.
type Address struct {
    ADDRESS string `json:"address"`
}

// Used to represent a crypto WIF
// that can be saved to a file.
type Wif struct {
    WIF string `json:"wif"`
}


// Name of configuration file
const CONFIG_NAME = "config"

// Name of folder that contains configuration
const CONFIG_FOLDER_NAME = "bithereum-node-tool"

// Default P2P Port
const AVAILABLE_P2P_PORT_START = 18553

// Default RPC Port
const AVAILABLE_RPC_PORT_START = 18554

// Path to the executable of this file
var APP_PATH = ""


// Helper function that looks for a float value
// in an array of floats
func floatInSlice(a float64, list []float64) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

// Checks if a full node is running by either observing ports
// that RPC/P2P are using or looking as running tasks depending
// on the operating system that this script is running on.
func isNODERunning(rpcport float64, port float64) bool {

    out, err := exec.Command(
      "ps",
      "aux").CombinedOutput();

      if err != nil {
          return false
      } else {
          output := string(out[:])
          return strings.Contains(output, "-rpcport="+fmt.Sprintf("%f", rpcport))
      }
}

// Checks to see if this system is listning for incoming tcp
// connections on a specified port.
func isPortInUse(port int) bool {

    out, err := exec.Command(
      "netstat",
      "-anp",
      "tcp").CombinedOutput();

    if err != nil {
        return false
    } else {
        output := string(out[:])
        return strings.Contains(output, "::1."+strconv.Itoa(port)) || strings.Contains(output, "*."+strconv.Itoa(port))
    }
}

// Checks for an available port number starting at the port
// number specified and excluding the ports in the excluded ports list.
func checkPortForAvilableFrom(port int, excludeports []float64) int {
   var availablePort = 0
   var _port = port

   for {
      if availablePort == 0 {
          if !isPortInUse(_port) && !floatInSlice(float64(_port), excludeports) {
            availablePort = _port;
          } else if _port >= 65000 {
              break;
          } else {
             _port++
          }
      }  else {
        break;
      }
   }

   return availablePort
}

// Persists configuration file to disk
func saveConfigurations(text string) {
    // Get home directory
    var homeDir, homeErr = homedir.Dir()
    if homeErr != nil {
        panic(homeErr)
    }

    // Make sure our configuration directory exists, if it doesn't create it.
    if _, err := os.Stat(homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME); os.IsNotExist(err) {
        os.MkdirAll(homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME, os.ModePerm)
    }

    // Configuration path
    var configPath = homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME + string(os.PathSeparator) + CONFIG_NAME + ".json"
    ioutil.WriteFile(configPath, []byte(text), 0644)
}

// Retrieves configuration from confif file on disk
func readConfigurations() string {

  var homeDir, homeErr = homedir.Dir()
  if homeErr != nil {
    panic(homeErr)
  }

  var configPath = homeDir + string(os.PathSeparator) + CONFIG_FOLDER_NAME + string(os.PathSeparator) + CONFIG_NAME + ".json"
  var output, readErr = ioutil.ReadFile(configPath)
  if readErr == nil {
      return string(output)
  }
  return ""
}

// Saves address and wif to disk
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

// Downloads the contents of a URL to a specified path
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

// Starts a full node
func startNODE(rpcuser string, rpcpass string, rpcport float64, peerport float64, datadir string) bool {

    var prefixPath = ""
    var extension = ""
    var ospathname = ""

    // Linux values
    ospathname = "linux64"


    _, err := exec.Command(
      prefixPath + "builds"+string(os.PathSeparator)+ospathname+string(os.PathSeparator)+"bin"+string(os.PathSeparator)+"bethd" + extension,
      "-daemon",
      "-rpcuser="+rpcuser,
      "-rpcpassword="+rpcpass,
      "-rpcport="+fmt.Sprintf("%f", rpcport),
      "-port="+fmt.Sprintf("%f", peerport),
      "-datadir="+datadir,
      "-dbcache=100",
      "-maxmempool=10",
      "-maxconnections=10",
      "-prune=550").CombinedOutput();

      if err != nil {
          return false;
      } else {
          return true;
      }
}

// Stops a full node
func stopNODE(rpcuser string, rpcpass string, rpcport float64, peerport float64) bool {

   var prefixPath = ""
   var extension = ""
   var ospathname = ""

   // Linux values
   ospathname = "linux64"

    _, err := exec.Command(
      prefixPath + "builds"+string(os.PathSeparator)+ospathname+string(os.PathSeparator)+"bin"+string(os.PathSeparator)+"beth-cli" + extension,
      "-rpcuser="+rpcuser,
      "-rpcpassword="+rpcpass,
      "-rpcport="+fmt.Sprintf("%f", rpcport),
      "-port="+fmt.Sprintf("%f", peerport),
      "stop").CombinedOutput();

    if err != nil {
          return false;
    } else {
        return true;
    }
}

// Checks if a given file/directory exists
func fileExists(dir string) bool {
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        return false;
    } else {
        return true;
    }
}

// Removes node files within a specified datadir
func removeFilesForNODEIfPossible(datadir string) bool {

    var path = datadir + string(os.PathSeparator);
    os.Remove(path + "db.log")
    os.Remove(path + "debug.log")
    os.Remove(path + "banlist.dat")
    os.Remove(path + "peers.dat")
    os.Remove(path + "wallet.dat")
    os.Remove(path + "fee_estimates.dat")
    os.Remove(path + "mempool.dat")
    os.RemoveAll(path + "blocks")
    os.RemoveAll(path + "chainstate")
    var isDeleted = !fileExists(path + "db.log") && !fileExists(path + "debug.log") && !fileExists(path + "banlist.dat") && !fileExists(path + "peers.dat") && !fileExists(path + "wallet.dat") && !fileExists(path + "fee_estimates.dat") && !fileExists(path + "mempool.dat") && !fileExists(path + "blockst") && !fileExists(path + "chainstate");
    return isDeleted;
}

// Initializes all backend events
func initWindowEvents(window *gotron.BrowserWindow) {

      // Provides general system state details such as RAM and OS
      window.On(&gotron.Event{Event: "system-state"}, func(bin []byte) {
            var data map[string]interface{}
            json.Unmarshal(bin, &data)
            var memory = (memory.TotalMemory()/1000000000)
            gi := goInfo.GetInfo()
            window.Send(&SystemStateEvent{
              Event: &gotron.Event{Event: "system-state"},
              OS: gi.OS,
              DIR: APP_PATH,
              MEMORY: memory})
      })


      // Carries out the removal of a data directory
      window.On(&gotron.Event{Event: "remove-datadir"}, func(bin []byte) {
            var data map[string]interface{}
            json.Unmarshal(bin, &data)
            var datadir = data["datadir"].(string)
            var isDeleted = removeFilesForNODEIfPossible(datadir);
            if (isDeleted) {
                window.Send(&DirectoryExistsEvent{
                  Event: &gotron.Event{Event: "remove-datadir"},
                  DIRECTORY: datadir,
                  EXISTS: !isDeleted})
            }
      })

      // Starts a full node
      window.On(&gotron.Event{Event: "start-node"}, func(bin []byte) {
          var data map[string]interface{}
          json.Unmarshal(bin, &data)
          var rpcuser = data["rpcuser"].(string)
          var rpcpass = data["rpcpass"].(string)
          var rpcport = data["rpcport"].(float64)
          var port  = data["port"].(float64)
          var datadir = data["datadir"].(string)
          startNODE(rpcuser, rpcpass, rpcport, port, datadir);
      })

      // Stops a full node
      window.On(&gotron.Event{Event: "stop-node"}, func(bin []byte) {
          var data map[string]interface{}
          json.Unmarshal(bin, &data)
          var rpcuser = data["rpcuser"].(string)
          var rpcpass = data["rpcpass"].(string)
          var rpcport = data["rpcport"].(float64)
          var port  = data["port"].(float64)
          stopNODE(rpcuser, rpcpass, rpcport, port);
      })

      // Checks to see if a full node is running
      window.On(&gotron.Event{Event: "check-node"}, func(bin []byte) {
          var data map[string]interface{}
          json.Unmarshal(bin, &data)
          var rpcport = data["rpcport"].(float64)
          var port = data["port"].(float64)
          var isRunning = isNODERunning(rpcport, port);
          window.Send(&CheckNodeEvent{
            Event: &gotron.Event{Event: "check-node"},
            ALIVE: isRunning,
            RPCPORT: rpcport})
      })

      // Persists configuration details to disk
      window.On(&gotron.Event{Event: "save-configuration"}, func(bin []byte) {
          var data map[string]interface{}
          json.Unmarshal(bin, &data)
          saveConfigurations(data["configuration"].(string))
      })

      // Retrieves configuration details from disk
      window.On(&gotron.Event{Event: "fetch-configuration"}, func(bin []byte) {
            var configuration = readConfigurations()
            window.Send(&ConfigEvent{
              Event: &gotron.Event{Event: "fetch-configuration"},
              CONFIG: configuration})
      })

      // Persists address and wif to a file on disk
      window.On(&gotron.Event{Event: "save-keys"}, func(bin []byte) {
           var data map[string]interface{}
           json.Unmarshal(bin, &data)

           var address = Address{}
           address.ADDRESS = data["address"].(string)

           var wif = Wif{}
           wif.WIF = data["wif"].(string)

           saveKeys(address, wif, data["path"].(string))
      })

      // Retrieves an available RPC and P2P port
      window.On(&gotron.Event{Event: "fetch-ports"}, func(bin []byte) {
           var data map[string]interface{}
           json.Unmarshal(bin, &data)

           var _usedports = data["usedports"].([]interface{})
           var usedports = make([]float64, len(_usedports))
           for i := range _usedports {
              usedports[i] = _usedports[i].(float64)
           }

           var rpcport = checkPortForAvilableFrom(AVAILABLE_RPC_PORT_START, usedports)
           var p2pport = checkPortForAvilableFrom(AVAILABLE_P2P_PORT_START, usedports)

           window.Send(&PortsEvent{
             Event: &gotron.Event{Event: "fetch-ports"},
             RPCPORT: rpcport,
             PEERPORT: p2pport})
      })
}

func run() {

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

    // Initialize Window Events
    initWindowEvents(window)

    // Open dev tools must be used after window.Start
    // window.OpenDevTools()

    // Wait for the application to close
    <-done
}

func main() {
	 run()
}
