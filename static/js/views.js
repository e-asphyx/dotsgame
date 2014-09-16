window.Views = window.Views || {};
window.Controllers = window.Controllers || {};

(function() {
	Views.UserProfile = Backbone.View.extend({
		initialize: function() {
			Utils.Proto(this, "template", _.template($("#user-profile-template").html())); /* DOM must be ready */
			this.listenTo(this.model, 'change', this.render);
		},

		render: function() {
			this.$el.html(this.template({o: this.model.toJSON()}));
			return this;
    	}
	});

	Controllers.UserProfile = function(id) {
		this.model = new Models.User({id: id});
		this.view = new Views.UserProfile({
			model: this.model,
			el: "#user-profile-box"
		});
		this.view.render();
		this.model.fetch();
	};
	
	Views.NumericIndicator = Backbone.View.extend({
		className: "button grow",

		initialize: function(options) {
			this.title = options.title || "";
			Utils.Proto(this, "template", _.template($("#numeric-indicator-template").html())); /* DOM must be ready */
			this.listenTo(this.model, 'change', this.render);
		},

		render: function() {
			this.$el.html(this.template({
				title: this.title,
				value: this.model.get("value") || 0
			}));
			return this;
    	}
	});

	Controllers.UserProfile = function(id) {
		this.model = new Models.User({id: id});
		this.view = new Views.UserProfile({
			model: this.model,
			el: "#user-profile-box"
		});
		this.view.render();
		this.model.fetch();
	};

	Views.ColorPicker = Backbone.View.extend({
		className: "modal",

		initialize: function(options) {
			this.schemes = options.schemes || [];
			Utils.Proto(this, "template", _.template($("#scheme-picker-template").html()));

			this.listenTo(this.collection, "change", this.render);
			this.listenTo(this.collection, "add", this.render);
			this.listenTo(this.collection, "remove", this.render);
		},

		render: function() {
			this.$el.html(this.template({items: this.collection.toJSON(), schemes: this.schemes}));
			return this;
    	}
	});

	$(document).ready(function(){
		if(window.location.hash && window.location.hash == '#_=_') {
			if(window.history && history.pushState) {
				window.history.pushState("", document.title, window.location.pathname);
			} else {
				// Prevent scrolling by storing the page's current scroll offset
				var scroll = {
					top: document.body.scrollTop,
					left: document.body.scrollLeft
				};
				window.location.hash = '';
				// Restore the scroll offset, should be flicker free
				document.body.scrollTop = scroll.top;
				document.body.scrollLeft = scroll.left;
			}
		}
		
		window.userProfile = new Controllers.UserProfile(window.AuthData.ID);
	});
})();
