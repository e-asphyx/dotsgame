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
