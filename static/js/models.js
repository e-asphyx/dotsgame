window.Models = window.Models || {};
window.Collections = window.Collections || {};

(function() {
	Models.User = Backbone.Model.extend({
		urlRoot: 'api/users',

		/*
		url: function() {
			// Give more priority to collection's url
      		var base = _.result(this.collection, 'url') || _.result(this, 'urlRoot');

      		if (this.isNew()) return base;
			return base.replace(/([^\/])$/, '$1/') + encodeURIComponent(this.id);
    	},
		*/

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
