'use strict';

angular.module('Application', [
  'ngRoute',
  'ngWebSocket',
  'Application.Controllers'
]).
config(['$locationProvider', '$interpolateProvider', '$routeProvider', function($locationProvider, $interpolateProvider, $routeProvider) {
}])

.run(['$rootScope', '$anchorScroll', function($rootScope, $anchorScroll) {
}]);
