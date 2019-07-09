angular.module('Application.Controllers', [])

.controller("BNTController", ["$scope", "$timeout", "$rootScope", "$websocket", function($scope, $timeout, $rootScope, $websocket) {


      Promise.delay = function(t, val) {
          return new Promise(resolve => {
              setTimeout(resolve.bind(null, val), t);
          });
      }

      Promise.raceAll = function(promises, timeoutTime, timeoutVal) {
          return Promise.all(promises.map(p => {
              return Promise.race([p, Promise.delay(timeoutTime, timeoutVal)])
          }));
      }

      var pollInterval = 3000;
      var pollRunningInterval = 5000;

      $scope.nodecreatedlimit = 1;
      $scope.systemstate = {};

      $scope.isWaitingIndicatorVisible = false;
      var showWaitingIndicator = function() {
          $timeout(function() {
              $scope.isWaitingIndicatorVisible = true;
          });
      };
      var hideWaitingIndicator = function() {
          $timeout(function() {
              $scope.isWaitingIndicatorVisible = false;
          }, 1000);
      };

      var configuration = {
          created_nodes: [],
          existing_nodes: [],
          removed_nodes: [],
          cache: {
              welcomeseen: false
          }
      };
      $scope.configuration = configuration;

      var poll_getinfo = function() {
            var requests = [];
            for (var index in $scope.configuration.created_nodes) {
                var node = $scope.configuration.created_nodes[index];
                requests.push(post(
                    "http://"+node.rpcusername+":"+node.rpcpassword+"@"+node.ipaddress+":"+node.rpcport,
                    {"jsonrpc": "1.0","method": "getinfo","params": [],"id": 1}
                ))
            }
            for (var index in $scope.configuration.existing_nodes) {
                var node = $scope.configuration.existing_nodes[index];
                requests.push(post(
                    "http://"+node.rpcusername+":"+node.rpcpassword+"@"+node.ipaddress+":"+node.rpcport,
                    {"jsonrpc": "1.0","method": "getinfo","params": [],"id": 1}
                ))
            }
            Promise.raceAll( requests, 3000, null ).then(function() {
                  let responses = arguments[0];
                  for (var nodeIndex in responses) {
                        var data = responses[nodeIndex];
                        var nodetype = nodeIndex >= $scope.configuration.created_nodes.length ? "existing_nodes" : "created_nodes";
                        (function( _data, _nodeIndex, _nodetype ) {
                            $timeout(function() {
                                  _nodeIndex = _nodetype == "existing_nodes" ? _nodeIndex-$scope.configuration.created_nodes.length : _nodeIndex;
                                  if ($scope.configuration[_nodetype][_nodeIndex]) {
                                      if (_nodetype === "existing_nodes") {
                                          if (_data) {
                                              $scope.configuration[_nodetype][_nodeIndex].nodestats_isrunning = true;
                                          }
                                          else {
                                              $scope.configuration[_nodetype][_nodeIndex].nodestats_isrunning = false;
                                          }
                                      }
                                      $scope.configuration[_nodetype][_nodeIndex].nodestats_getinfo = _data ? JSON.parse(_data.response).result : {}
                                  }
                              })
                        })( data, nodeIndex, nodetype );
                  }
                  setTimeout(poll_getinfo, pollInterval)
            });
      };

      var poll_running = function() {
            for (var index in $scope.configuration.created_nodes) {
              let node = $scope.configuration.created_nodes[index];
              ws.send(JSON.stringify({"event": "check-node", "rpcport": node.rpcport}))
            }
            setTimeout(poll_running, pollRunningInterval);
      };

      var poll = function() {
          poll_getinfo();
          poll_running();
          poll_checks();
      };

      var poll_checks = function() {
          if ($scope.configuration.removed_nodes.length > 0) {
              var removed_nodes = $scope.configuration.removed_nodes;
              for (var index in removed_nodes) {
                  ws.send(JSON.stringify({"event": "remove-datadir", "datadir": removed_nodes[index].datadir}))
              }
          }
          setTimeout(poll_checks, pollInterval)
      };

      var checkNODE = function(rpcuser, rpcpass, rpcport, port) {
          ws.send(JSON.stringify({"event": "check-node", "rpcuser": rpcuser, "rpcpass": rpcpass, "rpcport": rpcport, "port": port}))
      };

      var startNODE = function(rpcuser, rpcpass, rpcport, port, datadir) {
          ws.send(JSON.stringify({"event": "start-node", "rpcuser": rpcuser, "rpcpass": rpcpass, "rpcport": rpcport, "port": port, "datadir": datadir}))
      };

      var stopNODE = function(rpcuser, rpcpass, rpcport, port) {
           ws.send(JSON.stringify({"event": "stop-node", "rpcuser": rpcuser, "rpcpass": rpcpass, "rpcport": rpcport,  "port": port}))
      };

      var saveConfiguration = function() {
            var config = angular.toJson($scope.configuration);
            ws.send(JSON.stringify({"event": "save-configuration", "configuration": config}))
      };

      var fetchConfiguration = function() {
          showWaitingIndicator();
          ws.send(JSON.stringify({"event": "fetch-configuration"}))
      };

      var fetchSystemState = function() {
          ws.send(JSON.stringify({"event": "system-state"}))
      };

      var findRemovedNodeIndexByDatadir = function(datadir) {
            var created = $scope.configuration.created_nodes;
            var existing = $scope.configuration.existing_nodes;
            for (var index in created) {
                if (created[index].datadir == datadir) {
                    return index;
                }
            }
            for (var index in existing) {
                if (existing[index].datadir == datadir) {
                    return index;
                }
            }
            return -1;
      };

      var findNodeByRPC = function(rpcport) {
            var created = $scope.configuration.created_nodes;
            var existing = $scope.configuration.existing_nodes;
            for (var index in created) {
                if (created[index].rpcport == rpcport) {
                    return created[index];
                }
            }
            for (var index in existing) {
                if (existing[index].rpcport == rpcport) {
                    return existing[index];
                }
            }
            return false;
      };

      var fetchPorts = function() {
          var used_rpcports = $scope.configuration.created_nodes.map(function(node) {
              return node.rpcport;
          });
          var used_p2pports = $scope.configuration.created_nodes.map(function(node) {
              return node.p2pport;
          });
          var usedports = used_rpcports.concat(used_p2pports);
          ws.send(JSON.stringify({"event": "fetch-ports", "usedports": usedports}))
      };

      var handleInitState = function() {

          // Welcome screen
          if (!$scope.configuration.cache.welcomeseen) {
              popup.welcome.isVisible = true;
              $scope.configuration.cache.welcomeseen = true;
              saveConfiguration();
          }

          // for (var index in $scope.configuration.created_nodes) {
          //       var node = $scope.configuration.created_nodes[index];
          //       ws.send(JSON.stringify({
          //             "event": "start-node",
          //             "rpcuser": node.rpcusername,
          //             "rpcpass": node.rpcpassword,
          //             "rpcport": node.rpcport,
          //             "port": node.p2pport,
          //             "datadir": node.datadir,
          //       }))
          // }
      };

      var validator = {};
      validator.isValidIPADDRESS = function(value) {
          var ipformat = /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
          return (value && value.match(ipformat))
      };
      validator.isValidRPCPORT = function(value) {
          return value && !isNaN(parseInt(value));
      };
      validator.isValidP2PPORT = function(value) {
          return value && !isNaN(parseInt(value));
      };
      validator.isValidRPCUSERNAME = function(value) {
          return value && value.length > 0;
      };
      validator.isValidRPCPASSWORD = function(value) {
          return value && value.length > 0;
      };
      validator.isValidBTHADDRESS = function(value) {
        try {
             let decoded = bitcoin.address.fromBase58Check(value);
             return decoded["version"] == 25;
        } catch(e) {}
        return false;
      };
      validator.isValidExistingNode = function(fields) {
          return (
              validator.isValidIPADDRESS(fields.ipaddress) &&
              validator.isValidBTHADDRESS(fields.address) &&
              validator.isValidRPCPORT(fields.rpcport) &&
              validator.isValidP2PPORT(fields.p2pport) &&
              validator.isValidRPCUSERNAME(fields.rpcusername) &&
              validator.isValidRPCPASSWORD(fields.rpcpassword)
          )
      };
      $scope.validator = validator;

      // Estbalish backend connection
      let ws = $websocket("ws://localhost:" + global.backendPort + "/web/app/events");

      var scrollTop = function() {
          window.scrollTo(0, 0);
      };

      // Popups
      var popup = {};

      //////////////////////
      // Popup: Welcoming //
      //////////////////////
      popup.welcome = {};
      popup.welcome.isVisible = false;
      popup.welcome.continue = function() {
          this.isVisible = false;
      };
      popup.welcome.show = function() {
        this.isVisible = true;
      };
      popup.welcome.hide = function() {
        this.isVisible = true;
      };

      ///////////////////////////////////////
      // Popup: Node Created limit reached //
      ///////////////////////////////////////
      popup.nodecreatedlimit = {};
      popup.nodecreatedlimit.isVisible = false;
      popup.nodecreatedlimit.continue = function() {
          this.isVisible = false;
      };
      popup.nodecreatedlimit.show = function() {
        this.isVisible = true;
      };
      popup.nodecreatedlimit.hide = function() {
        this.isVisible = true;
      };

      ////////////////////////
      // Popup: Remove Node //
      ////////////////////////
      popup.removenode = {};
      popup.removenode.isVisible = false;
      popup.removenode.fields = {}
      popup.removenode.yes = function() {
          if (this.fields.nodegroup === "created_nodes") {
              var node = $scope.configuration[this.fields.nodegroup][this.fields.nodeindex];
              ws.send(JSON.stringify({
                    "event": "stop-node",
                    "rpcuser": node.rpcusername,
                    "rpcpass": node.rpcpassword,
                    "rpcport": node.rpcport,
                    "port": node.p2pport,
                    "datadir": node.datadir,
              }))
          }
          let nodeRemoved = $scope.configuration[this.fields.nodegroup].splice(this.fields.nodeindex, 1)[0];
          if (nodeRemoved && this.fields.nodegroup === "created_nodes") {
              $scope.configuration.removed_nodes.push(nodeRemoved);
          }
          saveConfiguration();
          this.hide();
      };
      popup.removenode.no = function() {
          this.hide();
      };
      popup.removenode.show = function(nodeindex, nodegroup) {
          this.isVisible = true;
          this.fields.nodeindex = nodeindex;
          this.fields.nodegroup = nodegroup;
          scrollTop();
      };
      popup.removenode.hide = function() {
          this.isVisible = false;
          this.fields = {}
      };

      // Popup: Edit Node
      popup.editnode = {};
      popup.editnode.page = 1;
      popup.editnode.fields = {};
      popup.editnode.isVisible = false;
      popup.editnode.update = function() {
          var node = {};
          node["ipaddress"] = popup.editnode.fields.ipaddress
          node["rpcport"] = popup.editnode.fields.rpcport
          node["p2pport"] = popup.editnode.fields.p2pport
          node["rpcusername"] = popup.editnode.fields.rpcusername
          node["rpcpassword"] = popup.editnode.fields.rpcpassword
          node["address"] = popup.editnode.fields.address
          $scope.configuration[this.nodetype][this.index] = node;
          saveConfiguration();
          this.hide();
      };
      popup.editnode.startNode = function() {
          ws.send(JSON.stringify({
              "event": "start-node",
              "rpcuser": this.fields.rpcusername,
              "rpcpass": this.fields.rpcpassword,
              "rpcport": this.fields.rpcport,
              "port": this.fields.p2pport,
              "datadir": this.fields.datadir,
          }))
          this.hide();
      };
      popup.editnode.stopNode = function() {
          ws.send(JSON.stringify({
              "event": "stop-node",
              "rpcuser": this.fields.rpcusername,
              "rpcpass": this.fields.rpcpassword,
              "rpcport": this.fields.rpcport,
              "port": this.fields.p2pport,
              "datadir": this.fields.datadir,
          }))
          this.hide();
      };
      popup.editnode.show = function(node, index, type) {
          this.isVisible = true;
          this.node = node;
          this.fields = JSON.parse(JSON.stringify(node));
          this.nodetype = type;
          this.index = index;
          scrollTop();
      };
      popup.editnode.hide = function() {
          this.isVisible = false;
          this.page = 1;
          this.fields = {};
      };

      // Popup: Add Node
      popup.addnode = {};
      popup.addnode.page = 1;
      popup.addnode.fields = {};
      popup.addnode.isVisible = false;
      popup.addnode.next = function(page) {
          if (page) this.page = page;
          else this.page++;
      };
      popup.addnode.back = function(page) {
         if (page > 0) this.page = page;
         else this.page > 1 ? --this.page : false;
      };
      popup.addnode.progressIfCanCreate = function() {
        if ($scope.configuration.created_nodes.length == $scope.nodecreatedlimit) {
            this.hide();
            popup.nodecreatedlimit.show();
        } else {
          this.page = 2;
        }
      };
      popup.addnode.addExisting = function() {
          this.page = 5;
          var node = {}
          node["datadir"] = popup.addnode.fields.location
          node["ipaddress"] = popup.addnode.fields.ipaddress
          node["rpcport"] = popup.addnode.fields.rpcport
          node["p2pport"] = popup.addnode.fields.p2pport
          node["rpcusername"] = popup.addnode.fields.rpcusername
          node["rpcpassword"] = popup.addnode.fields.rpcpassword
          node["address"] = popup.addnode.fields.address
          $scope.configuration.existing_nodes.push(node);
          saveConfiguration();
      };
      popup.addnode.createNode = function() {
            this.page = 5;
            var node = {}
            node["datadir"] = popup.addnode.fields.location
            node["ipaddress"] = "127.0.0.1"
            node["rpcport"] = popup.addnode.fields.rpcport
            node["p2pport"] = popup.addnode.fields.p2pport
            node["rpcusername"] = randomString(10)
            node["rpcpassword"] = randomString(10)
            node["address"] = popup.addnode.fields.address
            $scope.configuration.created_nodes.push(node);
            saveConfiguration();
            startNODE(node["rpcusername"],node["rpcpassword"],node["rpcport"],node["p2pport"],node["datadir"])
      };
      popup.addnode.show = function() {
          this.isVisible = true;
          fetchPorts();
          scrollTop();
      };
      popup.addnode.hide = function() {
          this.isVisible = false;
          this.page = 1;
          this.fields = {};
      };
      popup.addnode.chooseLocation = function() {
          document.getElementById("node-location").click();
      };
      popup.addnode.onChooseLocation = function(event) {
          let path = document.getElementById("node-location").files[0].path;
          document.getElementById("node-location-form").reset();
          $timeout(function() {
              popup.addnode.fields.location = path;
              popup.addnode.next();
          });
      };
      popup.addnode.chooseKeySaveLocation = function() {
          document.getElementById("node-keys").click();
      };
      popup.addnode.onChooseKeySaveLocation = function(event) {
          let path = document.getElementById("node-keys").files[0].path;
          document.getElementById("node-keys-form").reset();
          popup.addnode.fields.keysLocation = path;
          ws.send(JSON.stringify({
              "event": "save-keys",
              "address" : popup.addnode.fields.address,
              "wif": popup.addnode.fields.wif,
              "path": path
          }))
          alert("Your keys have been successfully downloaded to your computer!");
      };
      popup.addnode.generateBTHAddress = function() {
          let keyPair = bitcoin.ECPair.makeRandom()
          let address = keyPair.getAddress().toString()
          let wif = keyPair.toWIF()

          let decoded = bitcoin.address.fromBase58Check(address);
          let addressBTH = bitcoin.address.toBase58Check(decoded['hash'], 25);
          popup.addnode.fields.address = addressBTH
          popup.addnode.fields.wif = wif
      };

      $scope.$watch("popup.addnode.fields.address", function() {
            if (popup.addnode.fields.address && popup.addnode.fields.wif) {
                popup.addnode.fields.address.wif = ""
            }
      });

      // Websocket event handlers
      ws.onMessage(function(message) {
          let data = JSON.parse(message.data);
          if (data.event === "fetch-configuration") {
              $timeout(function() {
                  hideWaitingIndicator();
                  $scope.configuration = data.config ? JSON.parse(data.config) : configuration;
                  if (!$scope.configuration.cache) {
                      $scope.configuration.cache = configuration.cache;
                      saveConfiguration();
                  }
                  handleInitState();
                  poll();
                  fetchSystemState();
              })
          }
          if (data.event === "fetch-ports") {
              $timeout(function() {
                  popup.addnode.fields.rpcport = data.rpcport;
                  popup.addnode.fields.p2pport = data.p2pport;
              })

          }
          if (data.event === "check-node") {
              $timeout(function() {
                  var node = findNodeByRPC(data.rpcport)
                  if (node) node.nodestats_isrunning = data.alive;
              })
          }
          if (data.event === "system-state") {
              $timeout(function() {
                  if (data) {
                      $scope.systemstate = data;
                      $scope.nodecreatedlimit = parseInt(data.memory/2);
                  }
              })
          }
          if (data.event === "remove-datadir") {
              $timeout(function() {
                  var index = findRemovedNodeIndexByDatadir(data.directory);
                  $scope.configuration.removed_nodes.splice(index, 1);
                  saveConfiguration();
              })
          }
      });

      // Fetch configurations from disk
      fetchConfiguration();

      // Set popup
      $scope.popup = popup;


}])
