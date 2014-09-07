$(document).ready(function() {
	/* Dropdown helper */
	$("body").on("click", ".dropdown", function(evt) {
  		$(evt.currentTarget).toggleClass("open");
	});

	$("body").click(function(evt) {
		dropdown = $(".dropdown");

		if(!dropdown.is(evt.target) && dropdown.has(evt.target).length == 0) {
			dropdown.removeClass("open");
		}
	});
});
