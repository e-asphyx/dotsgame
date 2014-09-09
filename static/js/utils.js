$(document).ready(function() {
	/* Dropdown helper */
	$("body").on("click", ".dropdown", function(evt) {
		var container = $(".dropdown-container");

		if(!container.is(evt.target) && container.has(evt.target).length == 0) {
   			evt.preventDefault();

			var opened = $(evt.currentTarget).hasClass("open");
			$(".dropdown").removeClass("open"); /* close others */
			$(evt.currentTarget).toggleClass("open", !opened);
		}
	});

	$("body").click(function(evt) {
		var dropdown = $(".dropdown");

		if(!dropdown.is(evt.target) && dropdown.has(evt.target).length == 0) {
			dropdown.removeClass("open");
		}
	});
});
