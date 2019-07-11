package main

import (
    "log"
    "os"
    "runtime"
    "fmt"
    "os/exec"
    "time"
    "strconv"
    "strings"
    "encoding/json"
    "crypto/rand"
    "github.com/Equanox/gotron"
    "github.com/mitchellh/go-homedir"
    "github.com/schollz/jsonstore"
    "github.com/cavaliercoder/grab"
    "github.com/vsergeev/btckeygenie/btckey"
    "io/ioutil"
    "github.com/pbnjay/memory"
    "github.com/matishsiao/goInfo"
)

type ConfigEvent struct {
    *gotron.Event
    CONFIG string `json:"config"`
}

type PortsEvent struct {
    *gotron.Event
    RPCPORT int `json:"rpcport"`
    PEERPORT int `json:"p2pport"`
}

type SystemStateEvent struct {
    *gotron.Event
    OS string `json:"operatingsystem"`
    DIR string `json:"directory"`
    MEMORY uint64 `json:"memory"`
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

type CheckNodeEvent struct {
    *gotron.Event
    ALIVE bool `json:"alive"`
    RPCPORT float64 `json:"rpcport"`
}

type AddressEvent struct {
    *gotron.Event
    Address string `json:"address"`
    WIF string `json:"wif"`
}

type DirectoryExistsEvent struct {
    *gotron.Event
    DIRECTORY string `json:"directory"`
    EXISTS bool `json:"exists"`
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
var APP_PATH = ""

func setInterval(someFunc func(), milliseconds int, async bool) chan bool {

    // How often to fire the passed in function
    // in milliseconds
    interval := time.Duration(milliseconds) * time.Millisecond

    // Setup the ticket and the channel to signal
    // the ending of the interval
    ticker := time.NewTicker(interval)
    clear := make(chan bool)

    // Put the selection in a go routine
    // so that the for loop is none blocking
    go func() {
        for {

            select {
            case <-ticker.C:
                if async {
                    // This won't block
                    go someFunc()
                } else {
                    // This will block
                    someFunc()
                }
            case <-clear:
                ticker.Stop()
                return
            }

        }
    }()

    // We return the channel so we can pass in
    // a value to it to clear the interval
    return clear

}

func setTimeout(someFunc func(), milliseconds int) {

    timeout := time.Duration(milliseconds) * time.Millisecond

    // This spawns a goroutine and therefore does not block
    time.AfterFunc(timeout, someFunc)

}

func floatInSlice(a float64, list []float64) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func isNODERunning(rpcport float64, port float64) bool {

    if runtime.GOOS == "windows" {
        return isPortInUse(int(port))
    } else {
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
}

func isPortInUse(port int) bool {

    // here we perform the pwd command.
    // we can store the output of this in our out variable
    // and catch any errors in err
    out, err := exec.Command(
      "netstat",
      "-anp",
      "tcp").CombinedOutput();

    // if there is an error with our execution
    // handle it here
    if err != nil {
        log.Println(err)
        return false
    } else {
        // as the out variable defined above is of type []byte we need to convert
        // this to a string or else we will see garbage printed out in our console
        // this is how we convert it to a string
        output := string(out[:])
        log.Println(output)
        if runtime.GOOS == "windows" {
          return strings.Contains(output, "0.0.0.0:"+strconv.Itoa(port))
        } else {
          return strings.Contains(output, "::1."+strconv.Itoa(port)) || strings.Contains(output, "*."+strconv.Itoa(port))
        }
    }
}

func checkPortForAvilableFrom(port int, ports []float64) int {
   var availablePort = 0
   var _port = port

   for {
      if availablePort == 0 {
          if !isPortInUse(_port) && !floatInSlice(float64(_port), ports) {
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

func readConfigurations() string {
  // Get home directory
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

func startNODE(rpcuser string, rpcpass string, rpcport float64, peerport float64, datadir string) bool {

    var prefixPath = ""
    var extension = ""
    var ospathname = ""

    if runtime.GOOS == "windows" {
        extension = ".exe"
        ospathname = "windows32"
    } else if runtime.GOOS == "darwin" {
        prefixPath = APP_PATH + string(os.PathSeparator)
        ospathname = "macos64"
    } else {
        ospathname = "linux64"
    }

    // here we perform the pwd command.
    // we can store the output of this in our out variable
    // and catch any errors in err
    if runtime.GOOS == "windows" {

        log.Println("Run node for windows")
        log.Println( prefixPath + "builds"+string(os.PathSeparator)+ospathname+string(os.PathSeparator)+"bin"+string(os.PathSeparator)+"bethd" + extension )

        out, err := exec.Command(
          "powershell",
          "start-process",
          prefixPath + "builds"+string(os.PathSeparator)+ospathname+string(os.PathSeparator)+"bin"+string(os.PathSeparator)+"bethd" + extension,
          "-ArgumentList",
          "'-datadir="+datadir+" -rpcuser="+rpcuser+" -rpcpassword="+rpcpass+" -rpcport="+fmt.Sprintf("%f", rpcport)+" -port="+fmt.Sprintf("%f", peerport)+" -rpcallowip=0.0.0.0/0 -dbcache=100 -maxmempool=10 -maxconnections=10 -prune=550'",
          "-WindowStyle",
          "Hidden").CombinedOutput();

          if err != nil {
              log.Println(err)
              return false;
          } else {
              output := string(out[:])
              log.Println(output)
              return true;
          }

          return true;

    } else {
        out, err := exec.Command(
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
              log.Println(err)
              return false;
          } else {
              output := string(out[:])
              log.Println(output)
              return true;
          }
    }
}

func stopNODE(rpcuser string, rpcpass string, rpcport float64, peerport float64) bool {

  var prefixPath = ""
  var extension = ""
  var ospathname = ""

  if runtime.GOOS == "windows" {
      extension = ".exe"
      ospathname = "windows32"
  } else if runtime.GOOS == "darwin" {
      prefixPath = APP_PATH + string(os.PathSeparator)
      ospathname = "macos64"
  } else {
      ospathname = "linux64"
  }

    // here we perform the pwd command.
    // we can store the output of this in our out variable
    // and catch any errors in err
    _, err := exec.Command(
      prefixPath + "builds"+string(os.PathSeparator)+ospathname+string(os.PathSeparator)+"bin"+string(os.PathSeparator)+"beth-cli" + extension,
      "-rpcuser="+rpcuser,
      "-rpcpassword="+rpcpass,
      "-rpcport="+fmt.Sprintf("%f", rpcport),
      "-port="+fmt.Sprintf("%f", peerport),
      "stop").CombinedOutput();
    // if there is an error with our execution
    // handle it here
    if err != nil {
          // UNABLE TO STOP NODE
          return false;
    } else {
        // as the out variable defined above is of type []byte we need to convert
        // this to a string or else we will see garbage printed out in our console
        // this is how we convert it to a string
        // output := string(out[:])
        // log.Println(output)
        return true;
    }
}

func directoryExists(dir string) bool {
    if _, err := os.Stat(dir); os.IsNotExist(err) {
        return false;
    } else {
        return true;
    }
}

func removeDirForNODEIfPossible(dir string) bool {

    var path = dir + string(os.PathSeparator);
    os.Remove(path + "db.log")
    os.Remove(path + "debug.log")
    os.Remove(path + "banlist.dat")
    os.Remove(path + "peers.dat")
    os.Remove(path + "wallet.dat")
    os.Remove(path + "fee_estimates.dat")
    os.Remove(path + "mempool.dat")
    os.RemoveAll(path + "blocks")
    os.RemoveAll(path + "chainstate")
    var isDeleted = !directoryExists(path + "db.log") && !directoryExists(path + "debug.log") && !directoryExists(path + "banlist.dat") && !directoryExists(path + "peers.dat") && !directoryExists(path + "wallet.dat") && !directoryExists(path + "fee_estimates.dat") && !directoryExists(path + "mempool.dat") && !directoryExists(path + "blockst") && !directoryExists(path + "chainstate");
    return isDeleted;
}

func initWindowEvents(window *gotron.BrowserWindow) {

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

      window.On(&gotron.Event{Event: "remove-datadir"}, func(bin []byte) {
            var data map[string]interface{}
            json.Unmarshal(bin, &data)
            var datadir = data["datadir"].(string)
            var isDeleted = removeDirForNODEIfPossible(datadir);
            if (isDeleted) {
                window.Send(&DirectoryExistsEvent{
                  Event: &gotron.Event{Event: "remove-datadir"},
                  DIRECTORY: datadir,
                  EXISTS: !isDeleted})
            }
      })

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

      window.On(&gotron.Event{Event: "stop-node"}, func(bin []byte) {
          var data map[string]interface{}
          json.Unmarshal(bin, &data)
          var rpcuser = data["rpcuser"].(string)
          var rpcpass = data["rpcpass"].(string)
          var rpcport = data["rpcport"].(float64)
          var port  = data["port"].(float64)
          stopNODE(rpcuser, rpcpass, rpcport, port);
      })

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

      window.On(&gotron.Event{Event: "save-configuration"}, func(bin []byte) {
          var data map[string]interface{}
          json.Unmarshal(bin, &data)
          saveConfigurations(data["configuration"].(string))
      })

      window.On(&gotron.Event{Event: "fetch-configuration"}, func(bin []byte) {
            var configuration = readConfigurations()
            window.Send(&ConfigEvent{
              Event: &gotron.Event{Event: "fetch-configuration"},
              CONFIG: configuration})
      })

      window.On(&gotron.Event{Event: "generate-bth-address"}, func(bin []byte) {
           var priv, _ = btckey.GenerateKey(rand.Reader)
           window.Send(&AddressEvent{
              Event: &gotron.Event{Event: "generate-bth-address"},
              Address: priv.ToAddress(),
              WIF: priv.ToWIF()})
      })

      window.On(&gotron.Event{Event: "save-keys"}, func(bin []byte) {
           var data map[string]interface{}
           json.Unmarshal(bin, &data)

           var address = Address{}
           address.ADDRESS = data["address"].(string)

           var wif = Wif{}
           wif.WIF = data["wif"].(string)

           saveKeys(address, wif, data["path"].(string))
      })

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

    // Set APP PATH
    if runtime.GOOS == "darwin" {
        APP_PATH = os.Args[1]
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
