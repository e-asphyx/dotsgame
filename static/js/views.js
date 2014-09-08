window.Views = window.Views || {};
window.Controllers = window.Controllers || {};

(function(){
	window.Views.UserProfile = Backbone.View.extend({
		initialize: function() {
			this.template = _.template($("#user-profile-template").html()); /* DOM must be ready */
			this.listenTo(this.model, 'change', this.render);
		},

		render: function() {
			this.$el.html(this.template(this.model.toJSON()));
			return this;
    	}
	});

	window.Controllers.UserProfile = function(id) {
		this.model = new Models.User({id: id});
		this.view = new Views.UserProfile({
			model: this.model,
			el: "#user-profile-box"
		});
		this.view.render();
		this.model.fetch();
	};
})();

$(document).ready(function(){
	Backbone.history.start({pushState: true});
	if(window.location.hash == '#_=_') {
		Backbone.history.navigate(window.location.pathname, {trigger: false, replace: true});
    }

	window.userProfile = new Controllers.UserProfile(window.AuthData.ID);
});
