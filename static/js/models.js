window.Models = window.Models || {};
window.Collections = window.Collections || {};

(function() {
	Models.User = Backbone.Model.extend({
		urlRoot: '/api/users',
  		parse: function(response, options) {
			resp = _.clone(response);

			if(resp.timestamp) {
				resp.timestamp = new Date(resp.timestamp);
			}
			return resp;
		}
	});

	Collections.Users = Backbone.Collection.extend({
		model: Models.User,
		url: 'api/users' /* Note relative path */
	});
})();
