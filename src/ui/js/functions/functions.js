
var post = function(url, data, headers) {
  return new Promise(function(resolve, reject) {
          data = data || {}
          var xhr = new XMLHttpRequest();
          try {
              xhr.open("POST", url, true);
              for (var header in headers) {
                  xhr.setRequestHeader(header, headers[header]);
              }
              xhr.onreadystatechange = function() {
                  if (this.readyState == 4 && this.status == 200) {
                      resolve(xhr);
                  }
              };
              xhr.onload = function() {
                  if (xhr.status != 200) {
                      resolve();
                  }
              };
              xhr.onerror = function() {
                  resolve();
              }
              xhr.send(JSON.stringify(data));
          }
          catch(e) {
              resolve();
          }
  });
}

var randomString = function(length) {
    return Math.round((Math.pow(36, length + 1) - Math.random() * Math.pow(36, length))).toString(36).slice(1);
};
