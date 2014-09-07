window.Models = window.Models || {};

(function(){
	window.Models.User = Backbone.Model.extend({
		urlRoot : '/api/users',
		defaults: {
			"name": "",
			"picture": ""
		}
	});
})();
