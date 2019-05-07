angular.module('Application.Controllers', [])

.controller("BNTController", ["$scope", "$timeout", "$rootScope", "$websocket", function($scope, $timeout, $rootScope, $websocket) {

      // Estbalish backend connection
      let ws = $websocket("ws://localhost:" + global.backendPort + "/web/app/events");

      // Popups
      var popup = {};

      // Popup: Welcoming
      popup.welcome = {};
      popup.welcome.isVisible = false;
      popup.welcome.continue = function() {
          this.isVisible = false;
      };

      // Popup: Add Node
      popup.addnode = {};
      popup.addnode.page = 4;
      popup.addnode.fields = {};
      popup.addnode.isVisible = true;
      popup.addnode.next = function(page) {
          if (page) this.page = page;
          else this.page++;
      };
      popup.addnode.back = function(page) {
         if (page > 0) this.page = page;
         else this.page > 1 ? --this.page : false;
      };
      popup.addnode.preSync = function() {
         this.fields.isPreSync = true;
         this.next(8);
      };
      popup.addnode.networkSync = function() {
          this.fields.isPreSync = false;
          this.next();
      };
      popup.addnode.sync = function() {
          if (this.fields.isPreSync) {
              this.page = 9;
              this.fields.downloadProgress = {};
              this.fields.downloadStatus = {};
              ws.send(JSON.stringify({
                 "event": "createnode-download-presync",
                 "path": popup.addnode.fields.location
              }))
          }
          else {
              this.createNode();
          }
      };
      popup.addnode.addExisting = function() {
          if (/^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/.test(popup.addnode.fields.ipaddress) != true) {
              return alert("Invalid IP Address format")
          }
          if (!popup.addnode.fields.rpcport || isNaN(parseInt(popup.addnode.fields.rpcport))) {
              return alert("Enter a valid RPC Port")
          }
          if (!popup.addnode.fields.p2pport || isNaN(parseInt(popup.addnode.fields.p2pport))) {
              return alert("Enter a valid P2P Port")
          }
          if (!popup.addnode.fields.rpcusername) {
              return alert("Enter a RPC Username")
          }
          if (!popup.addnode.fields.rpcpassword) {
              return alert("Enter a RPC Password")
          }
          this.page = 10;
          var data = {}
          data["event"] = "addnode-done"
          data["path"] = popup.addnode.fields.location
          data["ipaddress"] = popup.addnode.fields.ipaddress
          data["rpcport"] = popup.addnode.fields.rpcport
          data["p2pport"] = popup.addnode.fields.p2pport
          data["rpcusername"] = popup.addnode.fields.rpcusername
          data["rpcpassword"] = popup.addnode.fields.rpcpassword
          data["address"] = popup.addnode.fields.address
          ws.send(JSON.stringify(data))
      };
      popup.addnode.createNode = function() {
          this.page = 10;
          if (this.fields.isPreSync) {
            ws.send(JSON.stringify({
               "event": "createnode-done-presync",
               "path": popup.addnode.fields.location,
               "address": popup.addnode.fields.address
            }))
          }
          else {
            ws.send(JSON.stringify({
               "event": "createnode-done-networksync",
               "path": popup.addnode.fields.location,
               "address": popup.addnode.fields.address
            }))
          }
      };
      popup.addnode.show = function() {
          this.isVisible = true;
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
          this.fields.hasCreatedAddress = true;
          ws.send(JSON.stringify({"event": "generate-bth-address"}))
      };
      popup.addnode.isValidBTHAddress = function() {
          try {
            let decoded = bitcoin.address.fromBase58Check(popup.addnode.fields.address);
            return decoded["version"] == 25;
          } catch(e) {}

          return false;
      };

      // Websocket event handlers
      ws.onMessage(function(message) {
          let data = JSON.parse(message.data);
          if (data.event === "download-progress") {
              $timeout(function() {
                  popup.addnode.fields.downloadProgress = data;
              });
          }
          if (data.event === "download-status") {
              $timeout(function() {
                  popup.addnode.fields.downloadStatus = data;
                  if (data.wassuccess && popup.addnode.fields.downloadProgress) {
                      popup.addnode.fields.downloadProgress.progress = 100;
                  }
              });
          }
          if (data.event === "config-fetch") {
              $timeout(function() {
                   $scope.config = data.config;
              });
          }
          if (data.event === "generate-bth-address") {
              $timeout(function() {
                  var decoded = bitcoin.address.fromBase58Check(data.address);
                  data.address = bitcoin.address.toBase58Check(decoded['hash'], 25);
                  popup.addnode.fields.address = data.address
                  popup.addnode.fields.wif = data.wif
                  popup.addnode.fields.hasCreatedAddress = true;
              });
          }
      });

      ws.send(JSON.stringify({"event": "config-fetch"}))

      // Make popup module accessible to current scope
      $scope.popup = popup;
}])
